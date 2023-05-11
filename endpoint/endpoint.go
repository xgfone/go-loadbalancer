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

import "context"

// Endpoint represents an backend endpoint.
type Endpoint interface {
	// Static information
	ID() string
	Type() string
	Info() interface{}

	// Dynamic information
	State() State
	Status() Status
	SetStatus(Status)

	// Handler
	Update(info interface{}) error
	Serve(ctx context.Context, req interface{}) error
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

// Discovery is used to discover the endpoints.
type Discovery interface {
	AllEndpoints() Endpoints
	OffEndpoints() Endpoints
	OnEndpoints() Endpoints
	OnlineNum() int
}

var _ Discovery = Endpoints(nil)

// Endpoints represents a group of the endpoints.
type Endpoints []Endpoint

// OnlineNum implements the interface Discovery#OnlineNum
// to return the number of all the online endpoints.
func (eps Endpoints) OnlineNum() int { return len(eps.OnEndpoints()) }

// OnEndpoints implements the interface Discovery#OnEndpoints
// to return all the online endpoints.
func (eps Endpoints) OnEndpoints() Endpoints {
	var offline bool
	for _, s := range eps {
		if s.Status().IsOffline() {
			offline = true
			break
		}
	}
	if !offline {
		return eps
	}

	endpoints := make(Endpoints, 0, len(eps))
	for _, s := range eps {
		if s.Status().IsOnline() {
			endpoints = append(endpoints, s)
		}
	}
	return endpoints
}

// OffEndpoints implements the interface Discovery#OffEndpoints
// to return all the offline endpoints.
func (eps Endpoints) OffEndpoints() Endpoints {
	var online bool
	for _, s := range eps {
		if s.Status().IsOnline() {
			online = true
			break
		}
	}
	if !online {
		return eps
	}

	endpoints := make(Endpoints, 0, len(eps))
	for _, s := range eps {
		if s.Status().IsOffline() {
			endpoints = append(endpoints, s)
		}
	}
	return endpoints
}

// AllEndpoints implements the interface Discovery#AllEndpoints
// to return all the endpoints.
func (eps Endpoints) AllEndpoints() Endpoints { return eps }

// Contains reports whether the endpoints contains the endpoint indicated by the id.
func (eps Endpoints) Contains(endpointID string) bool {
	for _, s := range eps {
		if s.ID() == endpointID {
			return true
		}
	}
	return false
}

// Sort the endpoints by the ASC order.
func (eps Endpoints) Len() int      { return len(eps) }
func (eps Endpoints) Swap(i, j int) { eps[i], eps[j] = eps[j], eps[i] }
func (eps Endpoints) Less(i, j int) bool {
	iw, jw := GetWeight(eps[i]), GetWeight(eps[j])
	if iw < jw {
		return true
	} else if iw == jw {
		return eps[i].ID() < eps[j].ID()
	} else {
		return false
	}
}

// GetWeight returns the weight of the endpoint if it has implements
// the interface WeightedEndpoint. Or, check whether it has implemented
// the interface{ Unwrap() Endpoint } and unwrap it.
// If still failing, return 1 instead.
func GetWeight(ep Endpoint) int {
	switch s := ep.(type) {
	case WeightedEndpoint:
		return s.Weight()

	case interface{ Unwrap() Endpoint }:
		return GetWeight(s.Unwrap())

	default:
		return 1
	}
}
