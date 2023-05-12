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
	"io"
	"net/http"

	"github.com/xgfone/go-defaults"
	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/http/endpoint"
	"github.com/xgfone/go-loadbalancer/http/processor"
	"github.com/xgfone/go-loadbalancer/internal/nets"
)

// ServeHTTP implements the interface http.Handler, which is eqaul to
// f.ForwardHTTP(r.Context(), w, r, nil, nil).
func (f *Forwarder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.ForwardHTTP(r.Context(), w, r, nil, nil)
}

// ForwardHTTP is the same as Serve, but only for http, which is a simple
// implementation and uses "http/endpoint".Context as the request context.
//
// It uses "github.com/xgfone/go-defaults".HTTPIsRespondedFunc to check
// whether the response is responded or not. And it may be overrided
// for the complex http.ResponseWriter.
func (f *Forwarder) ForwardHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request,
	reqProcessor processor.Processor, resProcessor processor.Processor) error {

	// 1. Create a new request.
	req := r.Clone(ctx)
	//req.URL.Host = "" // Dial to the backend http endpoint.
	req.RequestURI = "" // Pretend to be a client request.
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}

	// 2. Process the request.
	c := processor.NewContext(w, r, req, nil)
	if reqProcessor != nil {
		reqProcessor.Process(ctx, c, nil)
	}

	// 3. Handle the response processor.
	if resProcessor == nil {
		resProcessor = processor.Response(defaultRespBodyProcessor)
	}
	resProcessor = wrapResponseProcessor(resProcessor)

	// 3. Forward the request and handle the response.
	err := f.Serve(ctx, endpoint.NewContext(w, r, req).WithRespProcessor(resProcessor))
	if err != nil {
		err = resProcessor.Process(ctx, c, err)
	}

	return err
}

func wrapResponseProcessor(p processor.Processor) processor.Processor {
	return processor.ProcessorFunc(func(ctx context.Context, c processor.Context, err error) error {
		if !defaults.HTTPIsResponded(ctx, c.SrcRes, c.SrcReq) {
			err = p.Process(ctx, c, err)
		} else if err == nil && c.SrcRes != nil {
			panic(fmt.Errorf("re-respond for %s %s", c.SrcReq.Method, c.SrcReq.RequestURI))
		}
		return err
	})
}

func defaultRespBodyProcessor(w http.ResponseWriter, r *http.Response, err error) error {
	switch {
	case err == nil:
		err = endpoint.HandleResponseBody(w, r) // Success

	case err == loadbalancer.ErrNoAvailableEndpoints:
		w.WriteHeader(503) // Service Unavailable

	case nets.IsTimeout(err):
		w.WriteHeader(504) // Gateway Timeout

	default:
		if h, ok := err.(http.Handler); ok {
			h.ServeHTTP(w, nil)
		} else {
			w.WriteHeader(502) // Bad Gateway
			_, err = io.WriteString(w, err.Error())
		}
	}
	return err
}
