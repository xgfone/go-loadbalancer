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
	"fmt"

	"github.com/xgfone/go-loadbalancer/balancer/leastconn"
	"github.com/xgfone/go-loadbalancer/balancer/random"
	"github.com/xgfone/go-loadbalancer/balancer/roundrobin"
	"github.com/xgfone/go-loadbalancer/balancer/sourceiphash"
)

var builders = make(map[string]Builder, 8)

func init() {
	RegisterBalancer(random.NewBalancer(""))
	RegisterBalancer(random.NewWeightedBalancer(""))
	RegisterBalancer(roundrobin.NewBalancer(""))
	RegisterBalancer(roundrobin.NewWeightedBalancer(""))
	RegisterBalancer(sourceiphash.NewBalancer(""))
	RegisterBalancer(leastconn.NewBalancer("", nil))
}

// Builder is used to build a new Balancer with the config.
type Builder func(policy string, config interface{}) (Balancer, error)

// RegisterBuidler registers the balancer builder for the given policy.
//
// If the balancer builder for policy has existed, override it to the new.
func RegisterBuidler(policy string, builder Builder) {
	if builder == nil {
		panic("balancer builder must not be nil")
	}
	builders[policy] = builder
}

// RegisterBalancer uses the balancer itself as the builder to register.
//
// For builtin balancers as following, they have been registered automatically,
// which will ignore any builder config and never return an error.
//
//	random
//	round_robin
//	weight_random
//	weight_round_robin
//	source_ip_hash
//	least_conn
func RegisterBalancer(b Balancer) {
	RegisterBuidler(b.Policy(), func(string, interface{}) (Balancer, error) { return b, nil })
}

// GetBuilder returns the registered balancer builder by the policy.
//
// If the balancer builder for policy does not exist, return nil.
func GetBuilder(policy string) Builder { return builders[policy] }

// GetAllPolicies returns the policies of all the balancer builders.
func GetAllPolicies() []string {
	policies := make([]string, 0, len(builders))
	for policy := range builders {
		policies = append(policies, policy)
	}
	return policies
}

// Build is a convenient function to build a new balancer for policy.
func Build(policy string, config interface{}) (balancer Balancer, err error) {
	if builder := GetBuilder(policy); builder == nil {
		err = fmt.Errorf("no the balancer builder for the policy '%s'", policy)
	} else if balancer, err = builder(policy, config); err == nil && balancer == nil {
		panic(fmt.Errorf("balancer builder for the policy '%s' returns a nil", policy))
	}
	return
}

// Must is equal to Build, but panics if there is an error.
func Must(policy string, config interface{}) (balancer Balancer) {
	balancer, err := Build(policy, config)
	if err != nil {
		panic(err)
	}
	return
}
