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
	"io"
	"net/http"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/http/processor"
	"github.com/xgfone/go-loadbalancer/internal/nets"
)

// ServeHTTP implements the interface http.Handler.
func (f *Forwarder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := f.ForwardHTTP(r.Context(), w, r, nil)
	switch {
	case err == nil:
		if resp != nil { // Success
			resp := resp.(*http.Response)
			defer resp.Body.Close()
			processor.HandleResponse(w, resp)
		}

	case err == loadbalancer.ErrNoAvailableEndpoints:
		w.WriteHeader(503) // Service Unavailable

	case nets.IsTimeout(err):
		w.WriteHeader(504) // Gateway Timeout

	default:
		if h, ok := err.(http.Handler); ok {
			h.ServeHTTP(w, nil)
		} else {
			w.WriteHeader(502) // Bad Gateway
			io.WriteString(w, err.Error())
		}
	}
}

// ForwardHTTP is the same as Serve, but just a simple implementation for http
func (f *Forwarder) ForwardHTTP(ctx context.Context, w http.ResponseWriter,
	r *http.Request, reqProcessor processor.Processor) (interface{}, error) {
	// 1. Create a new request.
	req := r.Clone(ctx)
	req.Close = false
	req.URL.User = nil
	req.Header.Del("Connection")
	//req.URL.Host = "" // Dial to the backend http endpoint.
	req.RequestURI = "" // Pretend to be a client request.
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}

	// 2. Process the request.
	if reqProcessor != nil {
		err := reqProcessor.Process(ctx, processor.NewContext(w, r, req))
		if err != nil {
			return nil, err
		}
	}

	// 3. Forward the request.
	return f.Serve(ctx, req)
}
