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
    "github.com/stretchr/testify/assert"
)

func Test_GetBeaconAddress_Found(t *testing.T) {
    teardown := setup()
    defer teardown()

    testInstanceData := map[string]interface{}{
        "InstanceAddress" : "INST_ADDR", 
        "BeaconAddress" : "BEACON_ADDR",
    }

    instances.InsertSchema(testInstanceData)

    res, err := GetBeaconAddress("INST_ADDR")

    assert.Nil(t, err, 
        "GetBeaconAddress should not return error beacon was found")

    assert.Equal(t, "BEACON_ADDR", res, 
        "GetBeaconAddress should give correct address")
}

func Test_GetBeaconAddress_NotFound(t *testing.T) {
    teardown := setup()
    defer teardown()

    res, err := GetBeaconAddress("BAD_ADDR")

    assert.NotNil(t, err, 
        "GetBeaconAddress should forward errors")

    assert.Equal(t, "", res, 
        "GetBeaconAddress should give empty string on error")
}

func Test_GetBeaconToken_NotFound(t *testing.T) {
    teardown := setup()
    defer teardown()

    res, err := GetBeaconToken("BAD_INST", "junk user")

    assert.NotNil(t, err, "GetBeaconToken should forward errors")

    assert.Equal(t, "", res, 
        "GetBeaconToken should give empty token on error")
}

func Test_GetBeaconToken_NotPermitted(t *testing.T) {
    teardown := setup()
    defer teardown()

    testBeaconData := map[string]interface{}{
        "Address" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{},
    }

    beacons.InsertSchema(testBeaconData)

    res, err := GetBeaconToken("BEACON_ADDR", "BAD_USER")

    assert.NotNil(t, err, 
        "GetBeaconToken should return error on bad permissions")

    assert.Equal(t, "", res, 
        "GetBeaconToken should give empty token on bad permissions")
}

func Test_GetBeaconToken_Valid(t *testing.T) {
    teardown := setup()
    defer teardown()

    testBeaconData := map[string]interface{}{
        "Address" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{"USER":true},
    }

    beacons.InsertSchema(testBeaconData)

    res, err := GetBeaconToken("BEACON_ADDR", "USER")

    assert.Nil(t, err, 
        "GetBeaconToken should return nil error on success")

    assert.Equal(t, "TOKEN", res, 
        "GetBeaconToken should give corrent token")
}

func tryHandleTest(t *testing.T, r *http.Request, m *mux.Router) {
    defer func() { recover() }()

    w := httptest.NewRecorder()
    m.ServeHTTP(w, r)

    // This won't run during a panic(), but we can't panic during a 404
    assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func Test_Handle(t *testing.T) {
    r := mux.NewRouter()
    Handle(r)

    routes := []struct {
        Method string
        Endpoint string
    } {
        {"PUT",    "/user/TEST"},
        {"DELETE", "/user/TEST"},
        {"PUT",    "/token/TEST"},
        {"POST",   "/create"},
        {"GET",    "/list"},
        {"GET",    "/list/TEST"},
        {"PUT",    "/refresh/TEST"},
    }

    for _, route := range routes {
        m := route.Method
        e := route.Endpoint

        req, _ := http.NewRequest(m, e, bytes.NewBuffer([]byte("")))
        tryHandleTest(t, req, r)
    }
}