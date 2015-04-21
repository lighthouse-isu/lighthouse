// Copyright 2014 Caleb Brose, Chris Fogerty, Rob Sheehy, Zach Taylor, Nick Miller
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

package batch

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
)

var nextPortNumber = 8080

func ShutdownServers(servers []*httptest.Server) {
	for _, s := range servers {
		s.Close()
	}
}

func SetupServers(handlers ...func(http.ResponseWriter, *http.Request)) ([]string, []*httptest.Server) {
	addresses := make([]string, len(handlers))
	servers := make([]*httptest.Server, len(handlers))

	for i, f := range handlers {
		if f == nil {
			f = func(http.ResponseWriter, *http.Request) {}
		}

		// Start a new test server to listen for requests from the tests
		server := httptest.NewUnstartedServer(http.HandlerFunc(f))
		server.Listener, _ = net.Listen("tcp", fmt.Sprintf("localhost:%d", nextPortNumber))
		server.Start()

		addresses[i] = strings.Replace(server.URL, "http://", "", 1)
		servers[i] = server
		nextPortNumber++
	}

	return addresses, servers
}
