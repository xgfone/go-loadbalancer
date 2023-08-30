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

package extep

import (
	"context"
	"sync/atomic"

	"github.com/xgfone/go-loadbalancer/endpoint"
)

// State is the runtime state of an endpoint, which can be inlined
// into other struct.
type State struct {
	Total   uint64 // The total number to handle all the requests.
	Failure uint64 // The total number to fail to handle the requests.
	Current uint64 // The number of the requests that are being handled.

	// For the extra runtime information.
	Extra interface{} `json:",omitempty" xml:",omitempty"`
}

// IncFailure increases the failure state.
func (rs *State) IncFailure() {
	atomic.AddUint64(&rs.Failure, 1)
}

// Inc increases the total and current state.
func (rs *State) Inc() {
	atomic.AddUint64(&rs.Total, 1)
	atomic.AddUint64(&rs.Current, 1)
}

// Dec decreases the current state.
func (rs *State) Dec() {
	atomic.AddUint64(&rs.Current, ^uint64(0))
}

// Clone clones itself to a new one.
//
// If Extra has implemented the interface { Clone() interface{} }, call it//
// to clone the field Extra.
func (rs *State) Clone() State {
	extra := rs.Extra
	if clone, ok := rs.Extra.(interface{ Clone() interface{} }); ok {
		extra = clone.Clone()
	}

	return State{
		Extra:   extra,
		Total:   atomic.LoadUint64(&rs.Total),
		Failure: atomic.LoadUint64(&rs.Failure),
		Current: atomic.LoadUint64(&rs.Current),
	}
}

// GetState returns the state of the endpoint.
//
// Return ZERO if ep has not implemented the interface StateEndpoint.
func GetState(ep endpoint.Endpoint) State {
	switch s := ep.(type) {
	case StateEndpoint:
		return s.State()
	case endpoint.Unwrapper:
		return GetState(s.Unwrap())
	default:
		return State{}
	}
}

// StateEndpoint is an extended endpoint supporting the state.
type StateEndpoint interface {
	endpoint.Endpoint
	State() State
}

// NewStateEndpoint returns a new state endpoint.
func NewStateEndpoint(ep endpoint.Endpoint) StateEndpoint {
	return &stateep{Endpoint: ep}
}

type stateep struct {
	endpoint.Endpoint
	state State
}

func (s *stateep) State() State              { return s.state.Clone() }
func (s *stateep) Unwrap() endpoint.Endpoint { return s.Endpoint }
func (s *stateep) Serve(c context.Context, r interface{}) (rp interface{}, e error) {
	s.state.Inc()
	defer s.state.Dec()
	if rp, e = s.Endpoint.Serve(c, r); e != nil {
		s.state.IncFailure()
	}
	return
}
