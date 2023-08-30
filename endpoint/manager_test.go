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

package endpoint_test

import (
	"testing"

	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/internal/tests"
)

func TestManager(t *testing.T) {
	m := endpoint.NewManager(4)

	m.UpsertEndpoints(
		tests.NewEndpoint("1.2.3.4", 1),
		tests.NewEndpoint("1.2.3.5", 1),
		tests.NewEndpoint("1.2.3.6", 1),
	)

	if num := m.Number(); num != 3 {
		t.Errorf("expect %d, but got %d", 3, num)
	}

	m.SetEndpointStatus("1.2.3.4", endpoint.StatusOffline)
	if num := m.Number(); num != 2 {
		t.Errorf("expect %d, but got %d: %v", 2, num, m.OnEndpoints())
	}

	m.SetEndpointStatus("1.2.3.5", endpoint.StatusOffline)
	if num := m.Number(); num != 1 {
		t.Errorf("expect %d, but got %d", 1, num)
	}

	m.SetEndpointStatus("1.2.3.6", endpoint.StatusOffline)
	if num := m.Number(); num != 0 {
		t.Errorf("expect %d, but got %d", 0, num)
	}

	for _, ep := range m.AllEndpoints() {
		if ep.Status() != endpoint.StatusOffline {
			t.Errorf("expect endpoint status %s", ep.Status())
		}
	}
}
