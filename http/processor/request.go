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
	_ RequestProcessor = RequestProcessorFunc(nil)
	_ RequestProcessor = RequestProcessors(nil)
)

type (
	// Request represents a request to send to the backend endpoint.
	Request struct {
		Req *http.Request
	}

	// RequestProcessor is used to process the request
	// before sending the reqeust to the backend endpoint.
	RequestProcessor interface {
		Process(context.Context, Request)
	}

	// RequestProcessorFunc is the request processor function.
	RequestProcessorFunc func(context.Context, Request)

	// RequestProcessors represents a group of request processors.
	RequestProcessors []RequestProcessor
)

// NewRequest return a new Request.
func NewRequest(req *http.Request) Request { return Request{Req: req} }

// Process implements the interface RequestProcessor.
//
// f may be nil, which is equal to do nothing and return the original request.
func (f RequestProcessorFunc) Process(c context.Context, r Request) {
	if f != nil {
		f(c, r)
	}
}

// NoneRequestProcessor is equal to RequestProcessor(nil).
func NoneRequestProcessor() RequestProcessor { return RequestProcessor(nil) }

// SimpleRequestProcessor converts a simple function to RequestProcessor.
func SimpleRequestProcessor(f func(Request)) RequestProcessor {
	return RequestProcessorFunc(func(c context.Context, r Request) { f(r) })
}

// Process implements the interface RequestProcessor.
func (ps RequestProcessors) Process(c context.Context, r Request) {
	for i, _len := 0, len(ps); i < _len; i++ {
		ps[i].Process(c, r)
	}
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
