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
	"testing"
)

func TestRegistry(t *testing.T) {
	DefalutRegistry.Register("add1", func(directive string, args ...any) (Processor, error) {
		return NoError(func(ctx context.Context, data interface{}) {
			v := data.(*int)
			*v = *v + 1
		}), nil
	})

	DefalutRegistry.Register("add2", func(directive string, args ...any) (Processor, error) {
		return NoError(func(ctx context.Context, data interface{}) {
			v := data.(*int)
			*v = *v + 2
		}), nil
	})

	// ------------------------------------------------------------------- //

	if ds := DefalutRegistry.Directives(); len(ds) != 2 {
		t.Errorf("expect %d directives, but got %d", 2, len(ds))
	} else {
		for _, d := range ds {
			switch d {
			case "add1", "add2":
			default:
				t.Errorf("unexpect directive '%s'", d)
			}
		}
	}

	add1, err := DefalutRegistry.Build("add1")
	if err != nil {
		t.Error(err)
	}

	add2, err := DefalutRegistry.Build("add2")
	if err != nil {
		t.Error(err)
	}

	var v int
	err = CompactProcessors(add1, add2).Process(context.Background(), &v)
	if err != nil {
		t.Error(err)
	} else if v != 3 {
		t.Errorf("expect %d, but got %d", 3, v)
	}

	// ------------------------------------------------------------------- //

	DefalutRegistry.Unregister("add1")
	if ds := DefalutRegistry.Directives(); len(ds) != 1 {
		t.Errorf("expect %d directives, but got %d", 1, len(ds))
	} else if ds[0] != "add2" {
		t.Errorf("expect directive '%s', but got '%s'", "add2", ds[0])
	}

	DefalutRegistry.Reset()
	if ds := DefalutRegistry.Directives(); len(ds) != 0 {
		t.Errorf("unexpect any directive, but got %v", ds)
	}

	// ------------------------------------------------------------------- //

	DefalutRegistry.Register("add4", func(directive string, args ...any) (Processor, error) {
		return NoError(func(ctx context.Context, data interface{}) {
			v := data.(*int)
			*v = *v + 4
		}), nil
	})

	add4, err := DefalutRegistry.Build("add4")
	if err != nil {
		t.Error(err)
	}

	if err := add4.Process(context.Background(), &v); err != nil {
		t.Error(err)
	} else if v != 7 {
		t.Errorf("expect %d, but got %d", 7, v)
	}
}
