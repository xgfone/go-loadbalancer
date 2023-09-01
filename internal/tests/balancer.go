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
	"net/http"
	"testing"

	"github.com/xgfone/go-loadbalancer/endpoint"
)

type forwarder interface {
	Forward(context.Context, interface{}, endpoint.Discovery) (interface{}, error)
}

// BenchBalancer benchmarks the forwarder.
func BenchBalancer(b *testing.B, forwarder forwarder) {
	eps := endpoint.Endpoints{
		endpoint.Noop("127.0.0.1", 1),
		endpoint.Noop("127.0.0.2", 2),
		endpoint.Noop("127.0.0.3", 3),
		endpoint.Noop("127.0.0.4", 4),
		endpoint.Noop("127.0.0.5", 5),
		endpoint.Noop("127.0.0.6", 6),
		endpoint.Noop("127.0.0.7", 7),
		endpoint.Noop("127.0.0.8", 8),
	}

	endpoint.Release(endpoint.Acquire(len(eps)))
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
	req.RemoteAddr = "127.0.0.1"

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			_, _ = forwarder.Forward(context.Background(), req, eps)
		}
	})
}
