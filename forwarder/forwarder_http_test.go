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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xgfone/go-loadbalancer/balancer/retry"
	"github.com/xgfone/go-loadbalancer/balancer/roundrobin"
	"github.com/xgfone/go-loadbalancer/endpoints/httpep"
)

func testHandler(key string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(rw, key)
	})
}

func TestLoadBalancer(t *testing.T) {
	balancer := roundrobin.NewBalancer("")
	forwarder := NewForwarder("test", balancer)
	forwarder.SwapBalancer(retry.NewRetry(balancer, 0, 0))

	go func() {
		server := http.Server{Addr: "127.0.0.1:8101", Handler: testHandler("8101")}
		server.ListenAndServe()
	}()

	go func() {
		server := http.Server{Addr: "127.0.0.1:8102", Handler: testHandler("8102")}
		server.ListenAndServe()
	}()

	time.Sleep(time.Millisecond * 100)

	ep1, err := httpep.Config{
		URL: httpep.URL{Hostname: "www.example.com", IP: "127.0.0.1", Port: 8101},

		StaticWeight: 1,
	}.NewEndpoint()
	if err != nil {
		t.Fatal(err)
	}

	ep2, err := httpep.Config{
		URL: httpep.URL{Hostname: "www.example.com", IP: "127.0.0.1", Port: 8102},

		StaticWeight: 2,
	}.NewEndpoint()
	if err != nil {
		t.Fatal(err)
	}

	if url := ep1.Info().(httpep.Config).URL.String(); url != "http://127.0.0.1:8101" {
		t.Errorf("expect url '%s', but got '%s'", "http://127.0.0.1:8101", url)
	}
	if err := ep1.Check(context.Background()); err != nil {
		t.Errorf("health check failed: %s", err)
	}

	forwarder.ResetEndpoints(ep1, ep2)

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
	forwarder.ServeHTTP(rec, req)
	forwarder.ServeHTTP(rec, req)
	forwarder.ServeHTTP(rec, req)
	forwarder.ServeHTTP(rec, req)

	expects := []string{
		"8101",
		"8102",
		"8101",
		"8102",
		"",
	}
	results := strings.Split(rec.Body.String(), "\n")
	if len(expects) != len(results) {
		t.Errorf("expect %d lines, but got %d", len(expects), len(results))
	} else {
		for i, line := range results {
			if line != expects[i] {
				t.Errorf("%d line: expect '%s', but got '%s'", i, expects[i], line)
			}
		}
	}

	state := ep1.State()
	if state.Total != 2 {
		t.Errorf("expect %d total requests, but got %d", 4, state.Total)
	}
	if state.Success != 2 {
		t.Errorf("expect %d success requests, but got %d", 4, state.Total)
	}

	/// ------------------------------------------------------------------ ///

	forwarder.SetEndpointOnline(ep1.ID(), false)
	if ep, ok := forwarder.GetEndpoint(ep1.ID()); !ok || ep.Status().IsOnline() {
		t.Errorf("invalid the endpoint1 online status: online=%v, ok=%v", ep.Status().IsOnline(), ok)
	}
	if ep, ok := forwarder.GetEndpoint(ep2.ID()); !ok || !ep.Status().IsOnline() {
		t.Errorf("invalid the endpoint2 online status: online=%v, ok=%v", ep.Status().IsOnline(), ok)
	}

	rec.Body.Reset()
	forwarder.ServeHTTP(rec, req)
	forwarder.ServeHTTP(rec, req)
	expects = []string{
		"8102",
		"8102",
		"",
	}
	results = strings.Split(rec.Body.String(), "\n")
	if len(expects) != len(results) {
		t.Errorf("expect %d lines, but got %d", len(expects), len(results))
	} else {
		for i, line := range results {
			if line != expects[i] {
				t.Errorf("%d line: expect '%s', but got '%s'", i, expects[i], line)
			}
		}
	}

	/// ------------------------------------------------------------------ ///

	forwarder.SetEndpointOnline(ep2.ID(), false)
	if ep, ok := forwarder.GetEndpoint(ep2.ID()); !ok || ep.Status().IsOnline() {
		t.Errorf("invalid the endpoint2 online status: online=%v, ok=%v", ep.Status().IsOnline(), ok)
	}

	eps := forwarder.GetAllEndpoints()
	if len(eps) != 2 {
		t.Errorf("expect %d endpoints, but got %d", 2, len(eps))
	}
	for _, ep := range eps {
		id := ep.ID()
		switch id {
		case ep1.ID(), ep2.ID():
		default:
			t.Errorf("unknown endpoint id '%s'", id)
		}

		if ep.Status().IsOnline() {
			t.Errorf("expect endpoint '%s' online is false, but got true", id)
		}
	}

	rec = httptest.NewRecorder()
	forwarder.ServeHTTP(rec, req)
	if rec.Code != 503 {
		t.Errorf("unexpected response: statuscode=%d, body=%s", rec.Code, rec.Body.String())
	}
}
