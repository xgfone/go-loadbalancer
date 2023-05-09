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
	"github.com/xgfone/go-loadbalancer/healthcheck"
)

var (
	_ healthcheck.Updater            = &Forwarder{}
	_ loadbalancer.EndpointDiscovery = &Forwarder{}
)

// Forwarder is used to forward the request to one of the backend endpoints.
type Forwarder struct {
	name      string
	timeout   int64
	balancer  atomicvalue.Value[balancer.Balancer]
	discovery atomicvalue.Value[loadbalancer.EndpointDiscovery]

	epmanager *endpointsManager
	httperror HTTPErrorHandler
}

// NewForwarder returns a new Forwarder to forward the request.
func NewForwarder(name string, balancer balancer.Balancer) *Forwarder {
	if balancer == nil {
		panic("the balancer is nil")
	}

	return &Forwarder{
		name:      name,
		balancer:  atomicvalue.NewValue(balancer),
		epmanager: newEndpointsManager(),
		httperror: handleHTTPError,
	}
}

// Name reutrns the name of the forwarder.
func (f *Forwarder) Name() string { return f.name }

// Type reutrns the type of the forwarder, that's loadbalancer.
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
func (f *Forwarder) GetEndpointDiscovery() (sd loadbalancer.EndpointDiscovery) {
	return f.discovery.Load()
}

// SwapEndpointDiscovery sets the endpoint discovery to discover the endpoints,
// and returns the old one.
//
// If sd is equal to nil, it will cancel the endpoint discovery.
// Or, use the endpoint discovery instead of the direct endpoints.
func (f *Forwarder) SwapEndpointDiscovery(new loadbalancer.EndpointDiscovery) (old loadbalancer.EndpointDiscovery) {
	old = f.discovery.Swap(new)
	// f.ResetEndpoints() // We need to clear all the endpoints??
	return
}

// Serve implement the interface loadbalancer.Endpoint#Serve,
// which will forward the request to one of the backend endpoints.
//
// req should have implemented one of the interfaces:
//
//	interface{ RemoteAddr() string }
//	interface{ RemoteAddr() net.IP }
//	interface{ RemoteAddr() net.Addr }
//	interface{ RemoteAddr() netip.Addr }
func (f *Forwarder) Serve(ctx context.Context, req interface{}) error {
	sd := f.GetEndpointDiscovery()
	if sd == nil {
		sd = f.epmanager
	}

	if sd.OnlineNum() <= 0 {
		return loadbalancer.ErrNoAvailableEndpoints
	}

	if timeout := f.GetTimeout(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return f.GetBalancer().Forward(ctx, req, sd)
}

// SetEndpointOnline implements the interface healthcheck.Updater#SetEndpointOnline
// to set the status of the endpoint to EndpointStatusOnline or EndpointStatusOffline.
func (f *Forwarder) SetEndpointOnline(epid string, online bool) {
	if online {
		f.SetEndpointStatus(epid, loadbalancer.EndpointStatusOnline)
	} else {
		f.SetEndpointStatus(epid, loadbalancer.EndpointStatusOffline)
	}
}

// SetEndpointStatus sets the status of the endpoint,
// which does nothing if the endpoint does not exist.
//
// This is the inner endpoint management of the loadbalancer forwarder.
func (f *Forwarder) SetEndpointStatus(epid string, status loadbalancer.EndpointStatus) {
	f.epmanager.SetEndpointStatus(epid, status)
}

// SetEndpointStatuses sets the statuses of a set of endpoints,
// which does nothing if the endpoint does not exist.
//
// This is the inner endpoint management of the loadbalancer forwarder.
func (f *Forwarder) SetEndpointStatuses(statuses map[string]loadbalancer.EndpointStatus) {
	f.epmanager.SetEndpointStatuses(statuses)
}

// ResetEndpoints resets all the endpoints to endpoints with EndpointStatusOnline.
//
// This is the inner endpoint management of the loadbalancer forwarder.
func (f *Forwarder) ResetEndpoints(eps ...loadbalancer.Endpoint) {
	f.epmanager.ResetEndpoints(eps...)
}

// UpsertEndpoints adds or updates the endpoints.
//
// Notice: the statuses of all the given new endpoints are EndpointStatusOnline.
//
// This is the inner endpoint management of the loadbalancer forwarder.
func (f *Forwarder) UpsertEndpoints(eps ...loadbalancer.Endpoint) {
	f.epmanager.UpsertEndpoints(eps...)
}

// UpsertEndpoint adds or updates the given endpoint with EndpointStatusOnline.
//
// This is the inner endpoint management of the loadbalancer forwarder,
// which implements the interface healthcheck.Updater#UpsertEndpoint.
func (f *Forwarder) UpsertEndpoint(ep loadbalancer.Endpoint) {
	f.epmanager.UpsertEndpoints(ep)
}

// RemoveEndpoint removes the endpoint by the endpoint id,
// which does nothing if the endpoint does not exist.
//
// This is the inner endpoint management of the loadbalancer forwarder,
// which implements the interface healthcheck.Updater#RemoveEndpoint.
func (f *Forwarder) RemoveEndpoint(epid string) {
	f.epmanager.RemoveEndpoint(epid)
}

// GetEndpoint returns the endpoint by the endpoint id.
//
// This is the inner endpoint management of the loadbalancer forwarder.
func (f *Forwarder) GetEndpoint(epid string) (ep loadbalancer.Endpoint, ok bool) {
	return f.epmanager.GetEndpoint(epid)
}

// OnlineNum implements the interfce loadbalancer.EndpointDiscovery#OnlineNum
// to return the number of the online endpoints.
func (f *Forwarder) OnlineNum() int {
	return f.epmanager.OnlineNum()
}

// OnEndpoints implements the inerface loadbalancer.EndpointDiscovery#OnEndpoints
// to only return all the online endpoints.
//
// This is the inner endpoint management of the loadbalancer forwarder.
func (f *Forwarder) OnEndpoints() loadbalancer.Endpoints {
	return f.epmanager.OnEndpoints()
}

// OffEndpoints implements the inerface loadbalancer.EndpointDiscovery#OffEndpoints
// to only return all the offline endpoints.
//
// This is the inner endpoint management of the loadbalancer forwarder.
func (f *Forwarder) OffEndpoints() loadbalancer.Endpoints {
	return f.epmanager.OffEndpoints()
}

// AllEndpoints implements the inerface loadbalancer.EndpointDiscovery#OnEndpoints
// to return all the endpoints, which are a set of the wrappers of the original
// endpoints and can be unwrapped by loadbalancer.UnwrapEndpoint.
//
// This is the inner endpoint management of the loadbalancer forwarder.
func (f *Forwarder) AllEndpoints() loadbalancer.Endpoints {
	return f.epmanager.AllEndpoints()
}
