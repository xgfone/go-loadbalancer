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

var (
	_ Processor = ProcessorFunc(nil)
	_ Processor = Processors(nil)
)

type (
	// Context represents a processor context.
	Context struct {
		SrcRes http.ResponseWriter
		SrcReq *http.Request
		DstReq *http.Request
	}

	// Processor is used to process the request or response.
	Processor interface {
		Process(context.Context, Context) error
	}

	// ExtProcessor is an extended processor.
	ExtProcessor interface {
		String() string
		Type() string
		Processor
	}

	// ProcessorFunc is the processor function.
	ProcessorFunc func(ctx context.Context, pc Context) error

	// Processors represents a group of processors.
	Processors []Processor
)

// WithSrcRes returns a new processor Context with the http response writer as SrcRes.
func (c Context) WithSrcRes(rw http.ResponseWriter) Context {
	c.SrcRes = rw
	return c
}

// WithSrcReq returns a new processor Context with the http request as SrcReq.
func (c Context) WithSrcReq(req *http.Request) Context {
	c.SrcReq = req
	return c
}

// WithDstReq returns a new processor Context with the http request as DstReq.
func (c Context) WithDstReq(req *http.Request) Context {
	c.DstReq = req
	return c
}

// Process implements the interface Processor.
//
// f may be nil, which is equal to do nothing and return nil.
func (f ProcessorFunc) Process(ctx context.Context, pc Context) error {
	if f == nil {
		return nil
	}
	return f(ctx, pc)
}

// Process implements the interface Processor.
func (ps Processors) Process(ctx context.Context, pc Context) (err error) {
	for i, _len := 0, len(ps); i < _len; i++ {
		if err = ps[i].Process(ctx, pc); err != nil {
			return
		}
	}
	return
}

// NewContext returns a new Context.
func NewContext(srcres http.ResponseWriter, srcreq, dstreq *http.Request) Context {
	return Context{SrcRes: srcres, SrcReq: srcreq, DstReq: dstreq}
}

// None is equal to ProcessorFunc(nil).
func None() Processor { return ProcessorFunc(nil) }

// NoError converts a function without return to a processor.
func NoError(f func(ctx context.Context, pc Context)) Processor {
	return ProcessorFunc(func(ctx context.Context, pc Context) error {
		f(ctx, pc)
		return nil
	})
}

// Request is a convenient function to return a simple request processor.
func Request(f func(*http.Request)) Processor {
	return ProcessorFunc(func(ctx context.Context, pc Context) error {
		f(pc.DstReq)
		return nil
	})
}

// ResponseHeader is a convenient function to a simple response header processor.
func ResponseHeader(f func(http.ResponseWriter)) Processor {
	return ProcessorFunc(func(_ context.Context, pc Context) error {
		f(pc.SrcRes)
		return nil
	})
}

// CompactProcessors compacts a group of processors to a processor,
// which will eliminate the empty processor.
func CompactProcessors(processors ...Processor) Processors {
	_processors := make(Processors, 0, len(processors))
	for _, processor := range processors {
		switch p := processor.(type) {
		case nil:
		case ProcessorFunc:
			if p != nil {
				_processors = append(_processors, p)
			}
		case Processors:
			if len(p) > 0 {
				_processors = append(_processors, CompactProcessors(p...)...)
			}
		default:
			_processors = append(_processors, p)
		}
	}
	return _processors
}

// GetProcessorType returns the type of the processor if it is an ExtProcessor.
// Or, reutrn "".
func GetProcessorType(processor Processor) string {
	for {
		switch p := processor.(type) {
		case ExtProcessor:
			return p.Type()
		case interface{ Unwrap() Processor }:
			return GetProcessorType(p.Unwrap())
		default:
			return ""
		}
	}
}

// NewExtProcessor returns a new ExtProcessor.
func NewExtProcessor(ptype, desc string, processor Processor) ExtProcessor {
	if ptype == "" {
		panic("ExtProcessor: type must not be empty")
	}
	if desc == "" {
		panic("ExtProcessor: desc must not be empty")
	}
	if processor == nil {
		panic("ExtProcessor: processor must not be nil")
	}
	return extProcessor{ptype: ptype, pdesc: desc, Processor: processor}
}

type extProcessor struct {
	ptype string
	pdesc string
	Processor
}

func (p extProcessor) String() string    { return p.pdesc }
func (p extProcessor) Type() string      { return p.ptype }
func (p extProcessor) Unwrap() Processor { return p.Processor }
