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
	"context"
	"net/http"

	"github.com/xgfone/go-atomicvalue"
	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/balancer"
	"github.com/xgfone/go-loadbalancer/forwarder"
	"github.com/xgfone/go-loadbalancer/healthcheck"
	"github.com/xgfone/go-loadbalancer/processor"
)

// Request represents a upstream request context.
type Request struct {
	RespBodyHandler processor.ResponseProcessor
	http.ResponseWriter
	*http.Request
}

// RequestID returns the request id.
func (r Request) RequestID() string { return r.Request.Header.Get("X-Request-Id") }

// RemoteAddr returns the address of the client.
func (r Request) RemoteAddr() string { return r.Request.RemoteAddr }

// GetRequest returns the request.
func (r Request) GetRequest() *http.Request { return r.Request }

// Option is used to configure the upstream.
type Option func(*Upstream)

// SetBalancer returns an upstream option to set the balancer.
func SetBalancer(balancer balancer.Balancer) Option {
	if balancer == nil {
		panic("upstream balancer must not be nil")
	}
	return func(u *Upstream) { u.forwarder.SwapBalancer(balancer) }
}

// SetHealthCheck returns an upstream option to set the healthcheck config
// of all the backend endpoints.
func SetHealthCheck(checker healthcheck.Checker) Option {
	return func(u *Upstream) { u.checkerconf.Store(checker) }
}

// SetEndpoints returns an upstream option to set all the endpoints.
func SetEndpoints(eps ...loadbalancer.Endpoint) Option {
	return func(u *Upstream) { u.healthcheck.ResetEndpoints(eps, u.Checker()) }
}

// Upstream represents an upstream to manage the backend endpoints.
type Upstream struct {
	forwarder   *forwarder.Forwarder
	healthcheck *healthcheck.HealthChecker
	checkerconf atomicvalue.Value[healthcheck.Checker]
}

// NewUpstream returns a new upstream.
func NewUpstream(name string, balancer balancer.Balancer, options ...Option) *Upstream {
	if name == "" {
		panic("the upstream name must not be empty")
	}
	forwarder := forwarder.NewForwarder(name, balancer)
	healthcheck := healthcheck.NewHealthChecker()
	healthcheck.AddUpdater(name, forwarder)

	up := &Upstream{forwarder: forwarder, healthcheck: healthcheck}
	up.Update(options...)
	return up
}

// Name returns the name of the upstream.
func (up *Upstream) Name() string { return up.forwarder.Name() }

// Policy returns the forwarding policy of the upstream.
func (up *Upstream) Policy() string { return up.forwarder.GetBalancer().Policy() }

// Checker returns the healthcheck configuration of the upstream.
func (up *Upstream) Checker() healthcheck.Checker { return up.checkerconf.Load() }

// Endpoints returns all the backend endpoints.
func (up *Upstream) Endpoints() loadbalancer.Endpoints { return up.forwarder.AllOriginEndpoints() }

// Forwader returns the inner forwarder.
func (up *Upstream) Forwader() forwarder.Forwarder { return *up.forwarder }

// Options returns the options of the upstream.
func (up *Upstream) Options() []Option {
	return []Option{
		SetBalancer(up.forwarder.GetBalancer()),
		SetHealthCheck(up.Checker()),
		SetEndpoints(up.Endpoints()...),
	}
}

// Update updates the upstream with the options.
func (up *Upstream) Update(options ...Option) {
	for _, option := range options {
		option(up)
	}
}

// Stop stops the inner healthcheck of the upstream.
func (up *Upstream) Stop() { up.healthcheck.Stop() }

// Start starts the inner healthcheck of the upstream.
func (up *Upstream) Start() { up.healthcheck.Start() }

// Forward forwards the request to one of the backend endpoints,
// which uses Request as the request context.
func (up *Upstream) Forward(c context.Context, w http.ResponseWriter, r *http.Request,
	respBodyHandler processor.ResponseProcessor) error {
	return up.forwarder.Serve(c, Request{
		RespBodyHandler: respBodyHandler,
		ResponseWriter:  w,
		Request:         r,
	})
}
