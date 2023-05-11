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

import "sync/atomic"

// State is the runtime state of an endpoint, which can be inlined
// into other struct.
type State struct {
	Total   uint64 // The total number to handle all the requests.
	Success uint64 // The total number to handle the requests successfully.
	Current uint64 // The number of the requests that are being handled.

	// For the extra runtime information.
	Extra interface{} `json:",omitempty" xml:",omitempty"`
}

// IncSuccess increases the success state.
func (rs *State) IncSuccess() {
	atomic.AddUint64(&rs.Success, 1)
}

// Inc increases the total and current state.
func (rs *State) Inc() {
	atomic.AddUint64(&rs.Total, 1)
	atomic.AddUint64(&rs.Current, 1)
}

// Dec decreases the current state.
func (rs *State) Dec() {
	atomic.AddUint64(&rs.Current, ^uint64(0))
}

// Clone clones itself to a new one.
//
// If Extra has implemented the interface { Clone() interface{} }, call it//
// to clone the field Extra.
func (rs *State) Clone() State {
	extra := rs.Extra
	if clone, ok := rs.Extra.(interface{ Clone() interface{} }); ok {
		extra = clone.Clone()
	}

	return State{
		Extra:   extra,
		Total:   atomic.LoadUint64(&rs.Total),
		Success: atomic.LoadUint64(&rs.Success),
		Current: atomic.LoadUint64(&rs.Current),
	}
}
