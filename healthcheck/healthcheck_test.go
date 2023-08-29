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
	"sync"
	"testing"
	"time"

	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/internal/tests"
)

type testUpdater struct{ eps sync.Map }

func newUpdater() *testUpdater { return &testUpdater{} }

func (u *testUpdater) UpsertEndpoint(ep endpoint.Endpoint)      { u.eps.Store(ep.ID(), false) }
func (u *testUpdater) RemoveEndpoint(id string)                 { u.eps.Delete(id) }
func (u *testUpdater) SetEndpointOnline(id string, online bool) { u.eps.Store(id, online) }
func (u *testUpdater) Endpoints() map[string]bool {
	eps := make(map[string]bool)
	u.eps.Range(func(key, value interface{}) bool {
		eps[key.(string)] = value.(bool)
		return true
	})
	return eps
}

func TestHealthCheck(t *testing.T) {
	updater1 := newUpdater()
	updater2 := newUpdater()

	hc := NewHealthChecker()
	hc.Start()
	defer hc.Stop()

	_ = hc.AddUpdater("updater1", updater1)
	hc.UpsertEndpoint(tests.NewEndpoint("id1", 1), Checker{})
	hc.UpsertEndpoint(tests.NewEndpoint("id2", 1), Checker{})
	_ = hc.AddUpdater("updater2", updater2)

	time.Sleep(time.Millisecond * 500)

	eps := make(map[string]bool)
	for _, ep := range hc.GetEndpoints() {
		eps[ep.Endpoint.ID()] = ep.Online
	}

	checkEndpoints(t, "hc", eps)
	checkEndpoints(t, "updater1", updater1.Endpoints())
	checkEndpoints(t, "updater2", updater2.Endpoints())
}

func checkEndpoints(t *testing.T, prefix string, eps map[string]bool) {
	if len(eps) != 2 {
		t.Errorf("%s: expect %d endpoints, but got %d", prefix, 2, len(eps))
	} else {
		for epid, online := range eps {
			switch epid {
			case "id1", "id2":
			default:
				t.Errorf("%s: unexpected endpoint '%s'", prefix, epid)
			}

			if !online {
				t.Errorf("%s: the endpoint '%s' is not online", prefix, epid)
			}
		}
	}
}
