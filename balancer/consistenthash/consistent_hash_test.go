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

package consistenthash

import (
	"context"
	"encoding/binary"
	"net"
	"net/http"
	"testing"

	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/endpoint/extep"
)

func TestBalancer(t *testing.T) {
	eps := endpoint.Endpoints{
		extep.NewStateEndpoint(endpoint.Noop("1.2.3.4:8000", 1)),
		extep.NewStateEndpoint(endpoint.Noop("1.2.3.4:8080", 1)),
		extep.NewStateEndpoint(endpoint.Noop("5.6.7.8:8000", 1)),
		extep.NewStateEndpoint(endpoint.Noop("5.6.7.8:8080", 1)),
	}

	balancer := NewBalancer("chash_ip", func(req interface{}) int {
		ip, _, err := net.SplitHostPort(req.(*http.Request).URL.Host)
		if err != nil {
			panic(err)
		}
		return int(binary.BigEndian.Uint64(net.ParseIP(ip)[8:16]))
	})

	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:80", nil)
	for i := 0; i < 8; i++ {
		_, err := balancer.Forward(context.TODO(), req, eps)
		if err != nil {
			t.Fatal(err)
		}
	}

	var num int
	counts := make([]uint64, len(eps))
	for i := 0; i < len(eps); i++ {
		total := extep.GetState(eps[i]).Total
		counts[i] = total
		if total > 0 {
			num++
		}
	}

	if num != 1 {
		t.Errorf("expect %d endpoint, but got %d", 1, num)
	}
}
