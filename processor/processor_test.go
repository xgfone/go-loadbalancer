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

func TestCompactProcessors(t *testing.T) {
	processors := CompactProcessors(
		nil,
		None(),

		NoError(func(_ context.Context, i interface{}) {
			v := i.(*int)
			*v = *v + 1
		}),

		ProcessorFunc(func(_ context.Context, i interface{}) error {
			v := i.(*int)
			*v = *v + 2
			return nil
		}),
	)
	if _len := len(processors); _len != 2 {
		t.Errorf("expect %d processor, but got %d", 2, _len)
	}

	var v int
	if err := processors.Process(context.Background(), &v); err != nil {
		t.Error(err)
	} else if v != 3 {
		t.Errorf("expect %d, but got %d", 3, v)
	}
}
