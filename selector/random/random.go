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

// Package random provides a selector based on the random.
package random

import (
	"context"
	"math/rand/v2"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
)

// Selector implements the Selector based on the random.
type Selector struct{ selector }

// NewSelector returns a new Selector based on the random with the policy name.
//
// If policy is empty, use "random" instead.
func NewSelector(policy string) *Selector {
	return &Selector{selector: newSelector(policy, "random")}
}

// Select selects one of the backend endpoints based on the random.
func (b *Selector) Select(c context.Context, r any, eps *loadbalancer.Static) (loadbalancer.Endpoint, error) {
	switch _len := len(eps.Endpoints); _len {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0], nil
	default:
		return eps.Endpoints[rand.IntN(_len)], nil
	}
}

// WeightedSelector implements the Selector based on the weighted random.
type WeightedSelector struct{ selector }

// NewWeightedSelector returns a new WeightedSelector based on the weighted random
// with the policy name.
//
// If policy is empty, use "weight_random" instead.
func NewWeightedSelector(policy string) *WeightedSelector {
	return &WeightedSelector{selector: newSelector(policy, "weight_random")}
}

// Select selects one of the backend endpoints based on the weighted random.
func (b *WeightedSelector) Select(c context.Context, r any, eps *loadbalancer.Static) (loadbalancer.Endpoint, error) {
	_len := len(eps.Endpoints)
	switch _len {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0], nil
	}

	var total int
	for i := range _len {
		total += endpoint.GetWeight(eps.Endpoints[i])
	}

	pos := rand.IntN(total)
	for {
		var total int
		for i := range _len {
			total += endpoint.GetWeight(eps.Endpoints[i])
			if pos <= total {
				return eps.Endpoints[i], nil
			}
		}
		pos %= total
	}
}

type selector string

// Policy returns the policy of the selector.
func (b selector) Policy() string { return string(b) }

func newSelector(policy, _default string) selector {
	if policy == "" {
		policy = _default
	}
	return selector(policy)
}
