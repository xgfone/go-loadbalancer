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

func TestEndpoints(t *testing.T) {
	ep1 := tests.NewEndpoint("192.168.1.1", 1)
	ep2 := tests.NewEndpoint("192.168.1.2", 1)
	ep3 := tests.NewEndpoint("192.168.1.3", 3)
	ep4 := tests.NewEndpoint("192.168.1.4", 3)
	ep5 := tests.NewEndpoint("192.168.1.5", 2)
	ep6 := tests.NewEndpoint("192.168.1.6", 2)

	eps := endpoint.Endpoints{ep1, ep2, ep3, ep4, ep5, ep6}
	endpoint.Sort(eps)

	expects := []string{
		"192.168.1.1", "192.168.1.2",
		"192.168.1.5", "192.168.1.6",
		"192.168.1.3", "192.168.1.4",
	}
	for i, ep := range eps {
		if id := ep.ID(); expects[i] != id {
			t.Errorf("expect the endpoint '%s', but got '%s'", expects[i], id)
		}
	}
}
