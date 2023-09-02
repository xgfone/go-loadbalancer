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

package endpoint_test

import (
	"testing"

	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/internal/tests"
)

func TestManager(t *testing.T) {
	m := endpoint.NewManager(4)

	m.Add(tests.NewNoopEndpoint("1.2.3.4", 1))
	m.Adds(tests.NewNoopEndpoint("1.2.3.5", 1), tests.NewNoopEndpoint("1.2.3.6", 1))

	m.Upsert(tests.NewNoopEndpoint("1.2.3.7", 1))
	m.Upserts(tests.NewNoopEndpoint("1.2.3.8", 1), tests.NewNoopEndpoint("1.2.3.9", 1))

	if num := m.Len(); num != 6 {
		t.Errorf("expect %d endpoints, but got %d", 6, num)
	}
	if num := m.Discover().Len(); num != 0 {
		t.Errorf("expect %d endpoints, but got %d", 0, num)
	}

	m.SetOnline("1.2.3.4", true)
	if num := m.Discover().Len(); num != 1 {
		t.Errorf("expect %d endpoints, but got %d", 1, num)
	}
	if eps := m.Discover().Endpoints; len(eps) != 1 {
		t.Errorf("expect %d online endpoints, but got %d", 1, len(eps))
	} else if id := eps[0].ID(); id != "1.2.3.4" {
		t.Errorf("expect online endpoint '%s', but got '%s'", "1.2.3.4", id)
	}

	m.SetOnline("1.2.3.5", true)
	if num := m.Discover().Len(); num != 2 {
		t.Errorf("expect %d endpoints, but got %d", 2, num)
	}
	if eps := m.Discover().Endpoints; len(eps) != 2 {
		t.Errorf("expect %d online endpoints, but got %d", 2, len(eps))
	} else if id := eps[0].ID(); id != "1.2.3.4" {
		t.Errorf("expect online endpoint '%s', but got '%s'", "1.2.3.4", id)
	} else if id := eps[1].ID(); id != "1.2.3.5" {
		t.Errorf("expect online endpoint '%s', but got '%s'", "1.2.3.5", id)
	}

	m.SetOnline("1.2.3.6", true)
	if num := m.Discover().Len(); num != 3 {
		t.Errorf("expect %d endpoints, but got %d", 3, num)
	}
	if eps := m.Discover().Endpoints; len(eps) != 3 {
		t.Errorf("expect %d online endpoints, but got %d", 3, len(eps))
	} else if id := eps[0].ID(); id != "1.2.3.4" {
		t.Errorf("expect online endpoint '%s', but got '%s'", "1.2.3.4", id)
	} else if id := eps[1].ID(); id != "1.2.3.5" {
		t.Errorf("expect online endpoint '%s', but got '%s'", "1.2.3.5", id)
	} else if id := eps[2].ID(); id != "1.2.3.6" {
		t.Errorf("expect online endpoint '%s', but got '%s'", "1.2.3.6", id)
	}

	if ep, online := m.Get("1.2.3.7"); ep == nil {
		t.Error("execpt an endpoint, but got none")
	} else if online {
		t.Error("expec an offline endpoint, but got online")
	}

	if eps := m.All(); len(eps) != 6 {
		t.Errorf("expect %d endpoints, but got %d", 6, len(eps))
	}

	m.Clear()
	if num := m.Discover().Len(); num != 0 {
		t.Errorf("expect %d online endpoints, but got %d", 0, num)
	}
	if num := m.Len(); num != 0 {
		t.Errorf("expect %d endpoints, but got %d", 0, num)
	}
}
