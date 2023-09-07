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

package endpoint

import (
	"sync"
	"sync/atomic"

	"github.com/xgfone/go-atomicvalue"
	"github.com/xgfone/go-loadbalancer"
)

type epwrapper struct {
	ep atomicvalue.Value[loadbalancer.Endpoint]
	on atomic.Bool
}

func newEndpointWrapper(ep loadbalancer.Endpoint) *epwrapper {
	return &epwrapper{ep: atomicvalue.NewValue(ep)}
}

func (w *epwrapper) Online() bool               { return w.on.Load() }
func (w *epwrapper) SetOnline(online bool) bool { return w.on.Swap(online) != online }

func (w *epwrapper) Endpoint() loadbalancer.Endpoint      { return w.ep.Load() }
func (w *epwrapper) SetEndpoint(ep loadbalancer.Endpoint) { w.ep.Store(ep) }

// Manager is used to manage a group of endpoints.
type Manager struct {
	lock   sync.RWMutex
	alleps map[string]*epwrapper
	oneps  atomicvalue.Value[*loadbalancer.Static]
}

// NewManager returns a new endpoint manager.
//
// NOTICE: For the added endpoint, it should be comparable.
func NewManager(initcap int) *Manager {
	m := &Manager{alleps: make(map[string]*epwrapper, initcap)}
	m.oneps.Store(loadbalancer.None)
	return m
}

var _ loadbalancer.Discovery = new(Manager)

// Discover returns all the online endpoints, which implements the interface Discovery.
func (m *Manager) Discover() *loadbalancer.Static { return m.oneps.Load() }

// All returns all the endpoints with the online status.
func (m *Manager) All() map[loadbalancer.Endpoint]bool {
	m.lock.RLock()
	eps := make(map[loadbalancer.Endpoint]bool, len(m.alleps))
	for _, w := range m.alleps {
		eps[w.Endpoint()] = w.Online()
	}
	m.lock.RUnlock()
	return eps
}

// Range ranges all the endpoints until the range function returns false
// or all the endpoints are ranged.
func (m *Manager) Range(f func(ep loadbalancer.Endpoint, online bool) bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, w := range m.alleps {
		if !f(w.Endpoint(), w.Online()) {
			break
		}
	}
}

// Len returns the length of all the endpoints.
func (m *Manager) Len() int {
	m.lock.RLock()
	n := len(m.alleps)
	m.lock.RUnlock()
	return n
}

// Get returns the endpoint by the id.
//
// If not exist, return (nil, false).
func (m *Manager) Get(epid string) (ep loadbalancer.Endpoint, online bool) {
	m.lock.RLock()
	w := m.alleps[epid]
	m.lock.RUnlock()

	if w != nil {
		ep = w.Endpoint()
		online = w.Online()
	}
	return
}

// Delete deletes the endpoint by the id.
//
// If not exist, do nothing.
func (m *Manager) Delete(epid string) {
	if len(epid) == 0 {
		return
	}

	m.lock.Lock()
	if w, ok := m.alleps[epid]; ok {
		delete(m.alleps, epid)
		if w.Online() {
			m.updateEndpoints()
		}
	}
	m.lock.Unlock()
}

// Deletes deletes a set of endpoints by the ids.
//
// If not exist, do nothing.
func (m *Manager) Deletes(epids ...string) {
	if len(epids) == 0 {
		return
	}

	m.lock.Lock()
	var changed bool
	for _, epid := range epids {
		if w, ok := m.alleps[epid]; ok {
			delete(m.alleps, epid)
			if w.Online() {
				changed = true
			}
		}
	}
	if changed {
		m.updateEndpoints()
	}
	m.lock.Unlock()
}

// Add adds the endpoint.
//
// If exist, do nothing and return false. Or, return true.
//
// NOTICE: the initial online status of the endpoint is false.
func (m *Manager) Add(ep loadbalancer.Endpoint) (ok bool) {
	if ep == nil {
		panic("Manager.AddEndpoint: endpoint must not be nil")
	}

	id := ep.ID()
	m.lock.Lock()
	if _, ok = m.alleps[id]; !ok {
		m.alleps[id] = newEndpointWrapper(ep)
	}
	m.lock.Unlock()
	return
}

// Adds adds a set of endpoints.
//
// If a certain endpoint has existed, do nothing.
//
// NOTICE: the initial online status of the added endpoint is false.
func (m *Manager) Adds(eps ...loadbalancer.Endpoint) {
	if len(eps) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	for _, ep := range eps {
		id := ep.ID()
		if _, ok := m.alleps[id]; !ok {
			m.alleps[id] = newEndpointWrapper(ep)
		}
	}
}

// Upsert adds the endpoint if not exist and return true,
// or replaces the endpoint and returns false.
//
// NOTICE: the initial online status of the added endpoint is false.
func (m *Manager) Upsert(ep loadbalancer.Endpoint) (ok bool) {
	if ep == nil {
		panic("Manager.AddEndpoint: endpoint must not be nil")
	}

	id := ep.ID()
	m.lock.Lock()
	if m.upsert(id, ep) {
		m.updateEndpoints()
	}
	m.lock.Unlock()
	return !ok
}

func (m *Manager) upsert(id string, ep loadbalancer.Endpoint) (updated bool) {
	w, ok := m.alleps[id]
	if ok {
		w.SetEndpoint(ep)
		updated = w.Online()
	} else {
		m.alleps[id] = newEndpointWrapper(ep)
	}
	return
}

// Upserts adds a set of endpoints if not exist, or replaces them.
//
// NOTICE: the initial online status of the added endpoint is false.
func (m *Manager) Upserts(eps ...loadbalancer.Endpoint) {
	if len(eps) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	var updated bool
	for _, ep := range eps {
		if m.upsert(ep.ID(), ep) && !updated {
			updated = true
		}
	}
	if updated {
		m.updateEndpoints()
	}
}

// Clear clears all the endpoints.
func (m *Manager) Clear() {
	m.lock.Lock()
	if len(m.alleps) > 0 {
		clear(m.alleps)
		m.updateEndpoints()
	}
	m.lock.Unlock()
}

// SetOnline sets the online status of the endpoint.
func (m *Manager) SetOnline(epid string, online bool) {
	m.lock.Lock()
	if w := m.alleps[epid]; w != nil && w.SetOnline(online) {
		m.updateEndpoints()
	}
	m.lock.Unlock()
}

// SetOnlines sets the online statuses of a set of endpoints.
func (m *Manager) SetOnlines(onlines map[string]bool) {
	m.lock.Lock()
	var changed bool
	for epid, online := range onlines {
		if w := m.alleps[epid]; w != nil {
			if w.SetOnline(online) {
				changed = true
			}
		}
	}
	if changed {
		m.updateEndpoints()
	}
	m.lock.Unlock()
}

// SetAllOnline sets the online status of all the endpoints to online.
func (m *Manager) SetAllOnline(online bool) {
	m.lock.Lock()
	var changed bool
	for _, w := range m.alleps {
		if w.SetOnline(online) {
			changed = true
		}
	}
	if changed {
		m.updateEndpoints()
	}
	m.lock.Unlock()
}

func (m *Manager) updateEndpoints() {
	oneps := loadbalancer.None
	if len(m.alleps) > 0 {
		oneps = loadbalancer.Acquire(len(m.alleps))
		for _, w := range m.alleps {
			if w.Online() {
				oneps.Endpoints = append(oneps.Endpoints, w.Endpoint())
			}
		}
		Sort(oneps.Endpoints)
	}
	loadbalancer.Release(m.oneps.Swap(oneps))
}
