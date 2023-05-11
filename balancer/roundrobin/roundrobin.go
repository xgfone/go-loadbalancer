// Copyright 2023 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package roundrobin provides a balancer based on the roundrobin.
package roundrobin

import (
	"context"
	"math"
	"sync"
	"sync/atomic"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
)

// Balancer implements the balancer based on the roundrobin policy.
type Balancer struct {
	policy string
	last   uint64
}

// NewBalancer returns a new balancer based on the roundrobin
// with the policy name.
//
// If policy is empty, use "round_robin" instead.
func NewBalancer(policy string) *Balancer {
	if policy == "" {
		policy = "round_robin"
	}
	return &Balancer{policy: policy, last: math.MaxUint64}
}

// Policy returns the policy of the balancer.
func (b *Balancer) Policy() string { return b.policy }

// Forward forwards the request to one of the backend endpoints.
func (b *Balancer) Forward(c context.Context, r interface{}, sd endpoint.Discovery) error {
	eps := sd.OnEndpoints()
	switch _len := len(eps); _len {
	case 0:
		return loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps[0].Serve(c, r)
	default:
		pos := atomic.AddUint64(&b.last, 1)
		return eps[pos%uint64(_len)].Serve(c, r)
	}
}

// WeightedBalancer implements the balancer based on the weighted roundrobin.
type WeightedBalancer struct {
	policy string

	count  int
	caches map[string]*weightedEndpoint
	lock   sync.Mutex
}

// NewWeightedBalancer returns a new balancer based on the weighted roundrobin
// with the policy name.
//
// If policy is empty, use "weight_round_robin" instead.
func NewWeightedBalancer(policy string) *WeightedBalancer {
	if policy == "" {
		policy = "weight_round_robin"
	}

	return &WeightedBalancer{
		policy: policy,
		caches: make(map[string]*weightedEndpoint, 16),
	}
}

// Policy returns the policy of the balancer.
func (b *WeightedBalancer) Policy() string { return b.policy }

// Forward forwards the request to one of the backend endpoints.
func (b *WeightedBalancer) Forward(c context.Context, r interface{}, sd endpoint.Discovery) error {
	eps := sd.OnEndpoints()
	switch len(eps) {
	case 0:
		return loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps[0].Serve(c, r)
	default:
		return b.selectNextEndpoint(eps).Serve(c, r)
	}
}

func (b *WeightedBalancer) selectNextEndpoint(eps endpoint.Endpoints) endpoint.Endpoint {
	b.lock.Lock()
	defer b.lock.Unlock()

	var total int
	var selected *weightedEndpoint
	for i, _len := 0, len(eps); i < _len; i++ {
		weight := endpoint.GetWeight(eps[i])
		total += weight

		id := eps[i].ID()
		ws, ok := b.caches[id]
		if ok {
			ws.CurrentWeight += weight
		} else {
			ws = &weightedEndpoint{Endpoint: eps[i], CurrentWeight: weight}
			b.caches[id] = ws
		}

		if selected == nil || selected.CurrentWeight < ws.CurrentWeight {
			selected = ws
		}
	}

	// We clean the down endpoints only each 1000 times.
	if b.count++; b.count >= 1000 {
		for id := range b.caches {
			if !eps.Contains(id) {
				delete(b.caches, id)
			}
		}
		b.count = 0
	}

	selected.CurrentWeight -= total
	return selected.Endpoint
}

type weightedEndpoint struct {
	endpoint.Endpoint
	CurrentWeight int
}
