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

package loadbalancer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// ErrNoAvailableEndpoints is used to represents no available endpoints.
var ErrNoAvailableEndpoints = errors.New("no available endpoints")

// Pre-define some endpoint statuses.
const (
	EndpointStatusOnline  EndpointStatus = "on"
	EndpointStatusOffline EndpointStatus = "off"
)

// EndpointStatus represents the endpoint status.
type EndpointStatus string

// IsOnline reports whether the endpoint status is online.
func (s EndpointStatus) IsOnline() bool { return s == EndpointStatusOnline }

// IsOffline reports whether the endpoint status is offline.
func (s EndpointStatus) IsOffline() bool { return s == EndpointStatusOffline }

// EndpointState is the runtime state of an endpoint.
type EndpointState struct {
	Total   uint64 // The total number to handle all the requests.
	Success uint64 // The total number to handle the requests successfully.
	Current uint64 // The number of the requests that are being handled.

	// For the extra runtime information.
	Extra interface{} `json:",omitempty" xml:",omitempty"`
}

// IncSuccess increases the success state.
func (rs *EndpointState) IncSuccess() {
	atomic.AddUint64(&rs.Success, 1)
}

// Inc increases the total and current state.
func (rs *EndpointState) Inc() {
	atomic.AddUint64(&rs.Total, 1)
	atomic.AddUint64(&rs.Current, 1)
}

// Dec decreases the current state.
func (rs *EndpointState) Dec() {
	atomic.AddUint64(&rs.Current, ^uint64(0))
}

// Clone clones itself to a new one.
//
// If Extra has implemented the interface { Clone() interface{} }, call it//
// to clone the field Extra.
func (rs *EndpointState) Clone() EndpointState {
	extra := rs.Extra
	if clone, ok := rs.Extra.(interface{ Clone() interface{} }); ok {
		extra = clone.Clone()
	}

	return EndpointState{
		Extra:   extra,
		Total:   atomic.LoadUint64(&rs.Total),
		Success: atomic.LoadUint64(&rs.Success),
		Current: atomic.LoadUint64(&rs.Current),
	}
}

// Endpoint represents an backend endpoint.
type Endpoint interface {
	// Static information
	ID() string
	Type() string
	Info() interface{}

	// Dynamic information
	State() EndpointState
	Status() EndpointStatus

	// Handler
	Update(info interface{}) error
	Serve(ctx context.Context, req interface{}) error
	Check(ctx context.Context, req interface{}) (ok bool)
}

// EndpointWrapper is a wrapper wrapping the backend endpoint.
type EndpointWrapper interface {
	Unwrap() Endpoint
}

// WeightedEndpoint represents an backend endpoint with the weight.
type WeightedEndpoint interface {
	// Weight returns the weight of the endpoint, which must be a positive integer.
	//
	// The bigger the value, the higher the weight.
	Weight() int

	Endpoint
}

// EndpointDiscovery is used to discover the endpoints.
type EndpointDiscovery interface {
	AllEndpoints() Endpoints
	OffEndpoints() Endpoints
	OnEndpoints() Endpoints
	OnlineNum() int
}

var _ EndpointDiscovery = Endpoints(nil)

// Endpoints represents a group of the endpoints.
type Endpoints []Endpoint

// OnlineNum implements the interface EndpointDiscovery#OnlineNum
// to return the number of all the online endpoints.
func (eps Endpoints) OnlineNum() int { return len(eps.OnEndpoints()) }

// OnEndpoints implements the interface EndpointDiscovery#OnEndpoints
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

// OffEndpoints implements the interface EndpointDiscovery#OffEndpoints
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

// AllEndpoints implements the interface EndpointDiscovery#AllEndpoints
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
	iw, jw := GetEndpointWeight(eps[i]), GetEndpointWeight(eps[j])
	if iw < jw {
		return true
	} else if iw == jw {
		return eps[i].ID() < eps[j].ID()
	} else {
		return false
	}
}

// GetEndpointWeight returns the weight of the endpoint if it has implements
// the interface WeightedEndpoint. Or, check whether it has implemented
// the interface EndpointWrapper and unwrap it.
// If still failing, return 1 instead.
func GetEndpointWeight(ep Endpoint) int {
	switch s := ep.(type) {
	case WeightedEndpoint:
		return s.Weight()

	case EndpointWrapper:
		return GetEndpointWeight(s.Unwrap())

	default:
		return 1
	}
}

var (
	eppool4   = sync.Pool{New: func() any { return make(Endpoints, 0, 4) }}
	eppool8   = sync.Pool{New: func() any { return make(Endpoints, 0, 8) }}
	eppool16  = sync.Pool{New: func() any { return make(Endpoints, 0, 16) }}
	eppool32  = sync.Pool{New: func() any { return make(Endpoints, 0, 32) }}
	eppool64  = sync.Pool{New: func() any { return make(Endpoints, 0, 64) }}
	eppool128 = sync.Pool{New: func() any { return make(Endpoints, 0, 128) }}
)

// AcquireEndpoints acquires a preallocated zero-length endpoints from the pool.
func AcquireEndpoints(expectedMaxCap int) Endpoints {
	switch {
	case expectedMaxCap <= 4:
		return eppool4.Get().(Endpoints)

	case expectedMaxCap <= 8:
		return eppool8.Get().(Endpoints)

	case expectedMaxCap <= 16:
		return eppool16.Get().(Endpoints)

	case expectedMaxCap <= 32:
		return eppool32.Get().(Endpoints)

	case expectedMaxCap <= 64:
		return eppool64.Get().(Endpoints)

	default:
		return eppool128.Get().(Endpoints)
	}
}

// ReleaseEndpoints releases the endpoints back into the pool.
func ReleaseEndpoints(eps Endpoints) {
	for i, _len := 0, len(eps); i < _len; i++ {
		eps[i] = nil
	}

	eps = eps[:0]
	cap := cap(eps)
	switch {
	case cap < 8:
		eppool4.Put(eps)

	case cap < 16:
		eppool8.Put(eps)

	case cap < 32:
		eppool16.Put(eps)

	case cap < 64:
		eppool32.Put(eps)

	case cap < 128:
		eppool64.Put(eps)

	default:
		eppool128.Put(eps)
	}
}
