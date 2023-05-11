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

// Package consistenthash provides a balancer based on the consistent hash.
package consistenthash

import (
	"context"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
)

// Balancer implements the balancer based on the consistent hash.
type Balancer struct {
	policy string
	hash   func(req interface{}) int
}

// NewBalancer returns a new balancer based on the consistent hash
// with the policy name.
func NewBalancer(policy string, hash func(req interface{}) int) *Balancer {
	if policy == "" {
		panic("consistenthash: policy name must not be empty")
	}
	if hash == nil {
		panic("consistenthash: hash function must be nil")
	}
	return &Balancer{policy: policy, hash: hash}
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
		return eps[b.hash(r)%_len].Serve(c, r)
	}
}
