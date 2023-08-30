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
	"sort"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
)

// Balancer implements the balancer based on the least number of the connection.
type Balancer struct{ policy string }

// NewBalancer returns a new balancer based on the random with the policy name.
//
// If policy is empty, use "least_conn" instead.
func NewBalancer(policy string) *Balancer {
	if policy == "" {
		policy = "least_conn"
	}
	return &Balancer{policy: policy}
}

// Policy returns the policy of the balancer.
func (b *Balancer) Policy() string { return b.policy }

// Forward forwards the request to one of the backend endpoints.
func (b *Balancer) Forward(c context.Context, r interface{}, sd endpoint.Discovery) (interface{}, error) {
	switch eps := sd.Endpoints(); len(eps) {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps[0].Serve(c, r)
	default:
		w := endpoint.Acquire(len(eps))
		w.Endpoints = append(w.Endpoints, eps...)
		sort.Stable(leastConnEndpoints(w.Endpoints))
		ep := w.Endpoints[0]
		endpoint.Release(w)
		return ep.Serve(c, r)
	}
}

type leastConnEndpoints endpoint.Endpoints

func (eps leastConnEndpoints) Len() int      { return len(eps) }
func (eps leastConnEndpoints) Swap(i, j int) { eps[i], eps[j] = eps[j], eps[i] }
func (eps leastConnEndpoints) Less(i, j int) bool {
	return eps[i].State().Current < eps[j].State().Current
}
