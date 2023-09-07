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

// Builder is used to build a processor by the directive and arguments.
type Builder func(directive string, args ...any) (Processor, error)

// Registry is used to collect a set of the processor builders.
type Registry struct {
	builders map[string]Builder
}

// DefalutRegistry is the default global builder registry.
var DefalutRegistry = NewRegistry()

// NewRegistry returns a new processor builder manager.
func NewRegistry() *Registry {
	return &Registry{builders: make(map[string]Builder, 32)}
}

// Reset clears all the registered processor builders.
func (m *Registry) Reset() { clear(m.builders) }

// Register registers the processor builder with the directive.
//
// If exists, override it.
func (m *Registry) Register(directive string, builder Builder) {
	if directive == "" {
		panic("processor directive must not be empty")
	}
	if builder == nil {
		panic("processor builder must not be nil")
	}
	m.builders[directive] = builder
}

// Unregister unregisters the processor builder by the directive.
//
// If not exist, do nothing.
func (m *Registry) Unregister(directive string) {
	delete(m.builders, directive)
}

// Get returns the registered processor builder by the directive.
//
// If not exist, return nil.
func (m *Registry) Get(directive string) Builder { return m.builders[directive] }

// Directives returns the directives of all the registered processor builders.
func (m *Registry) Directives() []string {
	directives := make([]string, 0, len(m.builders))
	for directive := range m.builders {
		directives = append(directives, directive)
	}
	return directives
}

// Build is a convenient function to build a new processor
// by the directive and arguments.
func (m *Registry) Build(directive string, args ...any) (processor Processor, err error) {
	if builder := m.Get(directive); builder == nil {
		err = fmt.Errorf("not found processor builder for directive '%s'", directive)
	} else if processor, err = builder(directive, args...); err == nil && processor == nil {
		panic(fmt.Errorf("got a nil from processor builder for directive '%s'", directive))
	}
	return
}
