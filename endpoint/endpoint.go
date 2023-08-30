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

package endpoint

import (
	"context"
	"sort"
)

// Endpoint represents an backend endpoint.
type Endpoint interface {
	// Static information
	ID() string
	Info() interface{}

	// Dynamic information
	State() State
	Status() Status
	SetStatus(Status)

	// Handler
	Update(info interface{}) error
	Serve(ctx context.Context, req interface{}) (interface{}, error)
	Check(ctx context.Context, req interface{}) (ok bool)
}

// WeightedEndpoint represents an backend endpoint with the weight.
type WeightedEndpoint interface {
	// Weight returns the weight of the endpoint, which must be a positive integer.
	//
	// The bigger the value, the higher the weight.
	Weight() int

	Endpoint
}

// Unwrapper is used to unwrap the inner endpoint.
type Unwrapper interface {
	Unwrap() Endpoint
}

// Discovery is used to discover the endpoints.
type Discovery interface {
	Endpoints() Endpoints
	Len() int
}

var _ Discovery = Endpoints(nil)

// Endpoints represents a group of the endpoints.
type Endpoints []Endpoint

// Len implements the interface Discovery#Len,
// which return the number of all the endpoints.
//
// NOTICE: it should be just used for test.
func (eps Endpoints) Len() (n int) { return len(eps.Endpoints()) }

// Endpoints implements the interface Discovery#Endpoints, which returns itself.
//
// NOTICE: it should be just used for test.
func (eps Endpoints) Endpoints() Endpoints { return eps }

// Contains reports whether the endpoints contains the endpoint indicated by the id.
func (eps Endpoints) Contains(endpointID string) bool {
	for _, s := range eps {
		if s.ID() == endpointID {
			return true
		}
	}
	return false
}

// Sort sorts the endpoints by the ASC order.
func Sort(eps Endpoints) {
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

// GetWeight returns the weight of the endpoint if it has implements
// the interface WeightedEndpoint. Or, check whether it has implemented
// the interface{ Unwrap() Endpoint } and unwrap it.
// If still failing, return 1 instead.
func GetWeight(ep Endpoint) int {
	switch s := ep.(type) {
	case WeightedEndpoint:
		if weight := s.Weight(); weight > 0 {
			return weight
		}
		return 1

	case interface{ Unwrap() Endpoint }:
		return GetWeight(s.Unwrap())

	default:
		return 1
	}
}
