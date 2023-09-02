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

package endpoint

import "github.com/xgfone/go-loadbalancer"

// Weighter is used to get the weight of an endpoint.
type Weighter interface {
	// Weight returns the weight of the endpoint, which must be a positive integer.
	//
	// The bigger the value, the higher the weight.
	Weight() int
}

// GetWeight returns the weight of the endpoint if it has implements
// the interface Weighter. Or, check whether it has implemented
// the interface loadbalancer.Unwrapper and unwrap it.
// If still failing, return 1 instead.
func GetWeight(ep loadbalancer.Endpoint) int {
	switch s := ep.(type) {
	case Weighter:
		if weight := s.Weight(); weight > 0 {
			return weight
		}
		return 1

	case loadbalancer.Unwrapper:
		return GetWeight(s.Unwrap())

	default:
		return 1
	}
}
