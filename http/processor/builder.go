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

import "fmt"

var builders = make(map[string]Builder, 32)

// Builder is used to build a processor by the directive and arguments.
type Builder func(directive string, args ...interface{}) (Processor, error)

// RegisterBuilder registers the processor builder with the directive.
//
// If the directive has existed, override it.
func RegisterBuilder(directive string, builder Builder) {
	if directive == "" {
		panic("processor directive must not be empty")
	}
	if builder == nil {
		panic("processor builder must not be nil")
	}
	builders[directive] = builder
}

// RegisterExtBuilder is the same as RegisterBuilder, but returns an ExtProcessor.
func RegisterExtBuilder(ptype, directive string, builder Builder) {
	if ptype == "" {
		panic("processor type must not be empty")
	}

	RegisterBuilder(directive, func(directive string, args ...interface{}) (Processor, error) {
		p, err := builder(directive, args...)
		if err != nil {
			return nil, err
		}
		desc := fmt.Sprintf("$%s: %v", directive, args)
		return NewExtProcessor(ptype, desc, p), nil
	})
}

// GetBuilder returns the registered processor builder by the directive.
//
// If the directive does not exist, return nil.
func GetBuilder(directive string) Builder { return builders[directive] }

// GetAllDirectives returns the directives of all the processor builders.
func GetAllDirectives() []string {
	directives := make([]string, 0, len(builders))
	for directive := range builders {
		directives = append(directives, directive)
	}
	return directives
}

// Build is a convenient function to build a new directive for directive.
func Build(directive string, args ...interface{}) (processor Processor, err error) {
	if builder := GetBuilder(directive); builder == nil {
		err = fmt.Errorf("no the processor builder for the directive '%s'", directive)
	} else if processor, err = builder(directive, args...); err == nil && processor == nil {
		panic(fmt.Errorf("processor builder for the directive '%s' returns a nil", directive))
	}
	return
}

// Must is equal to Build, but panics if there is an error.
func Must(directive string, args ...interface{}) (processor Processor) {
	processor, err := Build(directive, args...)
	if err != nil {
		panic(err)
	}
	return
}
