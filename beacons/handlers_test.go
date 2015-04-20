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

    "fmt"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "bytes"

    "github.com/stretchr/testify/assert"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/session"
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/beacons/aliases"
)

func runHandlerTest(
	method, endpoint string, 
	body interface{}, 
	route string, 
	handler func(w http.ResponseWriter, r *http.Request),
) *httptest.ResponseRecorder {

	w := httptest.NewRecorder()

	jsonBuff, _ := json.Marshal(body)
	buff := bytes.NewBuffer(jsonBuff)

	r, _ := http.NewRequest(method, endpoint, buff)
	r.RequestURI = endpoint

	auth.CreateUser("USER", "", "")
	session.SetValue(r, "auth", "email", "USER")

	m := mux.NewRouter()
	m.HandleFunc(route, handler)
    m.ServeHTTP(w, r)

    return w
}

func setupBeaconPermissions(beacon string, level int) {
	auth.CreateUser("USER", "", "")
	user, _ := auth.GetUser("USER")
	auth.SetUserBeaconAuthLevel(user, beacon, level)
}

func Test_GetAddressOf_Exists(t *testing.T) {
	setup()
	defer teardown()

	aliases.AddAlias("ALIAS", "ADDR")

	assert.Equal(t, "ADDR", getAddressOf("ALIAS"))
}

func Test_GetAddressOf_Unknown(t *testing.T) {
	setup()
	defer teardown()

	assert.Equal(t, "ADDR", getAddressOf("ADDR"))
}

func Test_HandleUpdateBeaconToken(t *testing.T) {
	setup()
	defer teardown()

	beacons.Insert(map[string]interface{}{
		"Address" : "ADDR", "Token" : "TOKEN",
	})

	setupBeaconPermissions("ADDR", 2)
	w := runHandlerTest("PUT", "/ADDR", "TOKEN_PASS", "/{Endpoint}", handleUpdateBeaconToken)

    assert.Equal(t, 200, w.Code)

    var data beaconData
    beacons.SelectRow(nil, nil, nil, &data)
    assert.Equal(t, "TOKEN_PASS", data.Token)
}

func Test_HandleUpdateBeaconToken_Invalid(t *testing.T) {
	setup()
	defer teardown()

	var w *httptest.ResponseRecorder

	// Missing address parameter
	w = runHandlerTest("PUT", "/", "", "/{Endpoint:.*}", handleUpdateBeaconToken)
	assert.Equal(t, 400, w.Code)

	// Can't modify beacon
	w = runHandlerTest("PUT", "/ADDR", "TOKEN", "/{Endpoint:.*}", handleUpdateBeaconToken)
	assert.Equal(t, 403, w.Code)

	setupBeaconPermissions("ADDR", 2)

	// Bad JSON
	w = runHandlerTest("PUT", "/ADDR", []int{1}, "/{Endpoint:.*}", handleUpdateBeaconToken)
	assert.Equal(t, 400, w.Code)

	// Beacon doesn't exist
	w = runHandlerTest("PUT", "/ADDR", "TOKEN", "/{Endpoint:.*}", handleUpdateBeaconToken)
	assert.Equal(t, 400, w.Code)
}

func Test_HandleBeaconCreate(t *testing.T) {
	setup()
	defer teardown()

	vms := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "[]")
	}

    defer setupServer(&vms).Close()

    body := map[string]interface{} {
    	"Address" : "localhost:8080", "Alias" : "ALIAS_PASS", "Token" : "TOKEN_PASS",
    }
	w := runHandlerTest("POST", "/", body, "/", handleBeaconCreate)

    assert.Equal(t, 200, w.Code)

    alias, _ := aliases.GetAliasOf("localhost:8080")
    assert.Equal(t, "ALIAS_PASS", alias)

    var beacon beaconData
    beacons.SelectRow(nil, nil, nil, &beacon)
    assert.Equal(t, "localhost:8080", beacon.Address)
	assert.Equal(t, "TOKEN_PASS", beacon.Token)
}

func Test_HandleBeaconCreate_Invalid(t *testing.T) {
	setup()
	defer teardown()

	var w *httptest.ResponseRecorder
	var body map[string]interface{}

	// Bad JSON
	w = runHandlerTest("POST", "/", []int{}, "/", handleBeaconCreate)
	assert.Equal(t, 400, w.Code)

	// No address
	body = map[string]interface{} {"Address" : "", "Alias" : "ALIAS", "Token" : ""}
	w = runHandlerTest("POST", "/", body, "/", handleBeaconCreate)
	assert.Equal(t, 400, w.Code)

	// No alias
	body = map[string]interface{} {"Address" : "ADDR", "Alias" : "", "Token" : ""}
	w = runHandlerTest("POST", "/", body, "/", handleBeaconCreate)
	assert.Equal(t, 400, w.Code)

	// VMs request fails
	body = map[string]interface{} {"Address" : "ADDR", "Alias" : "ALIAS", "Token" : ""}
	w = runHandlerTest("POST", "/", body, "/", handleBeaconCreate)
	assert.Equal(t, 500, w.Code)
	var beacon beaconData
	err := beacons.SelectRow(nil, nil, nil, &beacon)
	assert.Equal(t, databases.NoRowsError, err)

	// Duplicate beacon
	body = map[string]interface{} {"Address" : "localhost:8080", "Token" : ""}
	beacons.Insert(body)
	body["Alias"] = "ALIAS"

    defer setupServer(nil).Close()

	w = runHandlerTest("POST", "/", body, "/", handleBeaconCreate)
	assert.Equal(t, 400, w.Code)
}

func Test_HandleListBeacons(t *testing.T) {
	setup()
	defer teardown()

	// This is just a wrapper from getBeaconsList which is covered in helpers_test.go
	w := runHandlerTest("GET", "/", nil, "/", handleListBeacons)
    assert.Equal(t, 200, w.Code)
}

func Test_HandleListInstances(t *testing.T) {
	setup()
	defer teardown()

	// This is just a wrapper from getInstancesList which is covered in helpers_test.go
	w := runHandlerTest("GET", "/ADDR", nil, "/{Endpoint}", handleListInstances)
    assert.Equal(t, 200, w.Code)
}