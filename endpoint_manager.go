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

package loadbalancer

import (
	"sort"
	"sync"
	"sync/atomic"

	"github.com/xgfone/go-generics/maps"
)

var _ EndpointWrapper = new(endpoint)

type endpoint struct {
	Endpoint
	status atomic.Value
}

func newEndpoint(ep Endpoint) *endpoint {
	s := &endpoint{Endpoint: ep}
	s.status.Store(EndpointStatusOnline)
	return s
}

func (s *endpoint) Unwrap() Endpoint {
	return s.Endpoint
}

func (s *endpoint) Status() EndpointStatus {
	return s.status.Load().(EndpointStatus)
}

func (s *endpoint) SetStatus(status EndpointStatus) (ok bool) {
	return s.status.CompareAndSwap(s.Status(), status)
}

var _ EndpointDiscovery = new(EndpointManager)

// EndpointManager is used to manage a group of endpoints.
type EndpointManager struct {
	lock sync.RWMutex
	eps  map[string]*endpoint

	oneps  atomic.Value
	offeps atomic.Value
	alleps atomic.Value
}

// NewEndpointManager returns a new endpoint manager.
func NewEndpointManager() *EndpointManager {
	m := &EndpointManager{eps: make(map[string]*endpoint, 8)}
	m.alleps.Store(Endpoints{})
	m.offeps.Store(Endpoints{})
	m.oneps.Store(Endpoints{})
	return m
}

// OnlineNum implements the interface EndpointDiscovery#OnlineNum.
func (m *EndpointManager) OnlineNum() int {
	return len(m.OnEndpoints())
}

// OnEndpoints implements the interface EndpointDiscovery#OnEndpoints.
func (m *EndpointManager) OnEndpoints() Endpoints {
	return m.oneps.Load().(Endpoints)
}

// OffEndpoints implements the interface EndpointDiscovery#OffEndpoints.
func (m *EndpointManager) OffEndpoints() Endpoints {
	return m.offeps.Load().(Endpoints)
}

// AllEndpoints implements the interface EndpointDiscovery#AllEndpoints.
func (m *EndpointManager) AllEndpoints() Endpoints {
	return m.alleps.Load().(Endpoints)
}

// AllOriginEndpoints is the same as AllEndpoints,
// but returns the original endpoints instead.
func (m *EndpointManager) AllOriginEndpoints() Endpoints {
	endpoints := m.AllEndpoints()
	eps := make(Endpoints, len(endpoints))
	for i, ep := range endpoints {
		eps[i] = ep.(*endpoint)
	}
	return eps
}

// SetEndpointStatus sets the status of the endpoint.
//
// The default status of the endpoint is EndpointStatusOnline.
func (m *EndpointManager) SetEndpointStatus(epid string, status EndpointStatus) {
	m.lock.RLock()
	if ep, ok := m.eps[epid]; ok {
		if ep.SetStatus(status) {
			m.updateEndpoints()
		}
	}
	m.lock.RUnlock()
}

// SetEndpointStatuses sets the statuses of a group of endpoints.
func (m *EndpointManager) SetEndpointStatuses(epid2statuses map[string]EndpointStatus) {
	m.lock.RLock()
	var changed bool
	for epid, status := range epid2statuses {
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

// GetEndpoint returns the endpoint by the id.
func (m *EndpointManager) GetEndpoint(epid string) (ep Endpoint, ok bool) {
	m.lock.RLock()
	ep, ok = m.eps[epid]
	m.lock.RUnlock()
	return
}

// ResetEndpoints resets all the endpoints to the news.
func (m *EndpointManager) ResetEndpoints(news ...Endpoint) {
	m.lock.Lock()
	defer m.lock.Unlock()

	maps.Clear(m.eps)
	maps.AddSlice(m.eps, news, func(s Endpoint) (string, *endpoint) {
		return s.ID(), newEndpoint(s)
	})
	m.updateEndpoints()
}

// UpsertEndpoints adds or updates the endpoints.
func (m *EndpointManager) UpsertEndpoints(eps ...Endpoint) {
	if len(eps) == 0 {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	for _, ep := range eps {
		id := ep.ID()
		if _ep, ok := m.eps[id]; ok {
			_ep.Update(ep.Info())
		} else {
			m.eps[id] = newEndpoint(ep)
		}
	}
	m.updateEndpoints()
}

// RemoveEndpoint removes the endpoint by the id.
func (m *EndpointManager) RemoveEndpoint(epid string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.eps[epid]; ok {
		delete(m.eps, epid)
		m.updateEndpoints()
	}

	return
}

func (m *EndpointManager) updateEndpoints() {
	oneps := AcquireEndpoints(len(m.eps))
	alleps := AcquireEndpoints(len(m.eps))
	offeps := AcquireEndpoints(0)
	for _, ep := range m.eps {
		alleps = append(alleps, ep)
		switch ep.Status() {
		case EndpointStatusOnline:
			oneps = append(oneps, ep.Endpoint)
		case EndpointStatusOffline:
			offeps = append(offeps, ep.Endpoint)
		}
	}

	swapEndpoints(&m.oneps, oneps)   // For online
	swapEndpoints(&m.offeps, offeps) // For offline
	swapEndpoints(&m.alleps, alleps) // For all
}

func swapEndpoints(dsteps *atomic.Value, neweps Endpoints) {
	if len(neweps) == 0 {
		oldeps := dsteps.Swap(Endpoints{}).(Endpoints)
		ReleaseEndpoints(oldeps)
		ReleaseEndpoints(neweps)
	} else {
		sort.Stable(neweps)
		oldeps := dsteps.Swap(neweps).(Endpoints)
		ReleaseEndpoints(oldeps)
	}
}
