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

// Package healthcheck provides a health checker to check whether a set
// of endpoints are healthy.
package healthcheck

import (
	"context"
	"fmt"
	"sync"

	"github.com/xgfone/go-atomicvalue"
	"github.com/xgfone/go-checker"
	"github.com/xgfone/go-loadbalancer/endpoint"
)

// DefaultHealthChecker is the default global health checker.
var DefaultHealthChecker = NewHealthChecker()

type epchecker struct {
	endpoint.Endpoint
	*checker.Checker

	config atomicvalue.Value[Checker]
	hcheck *HealthChecker
}

func newEndpointChecker(hc *HealthChecker, ep endpoint.Endpoint, config Checker) *epchecker {
	epc := &epchecker{Endpoint: ep, hcheck: hc}
	epc.Checker = checker.NewChecker(ep.ID(), epc, hc.setOnline)
	epc.SetChecker(config)
	return epc
}

func (c *epchecker) Check(ctx context.Context) bool {
	return c.hcheck.checkEndpoint(ctx, c.Endpoint, c.GetChecker().Req)
}

func (c *epchecker) GetChecker() Checker       { return c.config.Load() }
func (c *epchecker) SetChecker(config Checker) { c.config.Store(config); c.SetConfig(config.Config) }

func (c *epchecker) Unwrap() endpoint.Endpoint        { return c.Endpoint }
func (c *epchecker) GetEndpoint() endpoint.Endpoint   { return c.Endpoint }
func (c *epchecker) SetEndpoint(ep endpoint.Endpoint) { c.Endpoint.Update(ep.Info()) }

// Updater is used to update the endpoint status.
type Updater interface {
	UpsertEndpoint(endpoint.Endpoint)
	RemoveEndpoint(epid string)
	SetEndpointOnline(epid string, online bool)
}

// Checker is used to check the endpoint.
type Checker struct {
	Config checker.Config
	Req    interface{}
}

// HealthChecker is a health checker to check whether a set of endpoints
// are healthy.
//
// Notice: if there are lots of endpoints to be checked, you maybe need
// an external checker.
type HealthChecker struct {
	slock    sync.RWMutex
	epmaps   map[string]*epchecker
	cancel   context.CancelFunc
	context  context.Context
	updaters sync.Map

	epcheck atomicvalue.Value[func(context.Context, endpoint.Endpoint, interface{}) bool]
}

// NewHealthChecker returns a new health checker.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{epmaps: make(map[string]*epchecker, 16)}
}

func (hc *HealthChecker) setOnline(epid string, online bool) {
	hc.updaters.Range(func(_, value interface{}) bool {
		value.(Updater).SetEndpointOnline(epid, online)
		return true
	})
}

func (hc *HealthChecker) checkEndpoint(ctx context.Context, ep endpoint.Endpoint, req interface{}) bool {
	if check := hc.epcheck.Load(); check != nil {
		return check(ctx, ep, req)
	}
	return ep.Check(ctx, req)
}

// SetEndpointCheck set the check function to intercept the healthcheck of the endpoint.
//
// f may be nil, which is equal to call the Check method of Endpoint directly.
func (hc *HealthChecker) SetEndpointCheck(f func(context.Context, endpoint.Endpoint, interface{}) bool) {
	hc.epcheck.Store(f)
}

// Stop stops the health checker.
func (hc *HealthChecker) Stop() {
	hc.slock.Lock()
	defer hc.slock.Unlock()
	if hc.cancel != nil {
		hc.cancel()
		hc.cancel = nil
		hc.context = nil
	}
}

// Start starts the health checker.
func (hc *HealthChecker) Start() {
	hc.slock.Lock()
	defer hc.slock.Unlock()
	if hc.cancel == nil {
		hc.context, hc.cancel = context.WithCancel(context.Background())
		for _, c := range hc.epmaps {
			go c.Start(hc.context)
		}
	} else {
		panic("HealthChecker: has been started")
	}
}

// AddUpdater adds the healthcheck updater with the name.
func (hc *HealthChecker) AddUpdater(name string, updater Updater) (err error) {
	if name == "" {
		panic("HealthChecker: the name is empty")
	} else if updater == nil {
		panic("HealthChecker: the updater is nil")
	}

	if _, loaded := hc.updaters.LoadOrStore(name, updater); loaded {
		err = fmt.Errorf("the healthcheck updater named '%s' has been added", name)
	} else {
		hc.slock.RLock()
		defer hc.slock.RUnlock()
		for id, c := range hc.epmaps {
			updater.UpsertEndpoint(c.GetEndpoint())
			updater.SetEndpointOnline(id, c.Ok())
		}
	}
	return
}

// DelUpdater deletes the healthcheck updater by the name.
func (hc *HealthChecker) DelUpdater(name string) {
	if name == "" {
		panic("the healthcheck name is empty")
	}
	hc.updaters.Delete(name)
}

// GetUpdater returns the healthcheck updater by the name.
//
// Return nil if the healthcheck updater does not exist.
func (hc *HealthChecker) GetUpdater(name string) Updater {
	if name == "" {
		panic("the healthcheck name is empty")
	}

	if value, ok := hc.updaters.Load(name); ok {
		return value.(Updater)
	}
	return nil
}

// GetUpdaters returns all the healthcheck updaters.
func (hc *HealthChecker) GetUpdaters() map[string]Updater {
	updaters := make(map[string]Updater, 32)
	hc.updaters.Range(func(key, value interface{}) bool {
		updaters[key.(string)] = value.(Updater)
		return true
	})
	return updaters
}

// ResetChecker resets the checker of all the endpoints.
func (hc *HealthChecker) ResetChecker(checker Checker) {
	hc.slock.Lock()
	defer hc.slock.Unlock()
	for _, c := range hc.epmaps {
		c.SetChecker(checker)
	}
}

// ResetEndpoints resets all the endpoints to the news.
func (hc *HealthChecker) ResetEndpoints(news endpoint.Endpoints, checker Checker) {
	hc.slock.Lock()
	defer hc.slock.Unlock()

	for _, ep := range news {
		hc.upsertEndpoint(ep, checker)
	}

	for id := range hc.epmaps {
		if !news.Contains(id) {
			c, ok := hc.epmaps[id]
			if ok {
				delete(hc.epmaps, id)
			}
			hc.cleanChecker(id, c, ok)
		}
	}
}

// UpsertEndpoints adds or updates a set of the endpoints with the same healthcheck config.
func (hc *HealthChecker) UpsertEndpoints(eps endpoint.Endpoints, checker Checker) {
	hc.slock.Lock()
	defer hc.slock.Unlock()
	for _, ep := range eps {
		hc.upsertEndpoint(ep, checker)
	}
}

// UpsertEndpoint adds or updates the endpoint with the healthcheck config.
func (hc *HealthChecker) UpsertEndpoint(ep endpoint.Endpoint, checker Checker) {
	hc.slock.Lock()
	defer hc.slock.Unlock()
	hc.upsertEndpoint(ep, checker)
}

func (hc *HealthChecker) upsertEndpoint(ep endpoint.Endpoint, checker Checker) {
	id := ep.ID()
	c, ok := hc.epmaps[id]
	if ok {
		c.SetEndpoint(ep)
		c.SetChecker(checker)
	} else {
		c = newEndpointChecker(hc, ep, checker)
		hc.epmaps[id] = c
		if hc.context != nil { // Has started
			go c.Start(hc.context)
		}
	}

	hc.updaters.Range(func(_, value interface{}) bool {
		updater := value.(Updater)
		updater.UpsertEndpoint(ep)
		updater.SetEndpointOnline(id, c.Ok())
		return true
	})
}

// RemoveEndpoint removes the endpoint by the endpoint id.
func (hc *HealthChecker) RemoveEndpoint(epid string) {
	if epid == "" {
		panic("the endpoint id is empty")
	}

	hc.slock.Lock()
	c, ok := hc.epmaps[epid]
	if ok {
		delete(hc.epmaps, epid)
	}
	hc.slock.Unlock()
	hc.cleanChecker(epid, c, ok)
}

func (hc *HealthChecker) cleanChecker(id string, c *epchecker, ok bool) {
	if ok {
		c.Stop()
		hc.updaters.Range(func(_, value interface{}) bool {
			value.(Updater).RemoveEndpoint(id)
			return true
		})
	}
}

// GetEndpoint returns the endpoint by the endpoint id.
//
// Return nil if the endpoint does not exist.
func (hc *HealthChecker) GetEndpoint(epid string) (ep EndpointInfo, ok bool) {
	if epid == "" {
		panic("the endpoint id is empty")
	}

	hc.slock.RLock()
	c, ok := hc.epmaps[epid]
	hc.slock.RUnlock()

	if ok {
		ep.Online = c.Ok()
		ep.Checker = c.GetChecker()
		ep.Endpoint = c.GetEndpoint()
	}
	return
}

var _ endpoint.Endpoint = EndpointInfo{}

// EndpointInfo represents the information of the endpoint.
type EndpointInfo struct {
	endpoint.Endpoint
	Checker Checker
	Online  bool
}

// Unwrap unwraps the inner endpoint.
func (si EndpointInfo) Unwrap() endpoint.Endpoint {
	return si.Endpoint
}

// Status overrides the interface method endpoint.Endpoint#Status.
func (si EndpointInfo) Status() endpoint.Status {
	if si.Online {
		return endpoint.StatusOnline
	}
	return endpoint.StatusOffline
}

// GetEndpoints returns all the endpoints.
func (hc *HealthChecker) GetEndpoints() []EndpointInfo {
	hc.slock.RLock()
	eps := make([]EndpointInfo, 0, len(hc.epmaps))
	for _, c := range hc.epmaps {
		eps = append(eps, EndpointInfo{
			Endpoint: c.GetEndpoint(),
			Checker:  c.GetChecker(),
			Online:   c.Ok(),
		})
	}
	hc.slock.RUnlock()
	return eps
}

// EndpointIsOnline reports whether the endpoint is online.
func (hc *HealthChecker) EndpointIsOnline(epid string) (online, ok bool) {
	if epid == "" {
		panic("the endpoint id is empty")
	}

	hc.slock.RLock()
	c, ok := hc.epmaps[epid]
	hc.slock.RUnlock()

	if ok {
		online = c.Ok()
	}
	return
}
