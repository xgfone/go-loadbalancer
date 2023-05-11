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
	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/healthcheck"
)

var (
	_ healthcheck.Updater = &Forwarder{}
	_ endpoint.Discovery  = &Forwarder{}
)

// Forwarder is used to forward the request to one of the backend endpoints.
type Forwarder struct {
	name      string
	timeout   int64
	balancer  atomicvalue.Value[balancer.Balancer]
	discovery atomicvalue.Value[endpoint.Discovery]
	epmanager *endpoint.Manager
}

// NewForwarder returns a new Forwarder to forward the request.
func NewForwarder(name string, balancer balancer.Balancer) *Forwarder {
	if balancer == nil {
		panic("the balancer is nil")
	}

	return &Forwarder{
		name:      name,
		balancer:  atomicvalue.NewValue(balancer),
		epmanager: endpoint.NewManager(),
	}
}

// Name reutrns the name of the forwarder.
func (f *Forwarder) Name() string { return f.name }

// Type reutrns the type of the forwarder, that's "loadbalancer".
func (f *Forwarder) Type() string { return "loadbalancer" }

// GetBalancer returns the balancer.
func (f *Forwarder) GetBalancer() balancer.Balancer {
	return f.balancer.Load()
}

// SwapBalancer swaps the old balancer with the new.
func (f *Forwarder) SwapBalancer(new balancer.Balancer) (old balancer.Balancer) {
	return f.balancer.Swap(new)
}

// GetTimeout returns the maximum timeout.
func (f *Forwarder) GetTimeout() time.Duration {
	return time.Duration(atomic.LoadInt64(&f.timeout))
}

// SetTimeout sets the maximum timeout.
func (f *Forwarder) SetTimeout(timeout time.Duration) {
	atomic.StoreInt64(&f.timeout, int64(timeout))
}

// GetEndpointDiscovery returns the endpoint discovery.
//
// If not set the endpoint discovery, return nil.
func (f *Forwarder) GetEndpointDiscovery() (ed endpoint.Discovery) {
	return f.discovery.Load()
}

// SwapEndpointDiscovery sets the endpoint discovery to discover the endpoints,
// and returns the old one.
//
// If sd is equal to nil, it will cancel the endpoint discovery.
// Or, use the endpoint discovery instead of the direct endpoints.
func (f *Forwarder) SwapEndpointDiscovery(new endpoint.Discovery) (old endpoint.Discovery) {
	old = f.discovery.Swap(new)
	// f.ResetEndpoints() // We need to clear all the endpoints??
	return
}

// Discovery returns the customized Discovery or the inner manager.
func (f *Forwarder) Discovery() endpoint.Discovery {
	if ed := f.GetEndpointDiscovery(); ed != nil {
		return ed
	}
	return f.epmanager
}

// EndpointManager returns the inner endpoint manager of the forwarder.
func (f *Forwarder) EndpointManager() *endpoint.Manager { return f.epmanager }

// Serve implement the interface endpoint.Endpoint#Serve,
// which will forward the request to one of the backend endpoints.
//
// req should have implemented one of the interfaces:
//
//	interface{ RemoteAddr() string }
//	interface{ RemoteAddr() net.IP }
//	interface{ RemoteAddr() net.Addr }
//	interface{ RemoteAddr() netip.Addr }
func (f *Forwarder) Serve(ctx context.Context, req interface{}) error {
	ed := f.GetEndpointDiscovery()
	if ed == nil {
		ed = f.epmanager
	}

	if ed.OnlineNum() <= 0 {
		return loadbalancer.ErrNoAvailableEndpoints
	}

	if timeout := f.GetTimeout(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return f.GetBalancer().Forward(ctx, req, ed)
}

// ------------------------------------------------------------------------ //

// SetEndpointOnline implements the interface healthcheck.Updater#SetEndpointOnline
// to set the status of the endpoint to endpoint.StatusOnline or endpoint.StatusOffline.
//
// It uses the inner endpoint management of the forwarder.
func (f *Forwarder) SetEndpointOnline(epid string, online bool) {
	if online {
		f.epmanager.SetEndpointStatus(epid, endpoint.StatusOnline)
	} else {
		f.epmanager.SetEndpointStatus(epid, endpoint.StatusOffline)
	}
}

// UpsertEndpoint implements the interface healthcheck.Updater#UpsertEndpoint
// to add or update the given endpoint.
//
// It uses the inner endpoint management of the forwarder.
func (f *Forwarder) UpsertEndpoint(ep endpoint.Endpoint) {
	f.epmanager.UpsertEndpoints(ep)
}

// RemoveEndpoint implements the interface healthcheck.Updater#RemoveEndpoint
// to remove the endpoint by the endpoint id, which does nothing
// if the endpoint does not exist.
//
// It uses the inner endpoint management of the forwarder.
func (f *Forwarder) RemoveEndpoint(epid string) {
	f.epmanager.RemoveEndpoint(epid)
}

// ------------------------------------------------------------------------ //

// OnlineNum implements the interfce endpoint.Discovery#OnlineNum
// to return the number of the online endpoints.
func (f *Forwarder) OnlineNum() int {
	return f.Discovery().OnlineNum()
}

// OnEndpoints implements the inerface endpoint.Discovery#OnEndpoints
// to only return all the online endpoints.
func (f *Forwarder) OnEndpoints() endpoint.Endpoints {
	return f.Discovery().OnEndpoints()
}

// OffEndpoints implements the inerface endpoint.Discovery#OffEndpoints
// to only return all the offline endpoints.
func (f *Forwarder) OffEndpoints() endpoint.Endpoints {
	return f.Discovery().OffEndpoints()
}

// AllEndpoints implements the inerface endpoint.Discovery#OnEndpoints
// to return all the endpoints.
func (f *Forwarder) AllEndpoints() endpoint.Endpoints {
	return f.Discovery().AllEndpoints()
}
