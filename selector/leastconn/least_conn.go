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

// Package leastconn provides a selector based on the least connections.
package leastconn

import (
	"context"
	"slices"
	"sync"

	"github.com/xgfone/go-loadbalancer"
)

// Concurrenter is used to return the concurrent number of an endpoint.
type Concurrenter interface {
	Concurrent() int
}

// GetConcurrency is the default function to get the concurrency number of the endpoint.
//
// For the default implementation, check whether the endpoint has implemented
// the interface Concurrenter and call it. Or, return 0.
var GetConcurrency func(loadbalancer.Endpoint) int = func(e loadbalancer.Endpoint) int {
	if c, ok := e.(Concurrenter); ok {
		return c.Concurrent()
	}
	return 0
}

// Selector implements the selector based on the least number of the connection.
type Selector struct {
	policy string
	getcon func(loadbalancer.Endpoint) int

	lock sync.Mutex
}

// NewSelector returns a new selector based on the random with the policy name.
//
// If policy is empty, use "leastconn" instead.
// If getconcurrency is nil, use GetConcurrency instead.
func NewSelector(policy string, getconcurrency func(loadbalancer.Endpoint) int) *Selector {
	if policy == "" {
		policy = "leastconn"
	}
	return &Selector{policy: policy, getcon: getconcurrency}
}

// Policy returns the policy of the selector.
func (b *Selector) Policy() string { return b.policy }

// Select selects one of the backend endpoints based on the least number of connections.
func (b *Selector) Select(c context.Context, _ any, eps *loadbalancer.Static) (loadbalancer.Endpoint, error) {
	switch len(eps.Endpoints) {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0], nil
	default:
		return b.selectEndpoint(eps.Endpoints), nil
	}
}

func (b *Selector) selectEndpoint(eps loadbalancer.Endpoints) loadbalancer.Endpoint {
	b.lock.Lock()
	defer b.lock.Unlock()
	w := loadbalancer.Acquire(len(eps))
	w.Endpoints = append(w.Endpoints, eps...)
	b.sort(w.Endpoints)
	ep := w.Endpoints[0]
	loadbalancer.Release(w)
	return ep
}

func (b *Selector) sort(eps loadbalancer.Endpoints) {
	get := b.getcon
	if get == nil {
		get = GetConcurrency
	}
	slices.SortFunc(eps, func(a, b loadbalancer.Endpoint) int { return get(a) - get(b) })
}
