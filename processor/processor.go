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

// Package processor provides some common request and response processors.
package processor

import (
	"context"
	"net/http"
)

// RequestProcessor is used to process the request
// before sending the reqeust to the backend endpoint.
type RequestProcessor interface {
	ProcessRequest(context.Context, *http.Request)
}

// ResponseProcessor is used to process the response
// after receiving the response from the backend endpoint.
type ResponseProcessor interface {
	ProcessResponse(context.Context, http.ResponseWriter, *http.Request, *http.Response, error) error
}

type (
	// RequestProcessorFunc is the request processor function.
	RequestProcessorFunc func(context.Context, *http.Request)

	// ResponseProcessorFunc is the response processor function.
	ResponseProcessorFunc func(context.Context, http.ResponseWriter, *http.Request, *http.Response, error) error
)

var (
	_ RequestProcessor  = RequestProcessorFunc(nil)
	_ ResponseProcessor = ResponseProcessorFunc(nil)
)

// ProcessRequest implements the interface RequestProcessor.
//
// f may be nil, which is equal to do nothing and return the original request.
func (f RequestProcessorFunc) ProcessRequest(c context.Context, r *http.Request) {
	if f != nil {
		f(c, r)
	}
}

// ProcessResponse implements the interface ResponseProcessor.
//
// f may be nil, which is equal to do nothing and return the original error.
func (f ResponseProcessorFunc) ProcessResponse(c context.Context, w http.ResponseWriter, req *http.Request, res *http.Response, err error) error {
	if f != nil {
		err = f(c, w, req, res, err)
	}
	return err
}

// NoneRequestProcessor is equal to RequestProcessor(nil).
func NoneRequestProcessor() RequestProcessor { return RequestProcessor(nil) }

// NoneResponseProcessor is equal to ResponseProcessor(nil).
func NoneResponseProcessor() ResponseProcessor { return ResponseProcessor(nil) }

// SimpleRequestProcessor converts a simple function to RequestProcessor.
func SimpleRequestProcessor(f func(*http.Request)) RequestProcessor {
	return RequestProcessorFunc(func(c context.Context, r *http.Request) { f(r) })
}

// SimpleResponseProcessor converts a simple function to ResponseProcessor.
func SimpleResponseProcessor(f func(http.ResponseWriter, *http.Response, error) error) ResponseProcessor {
	return ResponseProcessorFunc(func(_ context.Context, w http.ResponseWriter, _ *http.Request, r *http.Response, e error) error {
		return f(w, r, e)
	})
}

var (
	_ RequestProcessor  = RequestProcessors(nil)
	_ ResponseProcessor = ResponseProcessors(nil)
)

type (
	// RequestProcessors represents a group of request processors.
	RequestProcessors []RequestProcessor

	// ResponseProcessors represents a group of response processors.
	ResponseProcessors []ResponseProcessor
)

// ProcessRequest implements the interface RequestProcessor.
func (ps RequestProcessors) ProcessRequest(c context.Context, r *http.Request) {
	for i, _len := 0, len(ps); i < _len; i++ {
		ps[i].ProcessRequest(c, r)
	}
}

// ProcessResponse implements the interface ResponseProcessor.
func (ps ResponseProcessors) ProcessResponse(c context.Context, w http.ResponseWriter, req *http.Request, res *http.Response, err error) error {
	for i, _len := 0, len(ps); i < _len; i++ {
		err = ps[i].ProcessResponse(c, w, req, res, err)
	}
	return err
}

// CompactRequestProcessors compacts a group of request processors,
// which will eliminate the empty processor, to a request processor.
func CompactRequestProcessors(processors ...RequestProcessor) RequestProcessors {
	_processors := make(RequestProcessors, 0, len(processors))
	for _, processor := range processors {
		switch p := processor.(type) {
		case nil:
		case RequestProcessorFunc:
			if p != nil {
				_processors = append(_processors, p)
			}
		case RequestProcessors:
			if len(p) > 0 {
				_processors = append(_processors, CompactRequestProcessors(p...)...)
			}
		default:
			_processors = append(_processors, p)
		}
	}
	return _processors
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
