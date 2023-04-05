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

package tests

import (
	"context"

	"github.com/xgfone/go-loadbalancer"
)

type endpoint struct {
	ip      string
	state   loadbalancer.EndpointState
	weight  int
	current uint64
}

func (ep *endpoint) Weight() int                         { return ep.weight }
func (ep *endpoint) ID() string                          { return ep.ip }
func (ep *endpoint) Type() string                        { return "" }
func (ep *endpoint) Info() interface{}                   { return nil }
func (ep *endpoint) Check(context.Context) error         { return nil }
func (ep *endpoint) Update(info interface{}) error       { return nil }
func (ep *endpoint) Status() loadbalancer.EndpointStatus { return loadbalancer.EndpointStatusOnline }
func (ep *endpoint) State() loadbalancer.EndpointState {
	state := ep.state.Clone()
	if ep.current > 0 {
		state.Current = ep.current
	}
	return state
}
func (ep *endpoint) Serve(c context.Context, r interface{}) error {
	ep.state.IncSuccess()
	ep.state.Inc()
	ep.state.Dec()
	return nil
}

// NewEndpoint returns a new endpoint to do the test.
func NewEndpoint(ip string, weight int) loadbalancer.Endpoint {
	return &endpoint{ip: ip, weight: weight, current: uint64(weight)}
}
