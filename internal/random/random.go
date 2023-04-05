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

// Package random provides a random generation function.
package random

import (
	"math/rand"
	"sync"
	"time"
)

// NewRandom returns a new function to generate the random number.
func NewRandom() func(int) int {
	lock := new(sync.Mutex)
	random := rand.New(rand.NewSource(time.Now().UnixNano())).Intn
	return func(i int) (n int) {
		lock.Lock()
		n = random(i)
		lock.Unlock()
		return
	}
}
