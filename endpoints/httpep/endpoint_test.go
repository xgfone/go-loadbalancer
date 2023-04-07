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

package httpep

import (
	"sort"
	"testing"

	"github.com/xgfone/go-loadbalancer"
)

func newEndpoint(weight int, ip string, port uint16) loadbalancer.Endpoint {
	ep, err := Config{StaticWeight: weight, URL: URL{IP: ip, Port: port}}.NewEndpoint()
	if err != nil {
		panic(err)
	}
	return ep
}

func TestHTTPEndpoints(t *testing.T) {
	ep1 := newEndpoint(1, "127.0.0.1", 8001)
	ep2 := newEndpoint(1, "127.0.0.1", 8002)
	ep3 := newEndpoint(3, "127.0.0.1", 8003)
	ep4 := newEndpoint(3, "127.0.0.1", 8004)
	ep5 := newEndpoint(2, "127.0.0.1", 8005)
	ep6 := newEndpoint(2, "127.0.0.1", 8006)

	eps := loadbalancer.Endpoints{ep1, ep2, ep3, ep4, ep5, ep6}
	sort.Stable(eps)

	exports := []uint16{8001, 8002, 8005, 8006, 8003, 8004}
	for i, ep := range eps {
		if port := ep.Info().(Config).URL.Port; exports[i] != port {
			t.Errorf("expect the port '%d', but got '%d'", exports[i], port)
		}
	}
}
