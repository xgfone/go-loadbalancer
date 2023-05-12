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

func nothing(_ context.Context, _ Context, e error) error { return e }

func TestCompactProcessors(t *testing.T) {
	processors := CompactProcessors(nil, None(), ProcessorFunc(nothing))
	if _len := len(processors); _len != 1 {
		t.Errorf("expect %d processor, but got %d", 1, _len)
	}
}

func TestGetProcessorType(t *testing.T) {
	f := func(p ExtProcessor) Processor { return p }

	ps := CompactProcessors(f(NewExtProcessor("t1", "t1", ProcessorFunc(nothing))))
	for _, p := range ps {
		if pt := GetProcessorType(p); pt != "t1" {
			t.Errorf("expect processor type '%s', but got '%s'", "t1", pt)
		}
	}
}
