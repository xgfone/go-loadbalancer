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
	"net/http"
)

var (
	_ ResponseProcessor = ResponseProcessorFunc(nil)
	_ ResponseProcessor = ResponseProcessors(nil)
)

type (
	// Response represents a response to forward
	// the response from the backend endpoint to respsone writer.
	Response struct {
		SrcRes http.ResponseWriter
		SrcReq *http.Request
		DstReq *http.Request
		DstRes *http.Response
	}

	// ResponseProcessor is used to process the response
	// after receiving the response from the backend endpoint.
	ResponseProcessor interface {
		Process(context.Context, Response, error) error
	}

	// ResponseProcessorFunc is the response processor function.
	ResponseProcessorFunc func(context.Context, Response, error) error

	// ResponseProcessors represents a group of response processors.
	ResponseProcessors []ResponseProcessor
)

// NewResponse returns a new Response.
func NewResponse(srcres http.ResponseWriter, srcreq, dstreq *http.Request, dstres *http.Response) Response {
	return Response{
		SrcRes: srcres,
		SrcReq: srcreq,
		DstReq: dstreq,
		DstRes: dstres,
	}
}

// WithSrcRes returns a new Response with the http response writer as SrcRes.
func (r Response) WithSrcRes(rw http.ResponseWriter) Response {
	r.SrcRes = rw
	return r
}

// WithSrcReq returns a new Response with the http request as SrcReq.
func (r Response) WithSrcReq(req *http.Request) Response {
	r.SrcReq = req
	return r
}

// WithDstReq returns a new Response with the http request as DstReq.
func (r Response) WithDstReq(req *http.Request) Response {
	r.DstReq = req
	return r
}

// WithDstRes returns a new Response with the http response as DstRes.
func (r Response) WithDstRes(res *http.Response) Response {
	r.DstRes = res
	return r
}

// Process implements the interface ResponseProcessor.
//
// f may be nil, which is equal to do nothing and return the original error.
func (f ResponseProcessorFunc) Process(ctx context.Context, resp Response, err error) error {
	if f != nil {
		err = f(ctx, resp, err)
	}
	return err
}

// NoneResponseProcessor is equal to ResponseProcessor(nil).
func NoneResponseProcessor() ResponseProcessor { return ResponseProcessor(nil) }

// SimpleResponseProcessor converts a simple function to ResponseProcessor.
func SimpleResponseProcessor(f func(http.ResponseWriter, *http.Response, error) error) ResponseProcessor {
	return ResponseProcessorFunc(func(_ context.Context, resp Response, err error) error {
		return f(resp.SrcRes, resp.DstRes, err)
	})
}

// Process implements the interface ResponseProcessor.
func (ps ResponseProcessors) Process(ctx context.Context, resp Response, err error) error {
	for i, _len := 0, len(ps); i < _len; i++ {
		err = ps[i].Process(ctx, resp, err)
	}
	return err
}

// CompactResponseProcessors compacts a group of response processors,
// which will eliminate the empty processor, to a response processor.
func CompactResponseProcessors(processors ...ResponseProcessor) ResponseProcessors {
	_processors := make(ResponseProcessors, 0, len(processors))
	for _, processor := range processors {
		switch p := processor.(type) {
		case nil:
		case ResponseProcessorFunc:
			if p != nil {
				_processors = append(_processors, p)
			}
		case ResponseProcessors:
			if len(p) > 0 {
				_processors = append(_processors, CompactResponseProcessors(p...)...)
			}
		default:
			_processors = append(_processors, p)
		}
	}
	return _processors
}
