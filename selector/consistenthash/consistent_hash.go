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

// Package consistenthash provides a selector based on the consistent hash.
package consistenthash

import (
	"context"

	"github.com/xgfone/go-loadbalancer"
)

// Selector implements the selector based on the consistent hash.
type Selector struct {
	policy string
	hash   func(req any) int
}

// NewSelector returns a new selector based on the consistent hash
// with the policy name.
func NewSelector(policy string, hash func(req any) int) *Selector {
	if policy == "" {
		panic("consistenthash: policy name must not be empty")
	}
	if hash == nil {
		panic("consistenthash: hash function must be nil")
	}
	return &Selector{policy: policy, hash: hash}
}

// Policy returns the policy of the selector.
func (b *Selector) Policy() string { return b.policy }

// Select selects one of the backend endpoints based on the consistent hash.
func (b *Selector) Select(c context.Context, r any, eps *loadbalancer.Static) (loadbalancer.Endpoint, error) {
	switch _len := len(eps.Endpoints); _len {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0], nil
	default:
		return eps.Endpoints[b.hash(r)%_len], nil
	}
}
