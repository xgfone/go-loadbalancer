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
	"sync"

	"github.com/xgfone/go-atomicvalue"
	"github.com/xgfone/go-generics/maps"
)

// DefaultManager is the default upstream manager.
var DefaultManager = NewManager()

// Manager is used to manage a group of upstreams.
type Manager struct {
	lock sync.RWMutex
	ups  map[string]*Upstream
	upm  atomicvalue.Value[map[string]*Upstream]

	add atomicvalue.Value[func(*Upstream)]
	del atomicvalue.Value[func(*Upstream)]
}

// NewManager returns a new upstream manager.
func NewManager() *Manager {
	m := &Manager{ups: make(map[string]*Upstream, 8)}
	m.OnAdd(nil)
	m.OnDel(nil)
	return m
}

func (m *Manager) updateUpstreams() { m.upm.Store(maps.Clone(m.ups)) }

// OnAdd sets the callback function, which is called when an upstream is added.
func (m *Manager) OnAdd(f func(*Upstream)) {
	if f == nil {
		f = onNothing
	}
	m.add.Store(f)
}

// OnDel sets the callback function, which is called when an upstream is deleted.
func (m *Manager) OnDel(f func(*Upstream)) {
	if f == nil {
		f = onNothing
	}
	m.del.Store(f)
}

func onNothing(*Upstream) {}

func (m *Manager) onadd(up *Upstream, ok bool) bool {
	if ok {
		m.add.Load()(up)
	}
	return ok
}

func (m *Manager) ondel(up *Upstream, ok bool) {
	if ok {
		m.del.Load()(up)
	}
}

func (m *Manager) addUpstream(up *Upstream) (ok bool) {
	m.lock.Lock()
	_, ok = m.ups[up.Name()]
	if ok = !ok; ok {
		m.ups[up.Name()] = up
		m.updateUpstreams()
	}
	m.lock.Unlock()
	return
}

// AddUpstream tries to add an upstream.
//
// If the upstream has existed, it does nothing and returns false.
func (m *Manager) AddUpstream(up *Upstream) (ok bool) {
	return m.onadd(up, m.addUpstream(up))
}

func (m *Manager) delUpstream(name string) *Upstream {
	m.lock.Lock()
	up, ok := maps.Pop(m.ups, name)
	if ok {
		m.updateUpstreams()
	}
	m.lock.Unlock()
	return up
}

// AddUpstream tries to delete an upstream by the name and return it.
//
// If the upstream does not exist, it does nothing and returns nil.
func (m *Manager) DeleteUpstream(name string) *Upstream {
	up := m.delUpstream(name)
	m.ondel(up, up != nil)
	return up
}

// UpdateUpstream updates an upstream with the options.
//
// If the upstream named name does not exist, it does nothing and return false.
func (m *Manager) UpdateUpstream(name string, options ...Option) (ok bool) {
	if len(options) == 0 {
		return false
	}

	up := m.GetUpstream(name)
	if ok = up != nil; ok {
		up.Update(options...)
		m.updateUpstreams()
	}

	return
}

// ResetUpstreams resets the upstreams.
func (m *Manager) ResetUpstreams(upstreams map[string][]Option) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Update or Add
	for name, options := range upstreams {
		if up, ok := m.ups[name]; ok {
			up.Update(options...)
		} else {
			up := NewUpstream(name, options...)
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

// GetUpstreams returns all the upstreams.
func (m *Manager) GetUpstreams() map[string]*Upstream {
	return maps.Clone(m.upm.Load())
}

// GetUpstream returns the upstream by the name.
func (m *Manager) GetUpstream(name string) *Upstream {
	return m.upm.Load()[name]
}
