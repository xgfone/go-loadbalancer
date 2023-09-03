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
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/xgfone/go-atomicvalue"
	"github.com/xgfone/go-checker"
)

// DefaultHealthChecker is the default global health checker.
var DefaultHealthChecker = New("default")

// DefaultConfig is the default configuration of the checker.
var DefaultConfig = Config{Failure: 2, Timeout: time.Second, Interval: time.Second * 10}

type _checker struct {
	id string
	hc *HealthChecker
	*checker.Checker
}

func newChecker(hc *HealthChecker, id string) *_checker {
	c := &_checker{id: id, hc: hc}

	config := hc.cconfig.Load().config
	config.Delay = delay(config.Interval)

	c.Checker = checker.NewChecker(id, c, hc.setOnline)
	c.Checker.SetJitter(jitter)
	c.Checker.SetConfig(config)
	return c
}

func (c *_checker) Online() bool                   { return c.Ok() }
func (c *_checker) Check(ctx context.Context) bool { return c.hc.check(ctx, c.id) }

func jitter(d time.Duration) time.Duration {
	return d + time.Duration(rand.Float64()*float64(d)*0.1)
}

func delay(d time.Duration) time.Duration {
	if d = time.Duration(rand.Float64() * float64(d) * 0.5); d < time.Millisecond*10 {
		d = 0
	}
	return d
}

// Config is the checker confgiuration.
type Config struct {
	Failure  int           // The failure number to be changed to unhealth.
	Timeout  time.Duration // The timeout duration to check the target.
	Interval time.Duration // The interval duration between twice checkings.
}

func (c Config) cconfig() cconfig {
	if c.Failure <= 0 {
		c.Failure = 2
	}
	if c.Timeout < 0 {
		c.Timeout = 0
	}
	if c.Interval < time.Second {
		c.Interval = time.Second * 10
	}

	return cconfig{
		Config: c,
		config: checker.Config{
			Failure:  c.Failure,
			Timeout:  c.Timeout,
			Interval: c.Interval,
		},
	}
}

type cconfig struct {
	config checker.Config
	Config
}

// HealthChecker is a health checker to check whether a set of targets are healthy.
type HealthChecker struct {
	name string

	lock    sync.RWMutex
	maps    map[string]*_checker
	cancel  context.CancelFunc
	context context.Context

	updater atomicvalue.Value[func(string, bool)]
	checker atomicvalue.Value[func(context.Context, string) bool]
	cconfig atomicvalue.Value[cconfig]
}

// New returns a new health checker.
func New(name string) *HealthChecker {
	return &HealthChecker{
		name:    name,
		maps:    make(map[string]*_checker, 16),
		cconfig: atomicvalue.NewValue(DefaultConfig.cconfig()),
	}
}

func (hc *HealthChecker) setOnline(id string, online bool) {
	if cb := hc.updater.Load(); cb != nil {
		cb(id, online)
	} else {
		slog.Warn("HealthChecker: not set the status callback function",
			"name", hc.name, "id", id, "online", online)
	}
}

func (hc *HealthChecker) check(ctx context.Context, id string) bool {
	if check := hc.checker.Load(); check != nil {
		return check(ctx, id)
	}
	slog.Warn("HealthChecker: not set the check function", "name", hc.name, "id", id)
	return false
}

// OnChanged sets the callback function when the status is changed.
func (hc *HealthChecker) OnChanged(cb func(id string, online bool)) {
	hc.updater.Store(cb)
}

// SetChecker set the check function to intercept the health checking.
func (hc *HealthChecker) SetChecker(f func(ctx context.Context, id string) bool) {
	hc.checker.Store(f)
}

// SetConfig sets the unified configuration of the checker.
func (hc *HealthChecker) SetConfig(c Config) {
	hc.lock.RLock()
	defer hc.lock.RUnlock()

	cconfig := c.cconfig()
	hc.cconfig.Store(cconfig)
	for _, c := range hc.maps {
		c.SetConfig(cconfig.config)
	}
}

// Config returns the unified configuration of the checker.
func (hc *HealthChecker) Config() Config {
	return hc.cconfig.Load().Config
}

// Start starts the health checker.
func (hc *HealthChecker) Start() {
	hc.lock.Lock()
	defer hc.lock.Unlock()

	if hc.cancel == nil {
		hc.context, hc.cancel = context.WithCancel(context.Background())
		for _, c := range hc.maps {
			go c.Start(hc.context)
		}
	} else {
		panic("HealthChecker: has been started")
	}
}

// Stop stops the health checker.
func (hc *HealthChecker) Stop() {
	hc.lock.Lock()
	defer hc.lock.Unlock()
	if hc.cancel != nil {
		hc.cancel()
		hc.cancel = nil
		hc.context = nil
	}
}

// AddTarget adds a checked target.
//
// If exist, do nothing.
func (hc *HealthChecker) AddTarget(id string) {
	hc.lock.Lock()
	defer hc.lock.Unlock()
	hc.addTarget(id)
}

// AddTargets adds a set of the checked targets.
//
// If a certain target has exists, do nothing for it.
func (hc *HealthChecker) AddTargets(ids ...string) {
	hc.lock.Lock()
	defer hc.lock.Unlock()
	for _, id := range ids {
		hc.addTarget(id)
	}
}

func (hc *HealthChecker) addTarget(id string) {
	if _, ok := hc.maps[id]; !ok {
		c := newChecker(hc, id)
		hc.maps[id] = c
		if hc.context != nil {
			go c.Start(hc.context)
		}
	}
}

// DelTarget deletes the target and stops to check its health.
func (hc *HealthChecker) DelTarget(id string) {
	hc.lock.Lock()
	defer hc.lock.Unlock()
	c, ok := hc.maps[id]
	if ok {
		hc.delTarget(id, c)
	}
}

// DelTargets deletes a set of targets and stops to their health.
func (hc *HealthChecker) DelTargets(ids ...string) {
	if len(ids) == 0 {
		return
	}

	hc.lock.Lock()
	defer hc.lock.Unlock()
	for _, id := range ids {
		if c, ok := hc.maps[id]; ok {
			hc.delTarget(id, c)
		}
	}
}

func (hc *HealthChecker) delTarget(id string, c *_checker) {
	delete(hc.maps, id)
	c.Stop()

	// (xgf): we need to call the callback to update the status to offline??
	// hc.setOnline(id, false)
}

// ResetTargets resets the targets to the given.
func (hc *HealthChecker) ResetTargets(ids ...string) {
	hc.lock.Lock()
	defer hc.lock.Unlock()

	// add new targets.
	idm := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idm[id] = struct{}{}
		hc.addTarget(id)
	}

	// remove the redundant targets.
	for id, c := range hc.maps {
		if _, ok := idm[id]; !ok {
			hc.delTarget(id, c)
		}
	}
}

// Targets returns the health statuses of all the targets.
func (hc *HealthChecker) Targets() map[string]bool {
	hc.lock.RLock()
	idm := make(map[string]bool, len(hc.maps))
	for id, c := range hc.maps {
		idm[id] = c.Online()
	}
	hc.lock.RUnlock()
	return idm
}

// Target returns the health status of the given target.
func (hc *HealthChecker) Target(id string) (online bool, ok bool) {
	hc.lock.RLock()
	c, ok := hc.maps[id]
	if ok {
		online = c.Online()
	}
	hc.lock.RUnlock()
	return
}
