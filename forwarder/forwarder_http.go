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
	"errors"
	"io"
	"net/http"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/httpx"
)

type timeoutError interface {
	Timeout() bool // Is the error a timeout?
	error
}

// IsTimeout reports whether the error is timeout.
func isTimeout(err error) bool {
	var timeoutErr timeoutError
	return errors.As(err, &timeoutErr) && timeoutErr.Timeout()
}

// ServeHTTP implements the interface http.Handler,
// which just forwards the request to one of the backend endpoints.
func (f *Forwarder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := f.forward(w, r)
	switch {
	case err == nil:
		if resp != nil { // Success
			resp := resp.(*http.Response)
			defer resp.Body.Close()
			_ = httpx.HandleResponse(w, resp, nil)
		}

	case err == loadbalancer.ErrNoAvailableEndpoints:
		w.WriteHeader(503) // Service Unavailable

	case isTimeout(err):
		w.WriteHeader(504) // Gateway Timeout

	default:
		if h, ok := err.(http.Handler); ok {
			h.ServeHTTP(w, nil)
		} else {
			w.WriteHeader(502) // Bad Gateway
			_, _ = io.WriteString(w, err.Error())
		}
	}
}

func (f *Forwarder) forward(_ http.ResponseWriter, r *http.Request) (any, error) {
	req := r.Clone(r.Context())
	req.Close = false
	req.URL.User = nil
	req.Header.Del("Connection")
	//req.URL.Host = "" // Dial to the backend http endpoint.
	req.RequestURI = "" // Pretend to be a client request.
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	return f.Serve(req.Context(), req)
}
