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

package balancer

import (
	"slices"
	"testing"

	"github.com/xgfone/go-loadbalancer/balancer/consistenthash"
)

func TestRegisteredBuiltinBuidler(t *testing.T) {
	Register(consistenthash.NewBalancer("hash_url", func(req any) int { return 0 }))

	expects := []string{
		"random",
		"roundrobin",
		"weight_random",
		"weight_roundrobin",
		"sourceip_hash",
		"leastconn",
		"hash_url",
	}

	for _, s := range expects {
		if Get(s) == nil {
			t.Errorf("not found balancer '%s'", s)
		}
	}

	for _, b := range Gets() {
		if !slices.Contains(expects, b.Policy()) {
			t.Errorf("%s is not in %v", b.Policy(), expects)
		}
	}

	for _, s := range Policies() {
		if !slices.Contains(expects, s) {
			t.Errorf("%s is not in %v", s, expects)
		}
	}

	Unregister("hash_url")
	if slices.Contains(Policies(), "hash_url") {
		t.Errorf("unexpect the balancer for policy 'hash_url'")
	}
}
