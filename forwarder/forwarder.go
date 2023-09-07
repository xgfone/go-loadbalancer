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

// Package forwarder provides a loadbalance forwarder.
package forwarder

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/xgfone/go-atomicvalue"
	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/balancer"
)

// Forwarder is used to forward the request to one of the backend endpoints.
type Forwarder struct {
	name      string
	timeout   int64
	balancer  atomicvalue.Value[balancer.Balancer]
	discovery atomicvalue.Value[loadbalancer.Discovery]
}

// New returns a new Forwarder to forward the request.
func New(name string, b balancer.Balancer, d loadbalancer.Discovery) *Forwarder {
	if b == nil {
		b = balancer.DefaultBalancer
	}
	if d == nil {
		panic("Forwarder.New: the endpoint discovery must not be nil")
	}

	f := &Forwarder{name: name}
	f.discovery.Store(d)
	f.balancer.Store(b)
	return f
}

// Name reutrns the name of the forwarder.
func (f *Forwarder) Name() string { return f.name }

// Type reutrns the type of the forwarder, that's "loadbalancer".
func (f *Forwarder) Type() string { return "loadbalancer" }

// GetTimeout returns the maximum timeout.
func (f *Forwarder) GetTimeout() time.Duration {
	return time.Duration(atomic.LoadInt64(&f.timeout))
}

// SetTimeout sets the maximum timeout.
func (f *Forwarder) SetTimeout(timeout time.Duration) {
	atomic.StoreInt64(&f.timeout, int64(timeout))
}

// GetBalancer returns the balancer.
func (f *Forwarder) GetBalancer() balancer.Balancer {
	return f.balancer.Load()
}

func (f *Forwarder) SetBalancer(b balancer.Balancer) {
	if b == nil {
		panic("Forwarder.SetBalancer: balancer must not be nil")
	}
	f.balancer.Store(b)
}

// SwapBalancer swaps the old balancer with the new.
func (f *Forwarder) SwapBalancer(new balancer.Balancer) (old balancer.Balancer) {
	if new == nil {
		panic("Forwarder.SwapBalancer: balancer must not be nil")
	}
	return f.balancer.Swap(new)
}

// GetDiscovery returns the endpoint discovery.
func (f *Forwarder) GetDiscovery() loadbalancer.Discovery {
	return f.discovery.Load()
}

// SetDiscovery sets the endpoint discovery to discover the endpoints.
func (f *Forwarder) SetDiscovery(d loadbalancer.Discovery) {
	if d == nil {
		panic("Forwarder.SetDiscovery: endpoint discovery must not be nil")
	}
	f.discovery.Store(d)
}

// SwapDiscovery swaps the old endpoint discovery with the new.
func (f *Forwarder) SwapDiscovery(new loadbalancer.Discovery) (old loadbalancer.Discovery) {
	if new == nil {
		panic("Forwarder.SwapDiscovery: endpoint discovery must not be nil")
	}
	return f.discovery.Swap(new)
}

// Serve implement the interface endpoint.Endpoint#Serve,
// which will forward the request to one of the backend endpoints.
func (f *Forwarder) Serve(ctx context.Context, req any) (any, error) {
	eps := f.Discover()
	if eps.Len() <= 0 {
		return nil, loadbalancer.ErrNoAvailableEndpoints
	}

	if timeout := f.GetTimeout(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return f.GetBalancer().Forward(ctx, req, eps)
}

var _ loadbalancer.LoadBalancer = new(Forwarder)

// ------------------------------------------------------------------------ //

var _ loadbalancer.Discovery = new(Forwarder)

// Discover return all the online endpoints,
// which implements the inerface endpoint.Discovery#Discover
func (f *Forwarder) Discover() *loadbalancer.Static { return f.GetDiscovery().Discover() }
