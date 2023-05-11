// Copyright 2021~2023 xgfone
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

// Package balancer provides a balancer interface and builder,
// which is used to forward the request to one of the backend endpoints
// by the specific policy.
package balancer

import (
	"context"

	"github.com/xgfone/go-loadbalancer/balancer/roundrobin"
	"github.com/xgfone/go-loadbalancer/endpoint"
)

// DefaultBalancer is the default balancer.
var DefaultBalancer Balancer = roundrobin.NewWeightedBalancer("")

// Forwarder is used to forward the request to one of the backend endpoints.
type Forwarder interface {
	Forward(ctx context.Context, req interface{}, sd endpoint.Discovery) error
}

// Balancer does a balancer to forward the request to one of the backend
// endpoints by the specific policy.
type Balancer interface {
	Policy() string
	Forwarder
}
