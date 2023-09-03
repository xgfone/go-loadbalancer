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

// BuilderManager is used to manage a set of the processor builders.
type BuilderManager struct {
	builders map[string]Builder
}

// DefalutBuilderManager is the default global builder manager.
var DefalutBuilderManager = NewBuilderManager()

// NewBuilderManager returns a new processor builder manager.
func NewBuilderManager() *BuilderManager {
	return &BuilderManager{builders: make(map[string]Builder, 32)}
}

// Reset clears all the registered processor builders.
func (m *BuilderManager) Reset() { clear(m.builders) }

// Register registers the processor builder with the directive.
//
// If the directive has existed, override it.
func (m *BuilderManager) Register(directive string, builder Builder) {
	if directive == "" {
		panic("processor directive must not be empty")
	}
	if builder == nil {
		panic("processor builder must not be nil")
	}
	m.builders[directive] = builder
}

// Get returns the registered processor builder by the directive.
//
// If the directive does not exist, return nil.
func (m *BuilderManager) Get(directive string) Builder { return m.builders[directive] }

// AllDirectives returns the directives of all the processor builders.
func (m *BuilderManager) AllDirectives() []string {
	directives := make([]string, 0, len(m.builders))
	for directive := range m.builders {
		directives = append(directives, directive)
	}
	return directives
}

// Build is a convenient function to build a new processor
// by the directive and arguments.
func (m *BuilderManager) Build(directive string, args ...any) (processor Processor, err error) {
	if builder := m.Get(directive); builder == nil {
		err = fmt.Errorf("no the processor builder for the directive '%s'", directive)
	} else if processor, err = builder(directive, args...); err == nil && processor == nil {
		panic(fmt.Errorf("processor builder for the directive '%s' returns a nil", directive))
	}
	return
}
