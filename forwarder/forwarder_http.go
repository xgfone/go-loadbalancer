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
	"github.com/xgfone/go-loadbalancer/http/endpoint"
	"github.com/xgfone/go-loadbalancer/http/processor"
	"github.com/xgfone/go-loadbalancer/internal/nets"
)

// ServeHTTP implements the interface http.Handler, which is eqaul to
// f.ForwardHTTP(r.Context(), w, r, nil, nil, nil).
func (f *Forwarder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.ForwardHTTP(r.Context(), w, r, nil, nil, nil)
}

// ForwardHTTP is the same as Serve, but only for http, which is a simple
// implementation and uses "http/endpoint".Context as the request context.
//
// It uses "github.com/xgfone/go-defaults".HTTPIsRespondedFunc to check
// whether the response is responded or not. And it may be overrided
// for the complex http.ResponseWriter.
func (f *Forwarder) ForwardHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request,
	reqProcessor processor.Processor, resHeaderProcessor processor.Processor,
	resBodyProcessor processor.ResponseProcessor) error {

	// 1. Create a new request.
	req := r.Clone(ctx)
	//req.URL.Host = "" // Dial to the backend http endpoint.
	req.RequestURI = "" // Pretend to be a client request.
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}

	// 2. Process the request.
	c := processor.NewContext(w, r, req)
	if reqProcessor != nil {
		reqProcessor.Process(ctx, c)
	}

	// 3. Handle the response body processor.
	if resBodyProcessor == nil {
		resBodyProcessor = processor.ResponseProcessorFunc(defaultRespBodyProcessor)
	}

	// 3. Forward the request and handle the response.
	err := f.Serve(ctx, endpoint.NewContext(w, r, req).WithResponseProcessor(resBodyProcessor))
	if err != nil {
		err = resBodyProcessor.ProcessError(ctx, c, err)
	}

	return err
}

func defaultRespBodyProcessor(_ context.Context, pc processor.Context, r *http.Response, err error) error {
	switch {
	case err == nil:
		err = endpoint.HandleResponseBody(pc.SrcRes, r) // Success

	case err == loadbalancer.ErrNoAvailableEndpoints:
		pc.SrcRes.WriteHeader(503) // Service Unavailable

	case nets.IsTimeout(err):
		pc.SrcRes.WriteHeader(504) // Gateway Timeout

	default:
		if h, ok := err.(http.Handler); ok {
			h.ServeHTTP(pc.SrcRes, nil)
		} else {
			pc.SrcRes.WriteHeader(502) // Bad Gateway
			_, err = io.WriteString(pc.SrcRes, err.Error())
		}
	}
	return err
}
