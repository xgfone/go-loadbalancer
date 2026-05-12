// Copyright 2026 xgfone
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

// Package roundrobin provides a selector based on the roundrobin.
package roundrobin

import (
	"context"
	"math"
	"sync"
	"sync/atomic"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
)

// Selector implements the selector based on the roundrobin policy.
type Selector struct {
	policy string
	last   uint64
}

// NewSelector returns a new Selector based on the roundrobin
// with the policy name.
//
// If policy is empty, use "roundrobin" instead.
func NewSelector(policy string) *Selector {
	if policy == "" {
		policy = "roundrobin"
	}
	return &Selector{policy: policy, last: math.MaxUint64}
}

// Policy returns the policy of the selector.
func (b *Selector) Policy() string { return b.policy }

// Select selects one of the backend endpoints based on the roundrobin policy.
func (b *Selector) Select(c context.Context, _ any, eps *loadbalancer.Static) (loadbalancer.Endpoint, error) {
	switch _len := len(eps.Endpoints); _len {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0], nil
	default:
		pos := atomic.AddUint64(&b.last, 1)
		return eps.Endpoints[pos%uint64(_len)], nil
	}
}

// WeightedSelector implements the Selector based on the weighted roundrobin.
type WeightedSelector struct {
	policy string

	count  int
	caches map[string]*weightedEndpoint
	lock   sync.Mutex
}

// NewWeightedSelector returns a new Selector based on the weighted roundrobin
// with the policy name.
//
// If policy is empty, use "weight_roundrobin" instead.
func NewWeightedSelector(policy string) *WeightedSelector {
	if policy == "" {
		policy = "weight_roundrobin"
	}

	return &WeightedSelector{
		policy: policy,
		caches: make(map[string]*weightedEndpoint, 16),
	}
}

// Policy returns the policy of the selector.
func (b *WeightedSelector) Policy() string { return b.policy }

// Select selects one of the backend endpoints based on the weighted roundrobin policy.
func (b *WeightedSelector) Select(c context.Context, r any, eps *loadbalancer.Static) (loadbalancer.Endpoint, error) {
	switch len(eps.Endpoints) {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0], nil
	default:
		return b.selectNextEndpoint(eps.Endpoints), nil
	}
}

func (b *WeightedSelector) selectNextEndpoint(eps loadbalancer.Endpoints) loadbalancer.Endpoint {
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
	loadbalancer.Endpoint
	CurrentWeight int
}
