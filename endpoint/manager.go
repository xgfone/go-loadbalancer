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
	"sort"
	"sync"
	"sync/atomic"

	"github.com/xgfone/go-generics/slices"
	"github.com/xgfone/go-loadbalancer"
)

var _ loadbalancer.EndpointDiscovery = new(Manager)

// Manager is used to manage a group of endpoints.
type Manager struct {
	lock sync.RWMutex
	eps  map[string]loadbalancer.Endpoint

	oneps  atomic.Value
	offeps atomic.Value
	alleps atomic.Value
}

// NewManager returns a new endpoint manager.
func NewManager() *Manager {
	m := &Manager{eps: make(map[string]loadbalancer.Endpoint, 8)}
	m.alleps.Store(loadbalancer.Endpoints{})
	m.offeps.Store(loadbalancer.Endpoints{})
	m.oneps.Store(loadbalancer.Endpoints{})
	return m
}

// OnlineNum implements the interface EndpointDiscovery#OnlineNum.
func (m *Manager) OnlineNum() int {
	return len(m.OnEndpoints())
}

// OnEndpoints implements the interface EndpointDiscovery#OnEndpoints.
func (m *Manager) OnEndpoints() loadbalancer.Endpoints {
	return m.oneps.Load().(loadbalancer.Endpoints)
}

// OffEndpoints implements the interface EndpointDiscovery#OffEndpoints.
func (m *Manager) OffEndpoints() loadbalancer.Endpoints {
	return m.offeps.Load().(loadbalancer.Endpoints)
}

// AllEndpoints implements the interface EndpointDiscovery#AllEndpoints.
func (m *Manager) AllEndpoints() loadbalancer.Endpoints {
	return m.alleps.Load().(loadbalancer.Endpoints)
}

// SetEndpointStatus sets the status of the endpoint.
func (m *Manager) SetEndpointStatus(epid string, status loadbalancer.EndpointStatus) {
	if ep, ok := m.GetEndpoint(epid); ok {
		ep.SetStatus(status)

		m.lock.RLock()
		m.updateEndpointsStatus()
		m.lock.RUnlock()
	}
}

// SetEndpointStatuses sets the statuses of a group of endpoints.
func (m *Manager) SetEndpointStatuses(epid2statuses map[string]loadbalancer.EndpointStatus) {
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
func (m *Manager) GetEndpoint(epid string) (ep loadbalancer.Endpoint, ok bool) {
	m.lock.RLock()
	ep, ok = m.eps[epid]
	m.lock.RUnlock()
	return
}

// ResetEndpoints resets all the endpoints to the news.
func (m *Manager) ResetEndpoints(news ...loadbalancer.Endpoint) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, ep := range news {
		if _ep, ok := m.eps[ep.ID()]; ok { // Update
			_ep.Update(ep.Info())
		} else { // Add
			m.eps[ep.ID()] = ep
		}
	}

	for id := range m.eps {
		index := slices.IndexFunc(news, func(ep loadbalancer.Endpoint) bool { return ep.ID() == id })
		if index == -1 { // Not Exist, and Delete
			delete(m.eps, id)
		}
	}

	m.updateEndpoints()
}

// UpsertEndpoints adds or updates the endpoints.
func (m *Manager) UpsertEndpoints(eps ...loadbalancer.Endpoint) {
	if len(eps) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	for _, ep := range eps {
		if _ep, ok := m.eps[ep.ID()]; ok {
			_ep.Update(ep.Info())
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
	oneps := loadbalancer.AcquireEndpoints(len(m.eps))
	alleps := loadbalancer.AcquireEndpoints(len(m.eps))
	offeps := loadbalancer.AcquireEndpoints(0)
	for _, ep := range m.eps {
		alleps = append(alleps, ep)
		switch ep.Status() {
		case loadbalancer.EndpointStatusOnline:
			oneps = append(oneps, ep)
		case loadbalancer.EndpointStatusOffline:
			offeps = append(offeps, ep)
		}
	}

	swapEndpoints(&m.oneps, oneps)   // For online
	swapEndpoints(&m.offeps, offeps) // For offline
	swapEndpoints(&m.alleps, alleps) // For all
}

func (m *Manager) updateEndpointsStatus() {
	oneps := loadbalancer.AcquireEndpoints(len(m.eps))
	offeps := loadbalancer.AcquireEndpoints(0)

	for _, ep := range m.AllEndpoints() {
		switch ep.Status() {
		case loadbalancer.EndpointStatusOnline:
			oneps = append(oneps, ep)
		case loadbalancer.EndpointStatusOffline:
			offeps = append(offeps, ep)
		}
	}

	swapEndpoints(&m.oneps, oneps)   // For online
	swapEndpoints(&m.offeps, offeps) // For offline
}

func swapEndpoints(dsteps *atomic.Value, neweps loadbalancer.Endpoints) {
	if len(neweps) == 0 {
		oldeps := dsteps.Swap(loadbalancer.Endpoints{}).(loadbalancer.Endpoints)
		loadbalancer.ReleaseEndpoints(oldeps)
		loadbalancer.ReleaseEndpoints(neweps)
	} else {
		sort.Stable(neweps)
		oldeps := dsteps.Swap(neweps).(loadbalancer.Endpoints)
		loadbalancer.ReleaseEndpoints(oldeps)
	}
}
