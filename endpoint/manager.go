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
	"slices"
	"sync"

	"github.com/xgfone/go-atomicvalue"
)

var _ Discovery = new(Manager)

// Manager is used to manage a group of endpoints.
type Manager struct {
	lock sync.RWMutex
	eps  map[string]Endpoint

	oneps  atomicvalue.Value[Endpoints]
	offeps atomicvalue.Value[Endpoints]
	alleps atomicvalue.Value[Endpoints]
}

// NewManager returns a new endpoint manager.
func NewManager() *Manager {
	return &Manager{eps: make(map[string]Endpoint, 8)}
}

// Number implements the interface Discovery#Number.
func (m *Manager) Number() int { return len(m.OnEndpoints()) }

// Endpoints is the alias of OnEndpoints,
// which implements the interface Discovery#Endpoints,
func (m *Manager) Endpoints() Endpoints { return m.OnEndpoints() }

// OnEndpoints returns all the online endpoints.
func (m *Manager) OnEndpoints() Endpoints { return m.oneps.Load() }

// OffEndpoints returns all the offline endpoints.
func (m *Manager) OffEndpoints() Endpoints { return m.offeps.Load() }

// AllEndpoints returns all the endpoints.
func (m *Manager) AllEndpoints() Endpoints { return m.alleps.Load() }

// SetEndpointStatus sets the status of the endpoint.
func (m *Manager) SetEndpointStatus(epid string, status Status) {
	if ep, ok := m.GetEndpoint(epid); ok {
		ep.SetStatus(status)

		m.lock.RLock()
		m.updateEndpointsStatus()
		m.lock.RUnlock()
	}
}

// SetEndpointStatuses sets the statuses of a group of endpoints.
func (m *Manager) SetEndpointStatuses(epid2statuses map[string]Status) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var changed bool
	for epid, status := range epid2statuses {
		if ep, ok := m.eps[epid]; ok {
			ep.SetStatus(status)
			changed = true
		}
	}

	if changed {
		m.updateEndpointsStatus()
	}
}

// GetEndpoint returns the endpoint by the id.
func (m *Manager) GetEndpoint(epid string) (ep Endpoint, ok bool) {
	m.lock.RLock()
	ep, ok = m.eps[epid]
	m.lock.RUnlock()
	return
}

// ResetEndpoints resets all the endpoints to the news.
func (m *Manager) ResetEndpoints(news ...Endpoint) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, ep := range news {
		if _ep, ok := m.eps[ep.ID()]; ok { // Update
			_ = _ep.Update(ep.Info())
		} else { // Add
			m.eps[ep.ID()] = ep
		}
	}

	for id := range m.eps {
		index := slices.IndexFunc(news, func(ep Endpoint) bool { return ep.ID() == id })
		if index == -1 { // Not Exist, and Delete
			delete(m.eps, id)
		}
	}

	m.updateEndpoints()
}

// UpsertEndpoints adds or updates the endpoints.
func (m *Manager) UpsertEndpoints(eps ...Endpoint) {
	if len(eps) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	for _, ep := range eps {
		if _ep, ok := m.eps[ep.ID()]; ok {
			_ = _ep.Update(ep.Info())
		} else {
			m.eps[ep.ID()] = ep
		}
	}
	m.updateEndpoints()
}

// RemoveEndpoint removes the endpoint by the id.
func (m *Manager) RemoveEndpoint(epid string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.eps[epid]; ok {
		delete(m.eps, epid)
		m.updateEndpoints()
	}
}

func (m *Manager) updateEndpoints() {
	oneps := Acquire(len(m.eps))
	alleps := Acquire(len(m.eps))
	offeps := Acquire(0)
	for _, ep := range m.eps {
		alleps = append(alleps, ep)
		switch ep.Status() {
		case StatusOnline:
			oneps = append(oneps, ep)
		case StatusOffline:
			offeps = append(offeps, ep)
		}
	}

	swapEndpoints(&m.oneps, oneps)   // For online
	swapEndpoints(&m.offeps, offeps) // For offline
	swapEndpoints(&m.alleps, alleps) // For all
}

func (m *Manager) updateEndpointsStatus() {
	oneps := Acquire(len(m.eps))
	offeps := Acquire(0)

	for _, ep := range m.AllEndpoints() {
		switch ep.Status() {
		case StatusOnline:
			oneps = append(oneps, ep)
		case StatusOffline:
			offeps = append(offeps, ep)
		}
	}

	swapEndpoints(&m.oneps, oneps)   // For online
	swapEndpoints(&m.offeps, offeps) // For offline
}

func swapEndpoints(dsteps *atomicvalue.Value[Endpoints], neweps Endpoints) {
	if len(neweps) == 0 {
		oldeps := dsteps.Swap(nil)
		Release(oldeps)
		Release(neweps)
	} else {
		Sort(neweps)
		oldeps := dsteps.Swap(neweps)
		Release(oldeps)
	}
}
