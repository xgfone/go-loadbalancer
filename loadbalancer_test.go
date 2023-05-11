// Copyright 2021~2023 xgfone
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

// Package loadbalancer provides some common functions.
package loadbalancer

import "testing"

func TestNewError(t *testing.T) {
	if err := NewError(true, nil); err != nil {
		t.Errorf("expect nil, but got %v", err)
	}

	switch err := NewError(true, ErrNoAvailableEndpoints).(type) {
	case retryError:
		if !err.Retry() {
			t.Errorf("expect retry is true, but got false")
		}
		if e := err.Unwrap(); e != ErrNoAvailableEndpoints {
			t.Errorf("expect error '%s', but got '%v'", ErrNoAvailableEndpoints, e)
		}

	default:
		t.Errorf("expect a retry error, but got %T", err)
	}
}
