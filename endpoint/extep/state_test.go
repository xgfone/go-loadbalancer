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

package extep

import "testing"

func TestState(t *testing.T) {
	s := new(State)

	s.Inc()
	s.Inc()
	if s.Total != 2 {
		t.Errorf("expect total %d, but got %d", 2, s.Total)
	}
	if s.Current != 2 {
		t.Errorf("expect current %d, but got %d", 2, s.Current)
	}
	if s.Failure != 0 {
		t.Errorf("expect current %d, but got %d", 0, s.Failure)
	}

	s.Dec()
	if s.Total != 2 {
		t.Errorf("expect total %d, but got %d", 2, s.Total)
	}
	if s.Current != 1 {
		t.Errorf("expect current %d, but got %d", 1, s.Current)
	}
	if s.Failure != 0 {
		t.Errorf("expect current %d, but got %d", 0, s.Failure)
	}

	s.IncFailure()
	if s.Total != 2 {
		t.Errorf("expect total %d, but got %d", 2, s.Total)
	}
	if s.Current != 1 {
		t.Errorf("expect current %d, but got %d", 1, s.Current)
	}
	if s.Failure != 1 {
		t.Errorf("expect current %d, but got %d", 1, s.Failure)
	}
}
