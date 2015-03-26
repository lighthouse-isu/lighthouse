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

package aliases

import (
	"testing"

	"net/http"
	"net/http/httptest"
	"encoding/json"
	"bytes"

	"github.com/gorilla/mux"

	"github.com/stretchr/testify/assert"

	"github.com/lighthouse/lighthouse/databases"
)

func setup() (table databases.TableInterface, teardown func()) {
	SetupTestingTable()
	table = aliases
	teardown = TeardownTestingTable
	return 
}

func Test_AddAlias(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	keyAlias := "ALIAS"
	keyAddress := "ADDRESS"

	AddAlias(keyAlias, keyAddress)

	var real struct {
		Address string
	}

	table.SelectRow([]string{"Address"}, nil, &real)

	assert.Equal(t, keyAddress, real.Address)
}

func Test_UpdateAlias(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	alias := map[string]interface{}{
		"Alias" : "ALIAS_FAIL",
		"Address" : "ADDRESS",
	}

	table.Insert(alias, "")

	keyAlias := "ALIAS_PASS"

	UpdateAlias(keyAlias, "ADDRESS")

	var real struct {
		Alias string
	}

	table.SelectRow([]string{"Alias"}, nil, &real)

	assert.Equal(t, keyAlias, real.Alias)
}

func Test_SetAlias(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	alias := map[string]interface{}{
		"Alias" : "ALIAS_OVERWRITE",
		"Address" : "ADDRESS_OVERWRITE",
	}

	table.Insert(alias, "")

	overwriteAlias := "ALIAS_UPDATED"
	SetAlias(overwriteAlias, "ADDRESS_OVERWRITE")

	addedAlias := "ALIAS_ADDED"
	SetAlias(addedAlias, "ADDRESS_ADDED")

	var real Alias
	where := make(databases.Filter)

	where["Address"] = "ADDRESS_OVERWRITE"
	table.SelectRow([]string{"Alias"}, where, &real)

	assert.Equal(t, overwriteAlias, real.Alias)

	where["Address"] = "ADDRESS_ADDED"
	table.SelectRow([]string{"Alias"}, where, &real)

	assert.Equal(t, addedAlias, real.Alias)
}

func Test_GetAddressOf(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	alias := map[string]interface{}{
		"Alias" : "ALIAS",
		"Address" : "ADDRESS",
	}

	table.Insert(alias, "")

	keyAddress := "ADDRESS"
	var realAddress string
	var err error

	realAddress, err = GetAddressOf("ALIAS")
	assert.Nil(t, err)
	assert.Equal(t, keyAddress, realAddress)

	realAddress, err = GetAddressOf("BAD_ALIAS")
	assert.NotNil(t, err)
	assert.Equal(t, "", realAddress)
}

func Test_GetAliasOf(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	alias := map[string]interface{}{
		"Alias" : "ALIAS",
		"Address" : "ADDRESS",
	}

	table.Insert(alias, "")

	keyAlias := "ALIAS"
	var realAddress string
	var err error

	realAddress, err = GetAliasOf("ADDRESS")
	assert.Nil(t, err)
	assert.Equal(t, keyAlias, realAddress)

	realAddress, err = GetAliasOf("BAD_ADDRESS")
	assert.NotNil(t, err)
	assert.Equal(t, "", realAddress)
}

func Test_HandleUpdateAlias_Existing(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	alias := map[string]interface{}{
		"Alias" : "ALIAS_FAIL",
		"Address" : "ADDRESS",
	}

	table.Insert(alias, "")

	keyAlias := "ALIAS_PASS"
	aliasJSON, _ := json.Marshal(keyAlias)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "/ADDRESS", bytes.NewBuffer(aliasJSON))

	m := mux.NewRouter()
	m.HandleFunc("/{Address:.*}", handleUpdateAlias)
	m.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var real struct {
		Alias string
	}

	table.SelectRow([]string{"Alias"}, nil, &real)
	assert.Equal(t, keyAlias, real.Alias)
}

func Test_HandleUpdateAlias_New(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	keyAlias := "ALIAS_PASS"
	aliasJSON, _ := json.Marshal(keyAlias)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "/ADDRESS", bytes.NewBuffer(aliasJSON))

	m := mux.NewRouter()
	m.HandleFunc("/{Address:.*}", handleUpdateAlias)
	m.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var real struct {
		Alias string
	}

	table.SelectRow([]string{"Alias"}, nil, &real)
	assert.Equal(t, keyAlias, real.Alias)
}

func Test_HandleUpdateAlias_Invalid(t *testing.T) {
	_, teardown := setup()
	defer teardown()

	addressJSON, _ := json.Marshal("")

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "/ADDRESS", bytes.NewBuffer(addressJSON))

	m := mux.NewRouter()
	m.HandleFunc("/{Address:.*}", handleUpdateAlias)
	m.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_HandleUpdateAlias_BadJSON(t *testing.T) {
	_, teardown := setup()
	defer teardown()

	addressJSON, _ := json.Marshal([]int{1})

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "/ADDRESS", bytes.NewBuffer(addressJSON))

	m := mux.NewRouter()
	m.HandleFunc("/{Address:.*}", handleUpdateAlias)
	m.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
        Method string
        Endpoint string
    } {
        {"PUT", "/ADDR"},
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