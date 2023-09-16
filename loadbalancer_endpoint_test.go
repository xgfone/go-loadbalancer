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
	"context"
	"testing"
)

type endpoint struct{ id string }

func newEndpoint(id string) Endpoint                       { return endpoint{id: id} }
func (e endpoint) ID() string                              { return e.id }
func (e endpoint) Serve(context.Context, any) (any, error) { return nil, nil }

func TestEndpoints(t *testing.T) {
	eps := Endpoints{newEndpoint("id1"), newEndpoint("id2"), newEndpoint("id3")}
	if _len := len(eps); _len != 3 {
		t.Errorf("expect %d endpoints, but got %d", 3, _len)
	}
	if !eps.Contains("id2") {
		t.Errorf("expect containing endpoint '%s', but got not", "id2")
	}
	if eps.Contains("id") {
		t.Errorf("unexpect containing endpoint '%s', but got one", "id")
	}
}
