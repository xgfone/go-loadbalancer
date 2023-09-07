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

// Package leastconn provides a balancer based on the least connections.
package leastconn

import (
	"context"
	"slices"
	"sync"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
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

// Balancer implements the balancer based on the least number of the connection.
type Balancer struct {
	policy string
	getcon func(loadbalancer.Endpoint) int

	lock sync.Mutex
}

// NewBalancer returns a new balancer based on the random with the policy name.
//
// If policy is empty, use "leastconn" instead.
// If getconcurrency is nil, use GetConcurrency instead.
func NewBalancer(policy string, getconcurrency func(loadbalancer.Endpoint) int) *Balancer {
	if policy == "" {
		policy = "leastconn"
	}
	return &Balancer{policy: policy, getcon: getconcurrency}
}

// Policy returns the policy of the balancer.
func (b *Balancer) Policy() string { return b.policy }

// Forward forwards the request to one of the backend endpoints.
func (b *Balancer) Forward(c context.Context, r any, sd endpoint.Discovery) (any, error) {
	switch eps := sd.Discover(); len(eps.Endpoints) {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0].Serve(c, r)
	default:
		return b.selectEndpoint(eps.Endpoints).Serve(c, r)
	}
}

func (b *Balancer) selectEndpoint(eps loadbalancer.Endpoints) loadbalancer.Endpoint {
	b.lock.Lock()
	defer b.lock.Unlock()
	w := endpoint.Acquire(len(eps))
	w.Endpoints = append(w.Endpoints, eps...)
	b.sort(w.Endpoints)
	ep := w.Endpoints[0]
	endpoint.Release(w)
	return ep
}

func (b *Balancer) sort(eps loadbalancer.Endpoints) {
	get := b.getcon
	if get == nil {
		get = GetConcurrency
	}
	slices.SortFunc(eps, func(a, b loadbalancer.Endpoint) int { return get(a) - get(b) })
}
