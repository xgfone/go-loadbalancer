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

package nets

import (
	"net"
	"testing"
	"time"
)

func TestIsTimeout(t *testing.T) {
	conn, err := net.DialTimeout("tcp4", "127.0.0.1:10000", time.Microsecond)
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("expect an error, but got nil")
	} else if !IsTimeout(err) {
		t.Errorf("expect a timeout error, but got '%v'", err)
	}
}
