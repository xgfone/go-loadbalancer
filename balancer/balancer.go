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

// Package balancer provides a balancer interface to forward the request
// to one of the backend endpoints by the specific policy.
package balancer

import (
	"context"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/selector"
	"github.com/xgfone/go-loadbalancer/selector/roundrobin"
)

// DefaultBalancer is the default balancer.
var DefaultBalancer Balancer = NewFromSelector(roundrobin.NewSelector(""))

// Forwarder is used to forward the request to one of the backend endpoints.
type Forwarder interface {
	Forward(ctx context.Context, req any, eps *loadbalancer.Static) (any, error)
}

// Balancer is used to forward the request to one of the backend endpoints
// by the specific policy.
type Balancer interface {
	Policy() string
	Forwarder
}

// NewFromSelector returns a balancer from the given selector.
func NewFromSelector(selector selector.Selector) Balancer {
	if selector == nil {
		panic("selector is nil")
	}
	return &selectorBalancer{Selector: selector}
}

type selectorBalancer struct {
	selector.Selector
}

func (b selectorBalancer) Forward(ctx context.Context, req any, eps *loadbalancer.Static) (any, error) {
	ep, err := b.Select(ctx, req, eps)
	if err != nil {
		return nil, err
	}
	return ep.Serve(ctx, req)
}
