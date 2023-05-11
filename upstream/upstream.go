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

// Package upstream provides a common simple upstream.
package upstream

import (
	"time"

	"github.com/xgfone/go-atomicvalue"
	"github.com/xgfone/go-loadbalancer/balancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/forwarder"
)

// Option is used to configure the upstream.
type Option func(*Upstream)

// SetContextData returns an upstream option to set the context data.
func SetContextData(contextData interface{}) Option {
	return func(u *Upstream) { u.context.Store(contextData) }
}

// SetBalancer returns an upstream option to set the balancer.
func SetBalancer(balancer balancer.Balancer) Option {
	if balancer == nil {
		panic("Upstream: balancer must not be nil")
	}
	return func(u *Upstream) { u.forwarder.SwapBalancer(balancer) }
}

func SetTimeout(timeout time.Duration) Option {
	if timeout < 0 {
		panic("Upstream: timeout must not be negative")
	}
	return func(u *Upstream) { u.forwarder.SetTimeout(timeout) }
}

// SetDiscovery sets the endpoint discovery.
func SetDiscovery(discovery endpoint.Discovery) Option {
	if discovery == nil {
		panic("Upstream: the endpoint discovery must not be nil")
	}
	return func(u *Upstream) { u.forwarder.SwapEndpointDiscovery(discovery) }
}

// Upstream represents an upstream to manage the backend endpoints.
type Upstream struct {
	forwarder *forwarder.Forwarder
	context   atomicvalue.Value[interface{}]
}

// NewUpstream returns a new upstream with balancer.DefaultBalancer.
func NewUpstream(name string, options ...Option) *Upstream {
	if name == "" {
		panic("Upstream: name must not be empty")
	}

	up := &Upstream{forwarder: forwarder.NewForwarder(name, balancer.DefaultBalancer)}
	up.Update(options...)
	return up
}

// Name returns the name of the upstream.
func (up *Upstream) Name() string { return up.forwarder.Name() }

// Policy returns the forwarding policy of the upstream.
func (up *Upstream) Policy() string { return up.forwarder.GetBalancer().Policy() }

// ContextData returns the context data of the upstream.
func (up *Upstream) ContextData() interface{} { return up.context.Load() }

// Endpoints returns the endpoint discovery.
func (up *Upstream) Discovery() endpoint.Discovery { return up.forwarder.GetEndpointDiscovery() }

// Timeout returns the timeout.
func (up *Upstream) Timeout() time.Duration { return up.forwarder.GetTimeout() }

// Forwader returns the inner forwarder.
func (up *Upstream) Forwader() *forwarder.Forwarder { return up.forwarder }

// Options returns the options of the upstream.
func (up *Upstream) Options() []Option {
	return []Option{
		SetTimeout(up.Timeout()),
		SetDiscovery(up.Discovery()),
		SetBalancer(up.forwarder.GetBalancer()),
		SetContextData(up.ContextData()),
	}
}

// Update updates the upstream with the options.
func (up *Upstream) Update(options ...Option) {
	for _, option := range options {
		option(up)
	}
}
