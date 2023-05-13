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

package processor

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xgfone/go-defaults"
)

var _ ResponseProcessor = ResponseProcessorFunc(nil)

// ResponseProcessor is used to handle the response body.
type ResponseProcessor interface {
	ProcessError(ctx context.Context, pc Context, err error) error
	ProcessResponse(ctx context.Context, pc Context, resp *http.Response) error
}

// ResponseProcessorFunc is a function to implement the interface ResponseProcessor.
type ResponseProcessorFunc func(ctx context.Context, pc Context, resp *http.Response, err error) error

// ProcessError implements the interface ResponseProcessor#ProcessError.
func (f ResponseProcessorFunc) ProcessError(ctx context.Context, pc Context, err error) error {
	return f(ctx, pc, nil, err)
}

// ProcessResponse implements the interface ResponseProcessor#ProcessResponse.
func (f ResponseProcessorFunc) ProcessResponse(ctx context.Context, pc Context, resp *http.Response) error {
	return f(ctx, pc, resp, nil)
}

// ResponseBody wraps the response processor to process the response body just once.
func ResponseBody(p ResponseProcessor) ResponseProcessor {
	return ResponseProcessorFunc(func(ctx context.Context, pc Context, resp *http.Response, err error) error {
		if !defaults.HTTPIsResponded(ctx, pc.SrcRes, pc.SrcReq) {
			if err == nil {
				p.ProcessResponse(ctx, pc, resp)
			} else {
				p.ProcessError(ctx, pc, err)
			}
		} else if err == nil && pc.SrcRes != nil {
			panic(fmt.Errorf("re-respond for %s %s", pc.SrcReq.Method, pc.SrcReq.RequestURI))
		}
		return err
	})
}
