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

package upstream

import (
	"maps"
	"sync"

	"github.com/xgfone/go-atomicvalue"
)

// DefaultManager is the default upstream manager.
var DefaultManager = NewManager()

// Manager is used to manage a group of upstreams.
type Manager struct {
	lock sync.RWMutex
	ups  map[string]*Upstream
	upm  atomicvalue.Value[map[string]*Upstream]

	addf atomicvalue.Value[func(*Upstream)]
	delf atomicvalue.Value[func(*Upstream)]
}

// NewManager returns a new upstream manager.
func NewManager() *Manager {
	return &Manager{
		ups:  make(map[string]*Upstream, 8),
		addf: atomicvalue.NewValue(onNothing),
		delf: atomicvalue.NewValue(onNothing),
	}
}

func (m *Manager) updateUpstreams() { m.upm.Store(maps.Clone(m.ups)) }

// OnAdd sets the callback function, which is called when an upstream is added.
func (m *Manager) OnAdd(f func(*Upstream)) {
	if f == nil {
		f = onNothing
	}
	m.addf.Store(f)
}

// OnDel sets the callback function, which is called when an upstream is deleted.
func (m *Manager) OnDel(f func(*Upstream)) {
	if f == nil {
		f = onNothing
	}
	m.delf.Store(f)
}

func onNothing(*Upstream) {}

func (m *Manager) onadd(up *Upstream, ok bool) bool {
	if ok {
		m.addf.Load()(up)
	}
	return ok
}

func (m *Manager) ondel(up *Upstream, ok bool) {
	if ok {
		m.delf.Load()(up)
	}
}

func (m *Manager) add(up *Upstream) (ok bool) {
	m.lock.Lock()
	_, ok = m.ups[up.Name()]
	if ok = !ok; ok {
		m.ups[up.Name()] = up
		m.updateUpstreams()
	}
	m.lock.Unlock()
	return
}

// Add tries to add an upstream.
//
// If the upstream has existed, it does nothing and returns false.
func (m *Manager) Add(up *Upstream) (ok bool) {
	return m.onadd(up, m.add(up))
}

func (m *Manager) del(name string) *Upstream {
	m.lock.Lock()
	up, ok := m.ups[name]
	if ok {
		delete(m.ups, name)
		m.updateUpstreams()
	}
	m.lock.Unlock()
	return up
}

// Delete tries to delete an upstream by the name and return it.
//
// If the upstream does not exist, it does nothing and returns nil.
func (m *Manager) Delete(name string) *Upstream {
	up := m.del(name)
	m.ondel(up, up != nil)
	return up
}

// Update updates an upstream with the options.
//
// If the upstream named name does not exist, it does nothing and return false.
func (m *Manager) Update(name string, options ...Option) (ok bool) {
	if len(options) == 0 {
		return false
	}

	up := m.Get(name)
	if ok = up != nil; ok {
		up.Update(options...)
		m.updateUpstreams()
	}

	return
}

// Reset resets the upstreams.
func (m *Manager) Reset(upstreams map[string][]Option) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Update or Add
	for name, options := range upstreams {
		if up, ok := m.ups[name]; ok {
			up.Update(options...)
		} else {
			up := New(name, options...)
			m.ups[name] = up
			m.onadd(up, true)
		}
	}

	// Delete
	for name, up := range m.ups {
		if _, ok := upstreams[name]; !ok {
			delete(m.ups, name)
			m.ondel(up, true)
		}
	}

	m.updateUpstreams()
}

// Gets returns all the upstreams, which is read-only.
func (m *Manager) Gets() map[string]*Upstream { return m.upm.Load() }

// GetUpstream returns the upstream by the name.
func (m *Manager) Get(name string) *Upstream { return m.upm.Load()[name] }
