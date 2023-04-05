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

var builders = make(map[string]Builder, 16)

func init() {
	RegisterBalancer(random.NewBalancer(""))
	RegisterBalancer(random.NewWeightedBalancer(""))
	RegisterBalancer(roundrobin.NewBalancer(""))
	RegisterBalancer(roundrobin.NewWeightedBalancer(""))
	RegisterBalancer(sourceiphash.NewBalancer(""))
	RegisterBalancer(leastconn.NewBalancer(""))
}

// Builder is used to build a new Balancer with the config.
type Builder func(config interface{}) (Balancer, error)

// RegisterBuidler registers the given balancer builder.
//
// If the balancer builder typed "typ" has existed, override it to the new.
func RegisterBuidler(typ string, builder Builder) { builders[typ] = builder }

// RegisterBalancer registers the balancer as the builder,
// which uses the policy as the buidler type.
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
	RegisterBuidler(b.Policy(), func(interface{}) (Balancer, error) { return b, nil })
}

// GetBuilder returns the registered balancer builder by the type.
//
// If the balancer builder typed "typ" does not exist, return nil.
func GetBuilder(typ string) Builder { return builders[typ] }

// GetAllBuilderTypes returns the types of all the balancer builders.
func GetAllBuilderTypes() []string {
	types := make([]string, 0, len(builders))
	for typ := range builders {
		types = append(types, typ)
	}
	return types
}

// Build is a convenient function to build a new balancer typed "typ".
func Build(typ string, config interface{}) (balancer Balancer, err error) {
	if builder := GetBuilder(typ); builder != nil {
		balancer, err = builder(config)
	} else {
		err = fmt.Errorf("no the balancer builder typed '%s'", typ)
	}
	return
}
