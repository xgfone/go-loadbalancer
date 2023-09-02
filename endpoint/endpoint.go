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

// Package endpoint provides some auxiliary functions about endpoint.
package endpoint

import (
	"context"
	"sort"
	"sync/atomic"

	"github.com/xgfone/go-loadbalancer"
)

// Endpoint is a common endpoint implementation.
type Endpoint struct {
	id    string
	serve loadbalancer.ServeFunc

	total  uint64
	concur int32
	weight int32
}

// New returns a new common endpoint with id, weight and serve function.
//
// If weight is equal to or less than 1, use 1 instead.
func New(id string, weight int, serve loadbalancer.ServeFunc) *Endpoint {
	if id == "" {
		panic("endpoint.New: id must not be empty")
	}
	if serve == nil {
		panic("endpoint.New: serve function must not be nil")
	}
	if weight < 1 {
		weight = 1
	}

	e := &Endpoint{id: id, serve: serve}
	e.SetWeight(weight)
	return e
}

// Serve returns the endpoint id, which implements the interface loadbalancer.Endpoint#ID.
func (e *Endpoint) ID() string { return e.id }

// String returns the description of the endpoint, which is equal to ID.
func (e *Endpoint) String() string { return e.id }

// Serve serves the request, which implements the interface loadbalancer.Endpoint#Serve.
func (e *Endpoint) Serve(c context.Context, r interface{}) (interface{}, error) {
	e.inc()
	defer e.dec()
	return e.serve(c, r)
}

func (e *Endpoint) inc() {
	atomic.AddUint64(&e.total, 1)
	atomic.AddInt32(&e.concur, 1)
}

func (e *Endpoint) dec() { atomic.AddInt32(&e.concur, -1) }

// Total returns the total number that the endpoint has served the requests.
func (e *Endpoint) Total() int { return int(atomic.LoadUint64(&e.total)) }

// Concurrent returns the number that the endpoint is serving the requests concurrently.
func (e *Endpoint) Concurrent() int { return int(atomic.LoadInt32(&e.concur)) }

// Weight returns the weight of the endpoint,
// which implements the interface Weighter.
func (e *Endpoint) Weight() int { return int(atomic.LoadInt32(&e.weight)) }

// SetWeight resets the weight of the endpoint.
//
// NOTICE: weight must be a positive integer.
func (e *Endpoint) SetWeight(weight int) {
	if weight <= 0 {
		panic("Endpoint.SetWeight: weight must be a positive integer")
	}
	atomic.StoreInt32(&e.weight, int32(weight))
}

var _ Weighter = new(Endpoint)

// SortEndpoints sorts the endpoints by the ASC order..
func Sort(eps loadbalancer.Endpoints) {
	if len(eps) == 0 {
		return
	}

	sort.SliceStable(eps, func(i, j int) bool {
		iw, jw := GetWeight(eps[i]), GetWeight(eps[j])
		if iw < jw {
			return true
		} else if iw == jw {
			return eps[i].ID() < eps[j].ID()
		} else {
			return false
		}
	})
}
