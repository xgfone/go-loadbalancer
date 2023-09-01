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

import "sync"

var (
	eppool4   = sync.Pool{New: func() any { return &Static{make(Endpoints, 0, 4)} }}
	eppool8   = sync.Pool{New: func() any { return &Static{make(Endpoints, 0, 8)} }}
	eppool16  = sync.Pool{New: func() any { return &Static{make(Endpoints, 0, 16)} }}
	eppool32  = sync.Pool{New: func() any { return &Static{make(Endpoints, 0, 32)} }}
	eppool64  = sync.Pool{New: func() any { return &Static{make(Endpoints, 0, 64)} }}
	eppool128 = sync.Pool{New: func() any { return &Static{make(Endpoints, 0, 128)} }}
)

// Acquire acquires a preallocated zero-length endpoints from the pool.
func Acquire(expectedMaxCap int) *Static {
	switch {
	case expectedMaxCap <= 4:
		return eppool4.Get().(*Static)

	case expectedMaxCap <= 8:
		return eppool8.Get().(*Static)

	case expectedMaxCap <= 16:
		return eppool16.Get().(*Static)

	case expectedMaxCap <= 32:
		return eppool32.Get().(*Static)

	case expectedMaxCap <= 64:
		return eppool64.Get().(*Static)

	default:
		return eppool128.Get().(*Static)
	}
}

// Release releases the static endpoints back into the pool.
func Release(eps *Static) {
	if eps == nil || len(eps.Endpoints) == 0 {
		return
	}

	clear(eps.Endpoints)
	eps.Endpoints = eps.Endpoints[:0]
	cap := cap(eps.Endpoints)
	switch {
	case cap < 8:
		eppool4.Put(eps)

	case cap < 16:
		eppool8.Put(eps)

	case cap < 32:
		eppool16.Put(eps)

	case cap < 64:
		eppool32.Put(eps)

	case cap < 128:
		eppool64.Put(eps)

	default:
		eppool128.Put(eps)
	}
}
