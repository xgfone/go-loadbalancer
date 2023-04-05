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

package forwarder

import (
	"sort"
	"sync"
	"sync/atomic"

	"github.com/xgfone/go-generics/maps"
	"github.com/xgfone/go-loadbalancer"
)

var _ loadbalancer.EndpointWrapper = new(endpoint)

type endpoint struct {
	loadbalancer.Endpoint
	status atomic.Value
}

func newEndpoint(ep loadbalancer.Endpoint) *endpoint {
	s := &endpoint{Endpoint: ep}
	s.status.Store(loadbalancer.EndpointStatusOnline)
	return s
}

func (s *endpoint) Unwrap() loadbalancer.Endpoint {
	return s.Endpoint
}

func (s *endpoint) Status() loadbalancer.EndpointStatus {
	return s.status.Load().(loadbalancer.EndpointStatus)
}

func (s *endpoint) SetStatus(status loadbalancer.EndpointStatus) (ok bool) {
	return s.status.CompareAndSwap(s.Status(), status)
}

var _ loadbalancer.EndpointDiscovery = new(endpointsManager)

type endpointsManager struct {
	lock sync.RWMutex
	eps  map[string]*endpoint

	oneps  atomic.Value
	offeps atomic.Value
	alleps atomic.Value
}

func newEndpointsManager() *endpointsManager {
	m := &endpointsManager{eps: make(map[string]*endpoint, 8)}
	m.alleps.Store(loadbalancer.Endpoints{})
	m.offeps.Store(loadbalancer.Endpoints{})
	m.oneps.Store(loadbalancer.Endpoints{})
	return m
}

// OnlineNum implements the interface loadbalancer.EndpointDiscovery#OnlineNum.
func (m *endpointsManager) OnlineNum() int {
	return len(m.OnEndpoints())
}

// OnEndpoints implements the interface loadbalancer.EndpointDiscovery#OnEndpoints.
func (m *endpointsManager) OnEndpoints() loadbalancer.Endpoints {
	return m.oneps.Load().(loadbalancer.Endpoints)
}

// OffEndpoints implements the interface loadbalancer.EndpointDiscovery#OffEndpoints.
func (m *endpointsManager) OffEndpoints() loadbalancer.Endpoints {
	return m.offeps.Load().(loadbalancer.Endpoints)
}

// AllEndpoints implements the interface loadbalancer.EndpointDiscovery#AllEndpoints.
func (m *endpointsManager) AllEndpoints() loadbalancer.Endpoints {
	return m.alleps.Load().(loadbalancer.Endpoints)
}

func (m *endpointsManager) SetEndpointStatus(epid string, status loadbalancer.EndpointStatus) {
	m.lock.RLock()
	if ep, ok := m.eps[epid]; ok {
		if ep.SetStatus(status) {
			m.updateEndpoints()
		}
	}
	m.lock.RUnlock()
}

func (m *endpointsManager) SetEndpointStatuses(statuses map[string]loadbalancer.EndpointStatus) {
	m.lock.RLock()
	var changed bool
	for epid, status := range statuses {
		if ep, ok := m.eps[epid]; ok {
			if ep.SetStatus(status) && !changed {
				changed = true
			}
		}
	}
	if changed {
		m.updateEndpoints()
	}
	m.lock.RUnlock()
	return
}

func (m *endpointsManager) GetEndpoint(epid string) (ep loadbalancer.Endpoint, ok bool) {
	m.lock.RLock()
	ep, ok = m.eps[epid]
	m.lock.RUnlock()
	return
}

func (m *endpointsManager) ResetEndpoints(eps ...loadbalancer.Endpoint) {
	m.lock.Lock()
	defer m.lock.Unlock()

	maps.Clear(m.eps)
	maps.AddSlice(m.eps, eps, func(s loadbalancer.Endpoint) (string, *endpoint) {
		return s.ID(), newEndpoint(s)
	})
	m.updateEndpoints()
}

func (m *endpointsManager) UpsertEndpoints(eps ...loadbalancer.Endpoint) {
	m.lock.Lock()
	defer m.lock.Unlock()

	maps.AddSlice(m.eps, eps, func(s loadbalancer.Endpoint) (string, *endpoint) {
		return s.ID(), newEndpoint(s)
	})
	m.updateEndpoints()
}

func (m *endpointsManager) RemoveEndpoint(epid string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.eps[epid]; ok {
		delete(m.eps, epid)
		m.updateEndpoints()
	}

	return
}

func (m *endpointsManager) updateEndpoints() {
	oneps := loadbalancer.AcquireEndpoints(len(m.eps))
	alleps := loadbalancer.AcquireEndpoints(len(m.eps))
	offeps := loadbalancer.AcquireEndpoints(0)
	for _, ep := range m.eps {
		alleps = append(alleps, ep)
		switch ep.Status() {
		case loadbalancer.EndpointStatusOnline:
			oneps = append(oneps, ep.Endpoint)
		case loadbalancer.EndpointStatusOffline:
			offeps = append(offeps, ep.Endpoint)
		}
	}

	swapEndpoints(&m.oneps, oneps)   // For online
	swapEndpoints(&m.offeps, offeps) // For offline
	swapEndpoints(&m.alleps, alleps) // For all
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
