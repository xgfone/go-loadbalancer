// Copyright 2023xgfone
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

package sourceiphash

import (
	"context"
	"net/http"
	"testing"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/internal/tests"
)

func BenchmarkSourceIPHash(b *testing.B) {
	tests.BenchBalancer(b, NewBalancer(""))
}

func TestSourceIPHash(t *testing.T) {
	ep1 := tests.NewNoopEndpoint("127.0.0.1", 1)
	ep2 := tests.NewNoopEndpoint("127.0.0.2", 2)
	ep3 := tests.NewNoopEndpoint("127.0.0.3", 3)
	eps := loadbalancer.Endpoints{ep1, ep2, ep3}
	discovery := endpoint.NewStatic(eps)

	req1, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
	req2, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
	req3, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
	req1.RemoteAddr = "192.168.0.0"
	req2.RemoteAddr = "192.168.0.1"
	req3.RemoteAddr = "192.168.0.2"

	balancer := NewBalancer("")
	_, _ = balancer.Forward(context.Background(), req1, discovery)
	_, _ = balancer.Forward(context.Background(), req1, discovery)
	_, _ = balancer.Forward(context.Background(), req1, discovery)
	_, _ = balancer.Forward(context.Background(), req1, discovery)
	_, _ = balancer.Forward(context.Background(), req1, discovery)
	_, _ = balancer.Forward(context.Background(), req1, discovery)
	_, _ = balancer.Forward(context.Background(), req2, discovery)
	_, _ = balancer.Forward(context.Background(), req3, discovery)

	if total := ep1.Total(); total != 6 {
		t.Errorf("expect %d endpoint1, but got %d", 6, total)
	}
	if total := ep2.Total(); total != 1 {
		t.Errorf("expect %d endpoint2, but got %d", 1, total)
	}
	if total := ep3.Total(); total != 1 {
		t.Errorf("expect %d endpoint3, but got %d", 1, total)
	}
}
