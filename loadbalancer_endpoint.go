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

package loadbalancer

import "context"

// ServeFunc is the endpoint serve function.
type ServeFunc func(ctx context.Context, req interface{}) (resp interface{}, err error)

// Endpoint represents a backend endpoint.
type Endpoint interface {
	ID() string

	Serve(ctx context.Context, req interface{}) (resp interface{}, err error)
}

// Unwrapper is used to unwrap the inner endpoint.
type Unwrapper interface {
	Unwrap() Endpoint
}

// Endpoints represents a group of the endpoints.
type Endpoints []Endpoint

// Len return the number of all the endpoints,
func (eps Endpoints) Len() (n int) { return len(eps) }

// Contains reports whether the endpoints contains the endpoint indicated by the id.
func (eps Endpoints) Contains(epid string) bool {
	for _, s := range eps {
		if s.ID() == epid {
			return true
		}
	}
	return false
}
