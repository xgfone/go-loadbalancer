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

package loadbalancer

import (
	"bytes"
	"context"
	"fmt"
	"testing"
)

type lbfunc func(context.Context, any) (any, error)

func (f lbfunc) Serve(c context.Context, r any) (any, error) { return f(c, r) }

func TestManager(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	m := NewManager()
	m.OnAdd(func(name string, _ LoadBalancer) {
		fmt.Fprintf(buf, "add lb '%s'", name)
	})
	m.OnDel(func(name string, _ LoadBalancer) {
		fmt.Fprintf(buf, "del lb '%s'", name)
	})

	_ = m.Add("lb1", lbfunc(nil))
	_ = m.Add("lb2", lbfunc(nil))
	_ = m.Add("lb2", lbfunc(nil))

	if lb := m.Get("lb1"); lb == nil {
		t.Error("expect lb1, but got nil")
	}
	if lb := m.Get("lb2"); lb == nil {
		t.Error("expect lb2, but got nil")
	}

	_ = m.Del("lb1")
	_ = m.Del("lb1")

	if lb := m.Get("lb1"); lb != nil {
		t.Error("unexpect lb1, but got one")
	}
	if lb := m.Get("lb2"); lb == nil {
		t.Error("expect lb2, but got nil")
	}

	if lbs := m.Gets(); len(lbs) != 1 {
		t.Errorf("expect 1 lb, but got %d", len(lbs))
	} else {
		for name := range lbs {
			if name != "lb2" {
				t.Errorf("expect 'lb2', but got '%s'", name)
			}
		}
	}
}
