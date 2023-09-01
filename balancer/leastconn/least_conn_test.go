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

package leastconn

import (
	"context"
	"testing"

	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/endpoint/extep"
	"github.com/xgfone/go-loadbalancer/internal/tests"
)

func BenchmarkLeastConn(b *testing.B) {
	tests.BenchBalancer(b, NewBalancer(""))
}

func TestLeastConn(t *testing.T) {
	ep1 := extep.NewStateEndpoint(endpoint.Noop("127.0.0.1", 1))
	ep2 := extep.NewStateEndpoint(endpoint.Noop("127.0.0.2", 2))
	ep3 := extep.NewStateEndpoint(endpoint.Noop("127.0.0.3", 3))
	eps := endpoint.Endpoints{ep1, ep2, ep3}

	balancer := NewBalancer("")
	for i := 0; i < 10; i++ {
		_, _ = balancer.Forward(context.Background(), nil, eps)
	}

	if state := extep.GetState(ep1); state.Total != 10 {
		t.Errorf("expect %d endpoint1, but got %d", 10, state.Total)
	}
	if state := extep.GetState(ep2); state.Total != 0 {
		t.Errorf("expect %d endpoint2, but got %d", 0, state.Total)
	}
	if state := extep.GetState(ep3); state.Total != 0 {
		t.Errorf("expect %d endpoint3, but got %d", 0, state.Total)
	}
}
