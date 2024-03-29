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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/balancer/retry"
	"github.com/xgfone/go-loadbalancer/balancer/roundrobin"
	"github.com/xgfone/go-loadbalancer/httpx"
)

func testHandler(key string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(201)
		fmt.Fprint(rw, key)
	})
}

func TestLoadBalancer(t *testing.T) {
	ep1 := httpx.Config{Host: "127.0.0.1", Port: 8101, Weight: 1}.NewEndpoint()
	ep2 := httpx.Config{Host: "127.0.0.1", Port: 8102, Weight: 2}.NewEndpoint()

	discovery := loadbalancer.NewStatic(loadbalancer.Endpoints{ep1, ep2})
	forwarder := New("test", retry.New(roundrobin.NewBalancer(""), 0, 0), discovery)

	go func() {
		server := http.Server{Addr: "127.0.0.1:8101", Handler: testHandler("8101")}
		_ = server.ListenAndServe()
	}()

	go func() {
		server := http.Server{Addr: "127.0.0.1:8102", Handler: testHandler("8102")}
		_ = server.ListenAndServe()
	}()

	time.Sleep(time.Millisecond * 100)

	var results []string
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1", nil)

	rec := httptest.NewRecorder()
	forwarder.ServeHTTP(rec, req)
	results = append(results, rec.Body.String())
	if rec.Code != 201 {
		t.Errorf("expect status code 201, but got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	forwarder.ServeHTTP(rec, req)
	results = append(results, rec.Body.String())
	if rec.Code != 201 {
		t.Errorf("expect status code 201, but got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	forwarder.ServeHTTP(rec, req)
	results = append(results, rec.Body.String())
	if rec.Code != 201 {
		t.Errorf("expect status code 201, but got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	forwarder.ServeHTTP(rec, req)
	results = append(results, rec.Body.String())
	if rec.Code != 201 {
		t.Errorf("expect status code 201, but got %d", rec.Code)
	}

	expects := []string{
		"8101",
		"8102",
		"8101",
		"8102",
	}

	if len(expects) != len(results) {
		t.Errorf("expect %d lines, but got %d: %v", len(expects), len(results), results)
	} else {
		for i, line := range results {
			if line != expects[i] {
				t.Errorf("%d line: expect '%s', but got '%s'", i, expects[i], line)
			}
		}
	}

	if total := ep1.Total(); total != 2 {
		t.Errorf("expect %d total requests, but got %d", 2, total)
	}
}
