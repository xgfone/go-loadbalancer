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

package healthcheck

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	num := new(atomic.Int32)

	hc := New("test")
	hc.SetConfig(Config{Interval: time.Second})
	hc.SetChecker(func(context.Context, string) bool { return true })
	hc.OnChanged(func(_ string, online bool) {
		if online {
			num.Add(1)
		} else {
			num.Add(-1)
		}
	})

	hc.Start()
	defer hc.Stop()

	hc.AddTarget("id1")
	hc.AddTargets("id2", "id3", "id4")
	time.Sleep(time.Second)

	targets := hc.Targets()
	if len(targets) != 4 {
		t.Errorf("expect %d targets, but got %d", 4, len(targets))
	} else {
		for id, online := range targets {
			if !online {
				t.Errorf("target '%s' expects online, but got not", id)
			}
		}
	}

	if n := num.Load(); n != 4 {
		t.Errorf("expect %d targets online, but got %d", 4, n)
	}

	hc.DelTarget("id1")
	hc.DelTargets("id2", "id3")
	time.Sleep(time.Second)

	targets = hc.Targets()
	if len(targets) != 1 {
		t.Errorf("expect %d targets, but got %d", 1, len(targets))
	} else if online, ok := targets["id4"]; !ok {
		t.Errorf("expect target '%s', but got %v", "id4", targets)
	} else if !online {
		t.Errorf("expect target '%s' online, but got not", "id4")
	}

	if online, _ := hc.Target("id4"); !online {
		t.Errorf("expect target '%s' online, but got not", "id4")
	}
}

func TestJitter(t *testing.T) {
	start := time.Second * 10
	end := start + time.Second
	for i := 0; i < 100; i++ {
		if d := jitter(start); d <= start || d >= end {
			t.Errorf("duration %s is not in (%s, %s)", d, start, end)
		}
	}

	start = time.Second
	end = start + time.Second/10
	for i := 0; i < 100; i++ {
		if d := jitter(start); d <= start || d >= end {
			t.Errorf("duration %s is not in (%s, %s)", d, start, end)
		}
	}
}
