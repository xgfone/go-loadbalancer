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

// Package loadbalancer provides some common functions.
package loadbalancer

import "context"

// LoadBalancer is a load balancer to serve the request.
type LoadBalancer interface {
	Serve(ctx context.Context, req any) (resp any, err error)
}

// ServeFunc is the loadbalancer serve function.
type ServeFunc func(ctx context.Context, req any) (resp any, err error)

// Serve implements the interface LoadBalancer.
func (f ServeFunc) Serve(ctx context.Context, req any) (any, error) {
	return f(ctx, req)
}
