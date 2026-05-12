// Copyright 2026 xgfone
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

package selector

import (
	"github.com/xgfone/go-loadbalancer/selector/leastconn"
	"github.com/xgfone/go-loadbalancer/selector/random"
	"github.com/xgfone/go-loadbalancer/selector/roundrobin"
	"github.com/xgfone/go-loadbalancer/selector/sourceiphash"
)

var selectors = make(map[string]Selector, 8)

func init() {
	Register(random.NewSelector(""))
	Register(random.NewWeightedSelector(""))
	Register(roundrobin.NewSelector(""))
	Register(roundrobin.NewWeightedSelector(""))
	Register(sourceiphash.NewSelector(""))
	Register(leastconn.NewSelector("", nil))
}

// Register registers the selector.
//
// Some selectors has been registered by default, as follow:
//
//	random
//	roundrobin
//	weight_random
//	weight_roundrobin
//	sourceip_hash
//	leastconn
//
// If exists, override it.
func Register(selector Selector) {
	if selector == nil {
		panic("Register: selector must not be nil")
	}
	selectors[selector.Policy()] = selector
}

// Unregister unregisters the selector by the policy.
//
// If not exist, do nothing.
func Unregister(policy string) { delete(selectors, policy) }

// Get returns the registered selector by the policy.
//
// If not exist, return nil.
func Get(policy string) Selector { return selectors[policy] }

// Gets returns all the registered selectors.
func Gets() []Selector {
	_selectors := make([]Selector, 0, len(selectors))
	for _, selector := range selectors {
		_selectors = append(_selectors, selector)
	}
	return _selectors
}

// Policies returns the policies of all the registered selectors.
func Policies() []string {
	policies := make([]string, 0, len(selectors))
	for policy := range selectors {
		policies = append(policies, policy)
	}
	return policies
}
