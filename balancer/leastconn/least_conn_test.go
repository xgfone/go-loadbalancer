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

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/internal/tests"
)

func BenchmarkLeastConn(b *testing.B) {
	tests.BenchBalancer(b, NewBalancer("", nil))
}

func TestLeastConn(t *testing.T) {
	ep1 := tests.NewNoopEndpoint("127.0.0.1", 1)
	ep2 := tests.NewNoopEndpoint("127.0.0.2", 2)
	ep3 := tests.NewNoopEndpoint("127.0.0.3", 3)
	eps := loadbalancer.Endpoints{ep1, ep2, ep3}
	discovery := loadbalancer.NewStatic(eps)

	balancer := NewBalancer("", nil)
	for i := 0; i < 10; i++ {
		_, _ = balancer.Forward(context.Background(), nil, discovery)
	}

	if total := ep1.Total(); total != 10 {
		t.Errorf("expect %d endpoint1, but got %d", 10, total)
	}
	if total := ep2.Total(); total != 0 {
		t.Errorf("expect %d endpoint2, but got %d", 0, total)
	}
	if total := ep3.Total(); total != 0 {
		t.Errorf("expect %d endpoint3, but got %d", 0, total)
	}
}
