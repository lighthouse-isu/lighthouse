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
	"testing"

	"bytes"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/lighthouse/lighthouse/auth"
	"github.com/stretchr/testify/assert"
)

func Test_GetBeaconAddress_Found(t *testing.T) {
	setup()
	defer teardown()

	testInstanceData := map[string]interface{}{
		"InstanceAddress": "INST_ADDR",
		"Name":            "NAME",
		"CanAccessDocker": true,
		"BeaconAddress":   "BEACON_ADDR",
	}

	instances.Insert(testInstanceData)

	res, err := GetBeaconAddress("INST_ADDR")

	assert.Nil(t, err,
		"GetBeaconAddress should not return error beacon was found")

	assert.Equal(t, "BEACON_ADDR", res,
		"GetBeaconAddress should give correct address")
}

func Test_GetBeaconAddress_NotFound(t *testing.T) {
	setup()
	defer teardown()

	res, err := GetBeaconAddress("BAD_ADDR")

	assert.NotNil(t, err,
		"GetBeaconAddress should forward errors")

	assert.Equal(t, "", res,
		"GetBeaconAddress should give empty string on error")
}

func Test_GetBeaconToken_NotFound(t *testing.T) {
	setup()
	defer teardown()

	auth.CreateUser("EMAIL", "", "")
	user, _ := auth.GetUser("EMAIL")
	auth.SetUserBeaconAuthLevel(user, "BAD_ADDR", auth.OwnerAuthLevel)

	res, err := TryGetBeaconToken("BAD_ADDR", user)

	assert.NotNil(t, err,
		"TryGetBeaconToken should forward errors")

	assert.Equal(t, "", res,
		"TryGetBeaconToken should give empty token on error")
}

func Test_GetBeaconToken_NotPermitted(t *testing.T) {
	setup()
	defer teardown()

	testBeaconData := map[string]interface{}{
		"BeaconAddress": "BEACON_ADDR",
		"Token":         "TOKEN",
	}

	beacons.Insert(testBeaconData)

	auth.CreateUser("EMAIL", "", "")
	user, _ := auth.GetUser("EMAIL")
	res, err := TryGetBeaconToken("BEACON_ADDR", user)

	assert.NotNil(t, err,
		"TryGetBeaconToken should return error on bad permissions")

	assert.Equal(t, "", res,
		"TryGetBeaconToken should give empty token on bad permissions")
}

func Test_GetBeaconToken_Valid(t *testing.T) {
	setup()
	defer teardown()

	testBeaconData := map[string]interface{}{
		"Address": "BEACON_ADDR",
		"Token":   "TOKEN",
	}

	beacons.Insert(testBeaconData)

	auth.CreateUser("EMAIL", "", "")
	user, _ := auth.GetUser("EMAIL")
	auth.SetUserBeaconAuthLevel(user, "BEACON_ADDR", auth.OwnerAuthLevel)

	res, err := TryGetBeaconToken("BEACON_ADDR", user)

	assert.Nil(t, err,
		"TryGetBeaconToken should return nil error on success")

	assert.Equal(t, "TOKEN", res,
		"TryGetBeaconToken should give corrent token")
}

func tryHandleTest(t *testing.T, r *http.Request, m *mux.Router) {
	defer func() { recover() }()

	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)

	// This won't run during a panic(), but we can't panic during a 404
	if http.StatusNotFound == w.Code {
		t.Log(r.URL.Path)
		t.Fail()
	}
}

func Test_Handle(t *testing.T) {
	r := mux.NewRouter()
	Handle(r)

	routes := []struct {
		Method   string
		Endpoint string
	}{
		{"PUT", "/token/TEST"},
		{"POST", "/create"},
		{"GET", "/list"},
		{"GET", "/list/TEST"},
		{"PUT", "/refresh/TEST"},
	}

	for _, route := range routes {
		m := route.Method
		e := route.Endpoint

		req, _ := http.NewRequest(m, e, bytes.NewBuffer([]byte("")))
		tryHandleTest(t, req, r)
	}
}

func Test_Init(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	// Basically just making sure this doesn't panic...
	Init(true)
}
