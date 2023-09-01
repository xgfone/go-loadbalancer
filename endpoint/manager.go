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
)

type epwrapper struct {
	ep atomicvalue.Value[Endpoint]
	ok atomic.Bool
}

func newEndpointWrapper(ep Endpoint) *epwrapper {
	return &epwrapper{ep: atomicvalue.NewValue(ep)}
}

func (w *epwrapper) Online() bool               { return w.ok.Load() }
func (w *epwrapper) SetOnline(online bool) bool { return w.ok.Swap(online) != online }

func (w *epwrapper) Endpoint() Endpoint      { return w.ep.Load() }
func (w *epwrapper) SetEndpoint(ep Endpoint) { w.ep.Store(ep) }

// Manager is used to manage a group of endpoints.
type Manager struct {
	lock   sync.RWMutex
	alleps map[string]*epwrapper
	oneps  atomicvalue.Value[*Static]
}

var defaultStatic = new(Static)

// NewManager returns a new endpoint manager.
func NewManager(initcap int) *Manager {
	return &Manager{
		alleps: make(map[string]*epwrapper, initcap),
		oneps:  atomicvalue.NewValue(defaultStatic),
	}
}

var _ Discovery = new(Manager)

// Onlen returns the length of the online endpoints.
// which implements the interface Discovery#Onlen,
func (m *Manager) Onlen() int { return len(m.On()) }

// Onlines is the alias of OnEndpoints,
// which implements the interface Discovery#Onlines,
func (m *Manager) Onlines() Endpoints { return m.On() }

// On returns all the online endpoints, which is read-only.
func (m *Manager) On() Endpoints { return m.oneps.Load().Endpoints }

// Off returns all the offline endpoints.
func (m *Manager) Off() Endpoints {
	m.lock.RLock()
	eps := make(Endpoints, 0, len(m.alleps)/4)
	for _, w := range m.alleps {
		if !w.Online() {
			eps = append(eps, w.Endpoint())
		}
	}
	m.lock.RUnlock()
	return eps
}

// All returns all the endpoints with the online status.
func (m *Manager) All() map[Endpoint]bool {
	m.lock.RLock()
	eps := make(map[Endpoint]bool, len(m.alleps))
	for _, w := range m.alleps {
		eps[w.Endpoint()] = w.Online()
	}
	m.lock.RUnlock()
	return eps
}

// Range ranges all the endpoints until the range function returns false
// or all the endpoints are ranged.
func (m *Manager) Range(f func(ep Endpoint, online bool) bool) {
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
func (m *Manager) Get(epid string) (ep Endpoint, online bool) {
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
func (m *Manager) Add(ep Endpoint) (ok bool) {
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
func (m *Manager) Adds(eps ...Endpoint) {
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
func (m *Manager) Upsert(ep Endpoint) (ok bool) {
	if ep == nil {
		panic("Manager.AddEndpoint: endpoint must not be nil")
	}

	id := ep.ID()
	m.lock.Lock()
	m.upsert(id, ep)
	m.lock.Unlock()
	return !ok
}

func (m *Manager) upsert(id string, ep Endpoint) {
	w, ok := m.alleps[id]
	if ok {
		w.SetEndpoint(ep)
		if w.Online() {
			m.updateEndpoints()
		}
	} else {
		m.alleps[id] = newEndpointWrapper(ep)
	}
}

// Upserts adds a set of endpoints if not exist, or replaces them.
//
// NOTICE: the initial online status of the added endpoint is false.
func (m *Manager) Upserts(eps ...Endpoint) {
	if len(eps) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	for _, ep := range eps {
		m.upsert(ep.ID(), ep)
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
	if w := m.alleps[epid]; w != nil {
		if w.SetOnline(online) {
			m.updateEndpoints()
		}
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
	oneps := defaultStatic
	if _len := len(m.alleps); _len > 0 {
		oneps = Acquire(len(m.alleps))
		for _, w := range m.alleps {
			if w.Online() {
				oneps.Endpoints = append(oneps.Endpoints, w.Endpoint())
			}
		}
		Sort(oneps.Endpoints)
	}

	if oldeps := m.oneps.Swap(oneps); cap(oldeps.Endpoints) > 0 {
		clear(oldeps.Endpoints)
		oldeps.Endpoints = oldeps.Endpoints[:0]
		Release(oldeps)
	}
}
