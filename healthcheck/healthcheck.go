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

	"github.com/xgfone/go-checker"
	"github.com/xgfone/go-generics/maps"
	"github.com/xgfone/go-loadbalancer"
)

// DefaultHealthChecker is the default global health checker.
var DefaultHealthChecker = NewHealthChecker()

type epchecker struct {
	loadbalancer.Endpoint
	*checker.Checker
}

func newEndpointChecker(hc *HealthChecker, ep loadbalancer.Endpoint, conf checker.Config) *epchecker {
	return &epchecker{Endpoint: ep, Checker: checker.NewChecker(ep.ID(), ep, hc.setOnline)}
}

func (c *epchecker) GetEndpoint() loadbalancer.Endpoint   { return c.Endpoint }
func (c *epchecker) SetEndpoint(ep loadbalancer.Endpoint) { c.Endpoint.Update(ep.Info()) }

// Updater is used to update the endpoint status.
type Updater interface {
	UpsertEndpoint(loadbalancer.Endpoint)
	RemoveEndpoint(epid string)
	SetEndpointOnline(epid string, online bool)
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

// UpsertEndpoints adds or updates a set of the endpoints with the same healthcheck config.
func (hc *HealthChecker) UpsertEndpoints(eps loadbalancer.Endpoints, config checker.Config) {
	hc.slock.Lock()
	defer hc.slock.Unlock()
	for _, ep := range eps {
		hc.upsertEndpoint(ep, config)
	}
}

// UpsertEndpoint adds or updates the endpoint with the healthcheck config.
func (hc *HealthChecker) UpsertEndpoint(ep loadbalancer.Endpoint, config checker.Config) {
	hc.slock.Lock()
	defer hc.slock.Unlock()
	hc.upsertEndpoint(ep, config)
}

func (hc *HealthChecker) upsertEndpoint(ep loadbalancer.Endpoint, config checker.Config) {
	id := ep.ID()
	c, ok := hc.epmaps[id]
	if ok {
		c.SetEndpoint(ep)
		c.SetConfig(config)
	} else {
		c = newEndpointChecker(hc, ep, config)
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
	c, ok := maps.Pop(hc.epmaps, epid)
	hc.slock.Unlock()
	if ok {
		c.Stop()
		hc.updaters.Range(func(_, value interface{}) bool {
			value.(Updater).RemoveEndpoint(epid)
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
		ep.Config = c.Config()
		ep.Endpoint = c.GetEndpoint()
	}
	return
}

var (
	_ loadbalancer.Endpoint        = EndpointInfo{}
	_ loadbalancer.EndpointWrapper = EndpointInfo{}
)

// EndpointInfo represents the information of the endpoint.
type EndpointInfo struct {
	loadbalancer.Endpoint
	Config checker.Config
	Online bool
}

// Unwrap unwraps the inner endpoint.
func (si EndpointInfo) Unwrap() loadbalancer.Endpoint {
	return si.Endpoint
}

// Status overrides the interface method loadbalancer.Endpoint#Status.
func (si EndpointInfo) Status() loadbalancer.EndpointStatus {
	if si.Online {
		return loadbalancer.EndpointStatusOnline
	}
	return loadbalancer.EndpointStatusOffline
}

// GetEndpoints returns all the endpoints.
func (hc *HealthChecker) GetEndpoints() []EndpointInfo {
	hc.slock.RLock()
	eps := make([]EndpointInfo, 0, len(hc.epmaps))
	for _, c := range hc.epmaps {
		eps = append(eps, EndpointInfo{
			Endpoint: c.GetEndpoint(),
			Config:   c.Config(),
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
