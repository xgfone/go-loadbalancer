// Copyright 2022~2023 xgfone
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
	"github.com/xgfone/go-loadbalancer/balancer/leastconn"
	"github.com/xgfone/go-loadbalancer/balancer/random"
	"github.com/xgfone/go-loadbalancer/balancer/roundrobin"
	"github.com/xgfone/go-loadbalancer/balancer/sourceiphash"
)

var balancers = make(map[string]Balancer, 8)

func init() {
	Register(random.NewBalancer(""))
	Register(random.NewWeightedBalancer(""))
	Register(roundrobin.NewBalancer(""))
	Register(roundrobin.NewWeightedBalancer(""))
	Register(sourceiphash.NewBalancer(""))
	Register(leastconn.NewBalancer("", nil))
}

// Register registers the balancer.
//
// Some balancers has been registered by default, as follow:
//
//	random
//	roundrobin
//	weight_random
//	weight_roundrobin
//	sourceip_hash
//	leastconn
//
// If exists, override it.
func Register(balancer Balancer) {
	if balancer == nil {
		panic("Register: balancer must not be nil")
	}
	balancers[balancer.Policy()] = balancer
}

// Unregister unregisters the balancer by the policy.
//
// If not exist, do nothing.
func Unregister(policy string) { delete(balancers, policy) }

// Get returns the registered balancer by the policy.
//
// If not exist, return nil.
func Get(policy string) Balancer { return balancers[policy] }

// Gets returns all the registered balancers.
func Gets() []Balancer {
	_balancers := make([]Balancer, 0, len(balancers))
	for _, balancer := range balancers {
		_balancers = append(_balancers, balancer)
	}
	return _balancers
}

// Policies returns the policies of all the registered balancers.
func Policies() []string {
	policies := make([]string, 0, len(balancers))
	for policy := range balancers {
		policies = append(policies, policy)
	}
	return policies
}
