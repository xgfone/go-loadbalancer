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

import "sync/atomic"

// Discovery is used to discover the endpoints.
type Discovery interface {
	Discover() *Static
}

var _ Discovery = DiscoveryFunc(nil)

// DiscoveryFunc is a discovery function.
type DiscoveryFunc func() *Static

// Discover implements the interface Discovery.
func (f DiscoveryFunc) Discover() *Static { return f() }

// Static is used to wrap a set of endpoints.
type Static struct{ Endpoints }

var _ Discovery = new(Static)

// None represents a static without endpoints.
var None = new(Static)

// NewStatic returns a new static with the endpoints.
func NewStatic(eps Endpoints) *Static { return &Static{Endpoints: eps} }

// NewStaticWithCap returns a new static with the 0-len and n-cap endpoints.
func NewStaticWithCap(n int) *Static { return NewStatic(make(Endpoints, 0, n)) }

// Discover returns itself, which implements the interface Discovery.
func (s *Static) Discover() *Static { return s }

// Len returns the number of the endpoints.
func (s *Static) Len() int { return len(s.Endpoints) }

// Append appends the endpoints.
func (s *Static) Append(eps ...Endpoint) {
	s.Endpoints = append(s.Endpoints, eps...)
}

// Discover implements the interface Discovery,
// which is just used to test in general.
func (eps Endpoints) Discover() *Static { return &Static{Endpoints: eps} }

var _ Discovery = Endpoints(nil)

var _ Discovery = new(AtomicStatic)

// AtomicStatic is a atomic static discovery.
type AtomicStatic struct {
	static atomic.Pointer[Static]
}

// NewAtomicStatic a new atomic static.
func NewAtomicStatic(static *Static) *AtomicStatic {
	s := new(AtomicStatic)
	s.Set(static)
	return s
}

// Discover implements the interface Discovery.
func (s *AtomicStatic) Discover() *Static {
	return s.static.Load()
}

// Swap swap the old static with the new.
func (s *AtomicStatic) Swap(new *Static) (old *Static) {
	return s.static.Swap(new)
}

// Set sets the static to new.
func (s *AtomicStatic) Set(new *Static) {
	s.static.Store(new)
}
