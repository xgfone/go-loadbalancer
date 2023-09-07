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

import "context"

var (
	_ Processor = ProcessorFunc(nil)
	_ Processor = Processors(nil)
)

type (
	// Processor is used to process the request or response.
	Processor interface {
		Process(ctx context.Context, data interface{}) error
	}

	// ProcessorFunc is the processor function.
	ProcessorFunc func(ctx context.Context, data interface{}) error

	// Processors represents a group of processors.
	Processors []Processor
)

// Process implements the interface Processor.
//
// f may be nil, which is equal to do nothing and return nil.
func (f ProcessorFunc) Process(ctx context.Context, data interface{}) error {
	if f == nil {
		return nil
	}
	return f(ctx, data)
}

// Process implements the interface Processor.
func (ps Processors) Process(ctx context.Context, data interface{}) (err error) {
	for i, _len := 0, len(ps); i < _len; i++ {
		if err = ps[i].Process(ctx, data); err != nil {
			return
		}
	}
	return
}

// None is equal to ProcessorFunc(nil).
func None() Processor { return ProcessorFunc(nil) }

// NoError converts a function without return to a processor.
func NoError(f func(context.Context, interface{})) Processor {
	return ProcessorFunc(func(ctx context.Context, data interface{}) error {
		f(ctx, data)
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
