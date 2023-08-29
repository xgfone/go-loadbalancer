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

package tests

import (
	"context"
	"testing"
)

func TestNewEndpoint(t *testing.T) {
	ep := NewEndpoint("1.2.3.4", 1)
	ep.current = 0
	_, _ = ep.Serve(context.Background(), nil)
	_, _ = ep.Serve(context.Background(), nil)

	state := ep.State()
	if state.Total != 2 {
		t.Errorf("expect total=%d, but got %d", 2, state.Total)
	}
	if state.Success != 2 {
		t.Errorf("expect success=%d, but got %d", 2, state.Success)
	}
	if state.Current != 0 {
		t.Errorf("expect current=%d, but got %d", 0, state.Current)
	}
}
