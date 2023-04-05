// Copyright 2021~2023 xgfone
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

package forwarder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xgfone/go-loadbalancer/balancer/random"
	"github.com/xgfone/go-loadbalancer/internal/slog"
	"github.com/xgfone/go-loadbalancer/internal/tests"
)

func BenchmarkLoadBalancer(b *testing.B) {
	slog.SetDiscardWriter() // Discard log for test.
	lb := NewForwarder("test", random.NewBalancer(""))
	lb.ResetEndpoints(tests.NewEndpoint("127.0.0.1", 1), tests.NewEndpoint("127.0.0.2", 1))

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		lb.ServeHTTP(rec, req)
	}
}