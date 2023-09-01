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

// Endpoint represents a backend endpoint.
type Endpoint interface {
	ID() string

	Serve(ctx context.Context, req interface{}) (resp interface{}, err error)
}

// Unwrapper is used to unwrap the inner endpoint.
type Unwrapper interface {
	Unwrap() Endpoint
}

// Discovery is used to discover the endpoints.
type Discovery interface {
	Onlines() Endpoints
	Onlen() int
}

// Static is used to wrap the endpoints to avoid the memory allocation.
type Static struct{ Endpoints }

// Onlines implements the interface Discovery#Onlines.
//
// If s is equal to nil, return nil.
func (s *Static) Onlines() Endpoints {
	if s == nil {
		return nil
	}
	return s.Endpoints
}

// Onlen implements the interface Discovery#Onlen.
//
// If s is equal to nil, return 0.
func (s *Static) Onlen() int {
	if s == nil {
		return 0
	}
	return len(s.Endpoints)
}

var (
	_ Discovery = Endpoints(nil)
	_ Discovery = new(Static)
)

// Endpoints represents a group of the endpoints.
type Endpoints []Endpoint

// Onlen return the number of all the endpoints,
// which implements the interface Discovery#Onlen,
func (eps Endpoints) Onlen() (n int) { return len(eps) }

// Onlines returns itself, which implements the interface Discovery#Onlines.
func (eps Endpoints) Onlines() Endpoints { return eps }

// Contains reports whether the endpoints contains the endpoint indicated by the id.
func (eps Endpoints) Contains(epid string) bool {
	for _, s := range eps {
		if s.ID() == epid {
			return true
		}
	}
	return false
}

// Sort sorts the endpoints by the ASC order..
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

// WeightEndpoint represents a backend endpoint with the weight.
type WeightEndpoint interface {
	// Weight returns the weight of the endpoint, which must be a positive integer.
	//
	// The bigger the value, the higher the weight.
	Weight() int

	Endpoint
}

// GetWeight returns the weight of the endpoint if it has implements
// the interface WeightEndpoint. Or, check whether it has implemented
// the interface Unwrapper and unwrap it.
// If still failing, return 1 instead.
func GetWeight(ep Endpoint) int {
	switch s := ep.(type) {
	case WeightEndpoint:
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

// ServeFunc is the endpoint serve function.
type ServeFunc func(context.Context, interface{}) (interface{}, error)

func noop(ctx context.Context, i interface{}) (interface{}, error) { return nil, nil }

// Noop returns a new weight endpoint that does nothing.
func Noop(id string, weight int) WeightEndpoint { return New(id, weight, noop) }

// New returns a new weight endpoint with the id and serve function.
func New(id string, weight int, serve ServeFunc) WeightEndpoint {
	if weight < 1 {
		weight = 1
	}
	return &endpoint{sf: serve, id: id, w: weight}
}

type endpoint struct {
	sf ServeFunc
	id string
	w  int
}

func (ep *endpoint) ID() string                                                  { return ep.id }
func (ep *endpoint) Weight() int                                                 { return ep.w }
func (ep *endpoint) String() string                                              { return ep.id }
func (ep *endpoint) Serve(c context.Context, r interface{}) (interface{}, error) { return ep.sf(c, r) }
