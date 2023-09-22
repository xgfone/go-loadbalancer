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

import (
	"cmp"
	"slices"
)

// SortEndpoints is used to sort a set of the endpoints.
//
// For the default implementation, sort them by the id from small to big.
var SortEndpoints func(Endpoints) = sortEndpoints

// Endpoint represents a backend endpoint.
type Endpoint interface {
	LoadBalancer
	ID() string
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

func sortEndpoints(eps Endpoints) {
	slices.SortStableFunc(eps, func(a, b Endpoint) int {
		return cmp.Compare(a.ID(), b.ID())
	})
}
