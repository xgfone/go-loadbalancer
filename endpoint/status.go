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
	StatusOnline  Status = "on"
	StatusOffline Status = "off"
)

// Status represents the endpoint status.
type Status string

// IsOnline reports whether the endpoint status is online.
func (s Status) IsOnline() bool { return s == "on" }

// IsOffline reports whether the endpoint status is offline.
func (s Status) IsOffline() bool { return s == "off" }

// StatusManager is used to manage the status of the endpoint thread-safely
// which can be inlined into other struct.
type StatusManager struct {
	value atomicvalue.Value[Status]
}

// Get returns the current status.
func (s *StatusManager) Get() Status {
	return s.value.Load()
}

// Set resets the current status to new.
func (s *StatusManager) Set(new Status) {
	s.value.Store(new)
}

// Update is the same as Set, but reports whether the status is changed.
func (s *StatusManager) Update(new Status) bool {
	return s.value.CompareAndSwap(s.Get(), new)
}

// Status is equal to Get, but used to implement the interface Endpoint#Status.
func (s *StatusManager) Status() Status {
	return s.value.Load()
}

// SetStatus is equal to Set, but used implement the interface Endpoint#SetStatus.
func (s *StatusManager) SetStatus(new Status) {
	s.value.Store(new)
}
