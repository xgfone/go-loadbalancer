// Copyright 2021~2023 xgfone
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

type noneep string

func (ep noneep) ID() string                                  { return string(ep) }
func (ep noneep) Serve(context.Context, any) (r any, e error) { return }

type doer interface{ Do(*Static) }

type dofunc func(*Static)

func (f dofunc) Do(s *Static) { f(s) }

func BenchmarkEndpointsDiscovery(b *testing.B) {
	var do doer = dofunc(func(s *Static) {})

	eps := Endpoints{noneep("ep1"), noneep("ep2"), noneep("ep3")}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			do.Do(eps.Discover())
		}
	})
}

func BenchmarkStaticDiscovery(b *testing.B) {
	var do doer = dofunc(func(s *Static) {})

	s := NewStatic(Endpoints{noneep("ep1"), noneep("ep2"), noneep("ep3")})

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			do.Do(s.Discover())
		}
	})
}
