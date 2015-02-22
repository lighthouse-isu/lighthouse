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

	table.SelectRowSchema([]string{"Address"}, nil, &real)

	assert.Equal(t, keyAddress, real.Address)
}

func Test_UpdateAlias(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	alias := map[string]interface{}{
		"Alias" : "ALIAS",
		"Address" : "ADDRESS_FAIL",
	}

	table.InsertSchema(alias)

	keyAddress := "ADDRESS_PASS"

	UpdateAlias("ALIAS", keyAddress)

	var real struct {
		Address string
	}

	table.SelectRowSchema([]string{"Address"}, nil, &real)

	assert.Equal(t, keyAddress, real.Address)
}

func Test_GetAddressOf(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	alias := map[string]interface{}{
		"Alias" : "ALIAS",
		"Address" : "ADDRESS",
	}

	table.InsertSchema(alias)

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

	table.InsertSchema(alias)

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
		"Alias" : "ALIAS",
		"Address" : "Address_FAIL",
	}

	table.InsertSchema(alias)

	keyAddress := "Address_PASS"
	addressJSON, _ := json.Marshal(keyAddress)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "/ALIAS", bytes.NewBuffer(addressJSON))

	m := mux.NewRouter()
	m.HandleFunc("/{Alias:.*}", handleUpdateAlias)
	m.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var real struct {
		Address string
	}

	table.SelectRowSchema([]string{"Address"}, nil, &real)
	assert.Equal(t, keyAddress, real.Address)
}

func Test_HandleUpdateAlias_New(t *testing.T) {
	table, teardown := setup()
	defer teardown()

	keyAddress := "Address_PASS"
	addressJSON, _ := json.Marshal(keyAddress)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "/ALIAS", bytes.NewBuffer(addressJSON))

	m := mux.NewRouter()
	m.HandleFunc("/{Alias:.*}", handleUpdateAlias)
	m.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var real struct {
		Address string
	}

	table.SelectRowSchema([]string{"Address"}, nil, &real)
	assert.Equal(t, keyAddress, real.Address)
}

func Test_HandleUpdateAlias_Invalid(t *testing.T) {
	_, teardown := setup()
	defer teardown()

	addressJSON, _ := json.Marshal("")

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "/ALIAS", bytes.NewBuffer(addressJSON))

	m := mux.NewRouter()
	m.HandleFunc("/{Alias:.*}", handleUpdateAlias)
	m.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}