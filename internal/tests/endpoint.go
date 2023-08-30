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

	"github.com/xgfone/go-loadbalancer/endpoint"
)

var _ endpoint.Endpoint = new(TestEndpoint)

type TestEndpoint struct {
	ip      string
	state   endpoint.State
	weight  int
	current uint64

	endpoint.StatusManager
}

func (ep *TestEndpoint) String() string                          { return ep.ip }
func (ep *TestEndpoint) Weight() int                             { return ep.weight }
func (ep *TestEndpoint) ID() string                              { return ep.ip }
func (ep *TestEndpoint) Info() interface{}                       { return nil }
func (ep *TestEndpoint) Update(info interface{}) error           { return nil }
func (ep *TestEndpoint) Check(context.Context, interface{}) bool { return true }
func (ep *TestEndpoint) State() endpoint.State {
	state := ep.state.Clone()
	if ep.current > 0 {
		state.Current = ep.current
	}
	return state
}
func (ep *TestEndpoint) Serve(context.Context, interface{}) (interface{}, error) {
	ep.state.IncSuccess()
	ep.state.Inc()
	ep.state.Dec()
	return nil, nil
}

// NewEndpoint returns a new endpoint to do the test.
func NewEndpoint(ip string, weight int) *TestEndpoint {
	ep := &TestEndpoint{ip: ip, weight: weight, current: uint64(weight)}
	ep.SetStatus(endpoint.StatusOnline)
	return ep
}
