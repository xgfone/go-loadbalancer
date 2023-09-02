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

// Package random provides a balancer based on the random.
package random

import (
	"context"
	"math/rand"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
)

var random = rand.Intn

// Balancer implements the balancer based on the random.
type Balancer struct{ balancer }

// NewBalancer returns a new balancer based on the random with the policy name.
//
// If policy is empty, use "random" instead.
func NewBalancer(policy string) *Balancer {
	return &Balancer{balancer: newBalancer(policy, "random")}
}

// Forward forwards the request to one of the backend endpoints.
func (b *Balancer) Forward(c context.Context, r interface{}, sd endpoint.Discovery) (interface{}, error) {
	eps := sd.Discover()
	switch _len := len(eps.Endpoints); _len {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0].Serve(c, r)
	default:
		return eps.Endpoints[random(_len)].Serve(c, r)
	}
}

// WeightedBalancer implements the balancer based on the weighted random.
type WeightedBalancer struct{ balancer }

// NewWeightedBalancer returns a new balancer based on the weighted random
// with the policy name.
//
// If policy is empty, use "weight_random" instead.
func NewWeightedBalancer(policy string) *WeightedBalancer {
	return &WeightedBalancer{balancer: newBalancer(policy, "weight_random")}
}

// Forward forwards the request to one of the backend endpoints.
func (b *WeightedBalancer) Forward(c context.Context, r interface{}, sd endpoint.Discovery) (interface{}, error) {
	eps := sd.Discover()
	_len := len(eps.Endpoints)
	switch _len {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0].Serve(c, r)
	}

	var total int
	for i := 0; i < _len; i++ {
		total += endpoint.GetWeight(eps.Endpoints[i])
	}

	pos := random(total)
	for {
		var total int
		for i := 0; i < _len; i++ {
			total += endpoint.GetWeight(eps.Endpoints[i])
			if pos <= total {
				return eps.Endpoints[i].Serve(c, r)
			}
		}
		pos %= total
	}
}

type balancer string

// Policy returns the policy of the balancer.
func (b balancer) Policy() string { return string(b) }

func newBalancer(policy, _default string) balancer {
	if policy == "" {
		policy = _default
	}
	return balancer(policy)
}
