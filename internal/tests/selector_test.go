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

package tests

import (
	"context"
	"testing"

	"github.com/xgfone/go-loadbalancer"
)

type _Selector func(context.Context, any, *loadbalancer.Static) (any, error)

func (f _Selector) Select(context.Context, any, *loadbalancer.Static) (loadbalancer.Endpoint, error) {
	return nil, nil
}

func BenchmarkBenchBalancer(b *testing.B) {
	BenchSelector(b, _Selector(nil))
}
