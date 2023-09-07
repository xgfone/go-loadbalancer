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

package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"K1": []string{"v1"}, "K2": []string{"v2"}},
		Body:       io.NopCloser(strings.NewReader("test1")),
	}

	rec := httptest.NewRecorder()
	if err := HandleResponse(rec, resp, nil); err != nil {
		t.Error(err)
	} else {
		if v := rec.Header().Get("K1"); v != "v1" {
			t.Errorf("expect header '%s' value '%s', but got '%s'", "K1", "v1", v)
		}
		if v := rec.Header().Get("K2"); v != "v2" {
			t.Errorf("expect header '%s' value '%s', but got '%s'", "K2", "v2", v)
		}
		if s := rec.Body.String(); s != "test1" {
			t.Errorf("expect response body '%s', but got '%s'", "test1", s)
		}
	}

	resp.Body = io.NopCloser(strings.NewReader("test2"))
	rec = httptest.NewRecorder()
	cb := func(w http.ResponseWriter) { w.Header().Del("K1") }
	if err := HandleResponse(rec, resp, cb); err != nil {
		t.Error(err)
	} else {
		if v := rec.Header().Get("K2"); v != "v2" {
			t.Errorf("expect header '%s' value '%s', but got '%s'", "K2", "v2", v)
		}
		if s := rec.Body.String(); s != "test2" {
			t.Errorf("expect response body '%s', but got '%s'", "test2", s)
		}
	}
}
