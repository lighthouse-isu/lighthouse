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

package beacons

import (
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/beacons/aliases"
	"github.com/lighthouse/lighthouse/databases"
)

func SetupTestingTable() {
	// schemas defined in beacons.go
	beacons = databases.CommonTestingTable(beaconSchema)
	instances = databases.CommonTestingTable(instanceSchema)
}

func TeardownTestingTable() {
	beacons = nil
	instances = nil
}

func setup() {
	SetupTestingTable()
	auth.SetupTestingTable()
	aliases.SetupTestingTable()
}

func teardown() {
	TeardownTestingTable()
	auth.TeardownTestingTable()
	aliases.TeardownTestingTable()
}

func setupServer(f *func(http.ResponseWriter, *http.Request)) *httptest.Server {

	// Handler function, defaults to an empty func
	var useFunc func(http.ResponseWriter, *http.Request)

	if f != nil {
		useFunc = *f
	} else {
		useFunc = func(http.ResponseWriter, *http.Request) {}
	}

	// Start a new test server to listen for requests from the tests
	server := httptest.NewUnstartedServer(http.HandlerFunc(useFunc))
	server.Listener, _ = net.Listen("tcp", ":8080")
	server.Start()

	return server
}
