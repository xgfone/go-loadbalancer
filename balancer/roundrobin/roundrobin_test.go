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

package roundrobin

import (
	"context"
	"testing"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/internal/tests"
)

func BenchmarkRoundRobin(b *testing.B) {
	tests.BenchBalancer(b, NewBalancer(""))
}

func BenchmarkWeightedRoundRobin(b *testing.B) {
	tests.BenchBalancer(b, NewWeightedBalancer(""))
}

func TestWeightedRoundRobin(t *testing.T) {
	ep1 := tests.NewEndpoint("127.0.0.1", 1)
	ep2 := tests.NewEndpoint("127.0.0.2", 2)
	ep3 := tests.NewEndpoint("127.0.0.3", 3)
	eps := loadbalancer.Endpoints{ep1, ep2, ep3}

	balancer := NewWeightedBalancer("")
	for i := 0; i < 18; i++ {
		balancer.Forward(context.Background(), nil, eps)
	}

	if state := ep1.State(); state.Total != 3 {
		t.Errorf("expect %d endpoint1, but got %d", 3, state.Total)
	}
	if state := ep2.State(); state.Total != 6 {
		t.Errorf("expect %d endpoint2, but got %d", 6, state.Total)
	}
	if state := ep3.State(); state.Total != 9 {
		t.Errorf("expect %d endpoint3, but got %d", 9, state.Total)
	}
}
