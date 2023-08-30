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

import "github.com/xgfone/go-atomicvalue"

// Pre-define some endpoint statuses.
const (
	StatusOnline  = "on"
	StatusOffline = "off"
)

// IsOnline reports whether the status of the endpoint is online or not.
func IsOnline(ep Endpoint) bool { return ep != nil && ep.Status() == StatusOnline }

// IsOffline reports whether the status of the endpoint is offline or not.
func IsOffline(ep Endpoint) bool { return ep != nil && ep.Status() == StatusOffline }

// StatusManager is used to manage the status of the endpoint thread-safely
// which can be inlined into other struct.
type StatusManager struct{ value atomicvalue.Value[string] }

// Get returns the current status.
func (s *StatusManager) Get() string { return s.value.Load() }

// Set resets the current status to new.
func (s *StatusManager) Set(new string) { s.value.Store(new) }

// Update is the same as Set, but reports whether the status is changed.
func (s *StatusManager) Update(new string) bool { return s.value.CompareAndSwap(s.Get(), new) }

// Status is equal to Get, but used to implement the interface Endpoint#Status.
func (s *StatusManager) Status() string { return s.value.Load() }

// SetStatus is equal to Set, but used implement the interface Endpoint#SetStatus.
func (s *StatusManager) SetStatus(new string) { s.value.Store(new) }
