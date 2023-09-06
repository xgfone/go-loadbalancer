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
	"maps"
	"sync"

	"github.com/xgfone/go-atomicvalue"
)

// DefaultManager is the default loadbalancer manager.
var DefaultManager = NewManager()

// Manager is used to manage a group of loadbalancers.
type Manager struct {
	lock sync.RWMutex
	lbs  map[string]LoadBalancer
	lbm  atomicvalue.Value[map[string]LoadBalancer]

	addf atomicvalue.Value[func(string, LoadBalancer)]
	delf atomicvalue.Value[func(string, LoadBalancer)]
}

// NewManager returns a new loadbalancer manager.
func NewManager() *Manager {
	m := &Manager{lbs: make(map[string]LoadBalancer, 8)}
	m.OnAdd(nil)
	m.OnDel(nil)
	return m
}

func (m *Manager) update() { m.lbm.Store(maps.Clone(m.lbs)) }

func onNothing(string, LoadBalancer) {}

// OnAdd sets the callback function, which is called when a loadbalancer is added.
func (m *Manager) OnAdd(f func(name string, lb LoadBalancer)) {
	if f == nil {
		f = onNothing
	}
	m.addf.Store(f)
}

// OnDel sets the callback function, which is called when a loadbalancer is deleted.
func (m *Manager) OnDel(f func(name string, lb LoadBalancer)) {
	if f == nil {
		f = onNothing
	}
	m.delf.Store(f)
}

func (m *Manager) onadd(name string, lb LoadBalancer, ok bool) bool {
	if ok {
		m.addf.Load()(name, lb)
	}
	return ok
}

func (m *Manager) ondel(name string, lb LoadBalancer, ok bool) {
	if ok {
		m.delf.Load()(name, lb)
	}
}

func (m *Manager) add(name string, f LoadBalancer) (ok bool) {
	m.lock.Lock()
	_, ok = m.lbs[name]
	if ok = !ok; ok {
		m.lbs[name] = f
		m.update()
	}
	m.lock.Unlock()
	return
}

// Add tries to add a loadbalancer.
//
// If exists, do nothing and returns false.
func (m *Manager) Add(name string, lb LoadBalancer) (ok bool) {
	return m.onadd(name, lb, m.add(name, lb))
}

func (m *Manager) del(name string) LoadBalancer {
	m.lock.Lock()
	f, ok := m.lbs[name]
	if ok {
		delete(m.lbs, name)
		m.update()
	}
	m.lock.Unlock()
	return f
}

// Del tries to delete a loadbalancer by the name and return it.
//
// If not exist, do nothing and returns nil.
func (m *Manager) Del(name string) LoadBalancer {
	f := m.del(name)
	m.ondel(name, f, f != nil)
	return f
}

// Clear clears and deletes all the added loadbalancers.
func (m *Manager) Clear() {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.lbs) == 0 {
		return
	}

	defer m.update()
	for name, lb := range m.lbs {
		delete(m.lbs, name)
		m.ondel(name, lb, true)
	}
}

// Get returns the loadbalancer by the name.
//
// If not exist, return nil.
func (m *Manager) Get(name string) LoadBalancer { return m.lbm.Load()[name] }

// Gets returns all the loadbalancers, which is read-only.
func (m *Manager) Gets() map[string]LoadBalancer { return m.lbm.Load() }
