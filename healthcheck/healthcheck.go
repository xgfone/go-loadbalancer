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
	"sync/atomic"
	"time"

	"github.com/xgfone/go-atomicvalue"
	"github.com/xgfone/go-generics/maps"
	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/internal/slog"
)

var (
	// DefaultHealthChecker is the default global health checker.
	DefaultHealthChecker = NewHealthChecker()

	// DefaultInterval is the default healthcheck interval.
	DefaultInterval = time.Second * 10

	// DefaultCheckConfig is the default health check configuration.
	DefaultCheckConfig = CheckConfig{Failure: 1, Timeout: time.Second, Interval: DefaultInterval}
)

// CheckConfig is the config to check the endpoint health.
type CheckConfig struct {
	Failure  int           `json:"failure,omitempty"`
	Timeout  time.Duration `json:"timeout,omitempty"`
	Interval time.Duration `json:"interval,omitempty"`
	Delay    time.Duration `json:"delay,omitempty"`
}

type endpointContext struct {
	tickch   chan time.Duration
	stopch   chan struct{}
	config   atomic.Value
	endpoint atomicvalue.Value[loadbalancer.Endpoint]
	failure  int
	online   int32
}

func newEndpointContext(ep loadbalancer.Endpoint, config CheckConfig) *endpointContext {
	c := &endpointContext{tickch: make(chan time.Duration), stopch: make(chan struct{})}
	c.SetEndpoint(ep)
	c.SetConfig(config)
	return c
}

// IsOnline reports whether the endpoint is online.
func (c *endpointContext) IsOnline() bool {
	return atomic.LoadInt32(&c.online) == 1
}

func (c *endpointContext) SetConfig(config CheckConfig) {
	if config.Interval <= 0 {
		if DefaultInterval > 0 {
			config.Interval = DefaultInterval
		} else {
			config.Interval = time.Second * 10
		}
	}

	c.config.Store(config)
	select {
	case c.tickch <- config.Interval:
	default:
	}
}

func (c *endpointContext) GetConfig() CheckConfig {
	return c.config.Load().(CheckConfig)
}

func (c *endpointContext) GetEndpoint() loadbalancer.Endpoint {
	return c.endpoint.Load()
}

func (c *endpointContext) SetEndpoint(ep loadbalancer.Endpoint) {
	c.endpoint.Store(ep)
}

func (c *endpointContext) Stop() {
	close(c.stopch)
}

func (c *endpointContext) beforeStart(hc *HealthChecker, exit <-chan struct{}) {
	config := c.GetConfig()
	if config.Delay > 0 {
		wait := time.NewTimer(config.Delay)
		select {
		case <-wait.C:
		case <-exit:
			wait.Stop()
			return
		}
	}
	c.checkHealth(hc, config)
}

func (c *endpointContext) Start(hc *HealthChecker, exit <-chan struct{}) {
	c.beforeStart(hc, exit)
	ticker := time.NewTicker(c.GetConfig().Interval)
	defer ticker.Stop()

	for {
		select {
		case <-exit:
			return

		case <-c.stopch:
			return

		case tick := <-c.tickch:
			ticker.Reset(tick)

		case <-ticker.C:
			c.checkHealth(hc, c.GetConfig())
		}
	}
}

func (c *endpointContext) wrapPanic() {
	if err := recover(); err != nil {
		slog.Error("wrap a panic when to check the endpoint health",
			"epid", c.GetEndpoint().ID())
	}
}

func (c *endpointContext) checkHealth(hc *HealthChecker, config CheckConfig) {
	defer c.wrapPanic()
	if c.updateOnlineStatus(c.checkEndpoint(config), config.Failure) {
		hc.setOnline(c.GetEndpoint().ID(), c.IsOnline())
	}
}

func (c *endpointContext) updateOnlineStatus(online bool, failure int) (ok bool) {
	if online {
		if c.failure > 0 {
			c.failure = 0
		}
		ok = atomic.CompareAndSwapInt32(&c.online, 0, 1)
	} else if c.failure++; c.failure > failure {
		ok = atomic.CompareAndSwapInt32(&c.online, 1, 0)
	}
	return
}

func (c *endpointContext) checkEndpoint(config CheckConfig) (online bool) {
	ctx := context.Background()
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	ep := c.GetEndpoint()
	online = ep.Check(ctx) == nil
	slog.Debug("HealthChecker: check the backend endpoint", "epid", ep.ID(), "online", online)
	return
}

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
	exit     chan struct{}
	slock    sync.RWMutex
	epmaps   map[string]*endpointContext
	updaters sync.Map
}

// NewHealthChecker returns a new health checker.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{epmaps: make(map[string]*endpointContext, 16)}
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
	if hc.exit != nil {
		close(hc.exit)
		hc.exit = nil
	}
}

// Start starts the health checker.
func (hc *HealthChecker) Start() {
	hc.slock.Lock()
	defer hc.slock.Unlock()

	if hc.exit == nil {
		hc.exit = make(chan struct{})
		for _, c := range hc.epmaps {
			go c.Start(hc, hc.exit)
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
		for id, c := range hc.epmaps {
			updater.UpsertEndpoint(c.GetEndpoint())
			updater.SetEndpointOnline(id, c.IsOnline())
		}
		hc.slock.RUnlock()
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
func (hc *HealthChecker) UpsertEndpoints(eps loadbalancer.Endpoints, config CheckConfig) {
	hc.slock.Lock()
	defer hc.slock.Unlock()
	for _, ep := range eps {
		hc.upsertEndpoint(ep, config)
	}
}

// UpsertEndpoint adds or updates the endpoint with the healthcheck config.
func (hc *HealthChecker) UpsertEndpoint(ep loadbalancer.Endpoint, config CheckConfig) {
	hc.slock.Lock()
	defer hc.slock.Unlock()
	hc.upsertEndpoint(ep, config)
}

func (hc *HealthChecker) upsertEndpoint(ep loadbalancer.Endpoint, config CheckConfig) {
	id := ep.ID()
	if c, ok := hc.epmaps[id]; ok {
		c.SetEndpoint(ep)
		c.SetConfig(config)
	} else {
		c = newEndpointContext(ep, config)
		hc.epmaps[id] = c
		if hc.exit != nil { // Has started
			go c.Start(hc, hc.exit)
		}

		hc.updaters.Range(func(_, value interface{}) bool {
			updater := value.(Updater)
			updater.UpsertEndpoint(ep)
			updater.SetEndpointOnline(id, c.IsOnline())
			return true
		})
	}
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
		ep.Online = c.IsOnline()
		ep.Config = c.GetConfig()
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
	Config CheckConfig
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
			Config:   c.GetConfig(),
			Online:   c.IsOnline(),
			Endpoint: c.GetEndpoint(),
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
		online = c.IsOnline()
	}
	return
}
