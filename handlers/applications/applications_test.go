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

package applications

import (
    "testing"
    "github.com/stretchr/testify/assert"

    "bytes"
    "net/http"
    "net/http/httptest"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/databases"
)

func Test_Init(t *testing.T) {
    databases.SetupTestingDefaultConnection()
    defer databases.TeardownTestingDefaultConnection()

    // Basically just making sure this doesn't panic...
    Init(true)
}

func Test_GetApplicationById_Known(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	keyApp := applicationData {
		Id : 0,
		CurrentDeployment : 314,
		Name : "TestApp",
	    Instances : []interface{}{"instance1"},
	}

	applications.Insert(makeDatabaseEntryFor(keyApp))

    keyApp.Instances, _ = convertInstanceList(keyApp.Instances)

	app, err := GetApplicationById(0)

	assert.Nil(t, err)
	assert.Equal(t, keyApp, app)
}

func Test_GetApplicationById_Unknown(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	_, err := GetApplicationById(0)

	assert.Equal(t, UnknownApplicationError, err)
}

func Test_GetApplicationByName_Known(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	keyApp := applicationData {
		Id : 0,
		CurrentDeployment : 314,
		Name : "TestApp",
	    Instances : []interface{}{"instance1"},
	}

	applications.Insert(makeDatabaseEntryFor(keyApp))

    keyApp.Instances, _ = convertInstanceList(keyApp.Instances)

	app, err := GetApplicationByName("TestApp")

	assert.Nil(t, err)
	assert.Equal(t, keyApp, app)
}

func Test_GetApplicationByName_Unknown(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	_, err := GetApplicationByName("TestApp")

	assert.Equal(t, UnknownApplicationError, err)
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
    	{"POST", "/create"},
    	{"GET",  "/list"},
    	{"GET",  "/list/1234"},
    	{"POST", "/start/1234"},
    	{"POST", "/stop/1234"},
    	{"PUT",  "/revert/1234"},
    	{"PUT",  "/update/1234"},
    }

    for _, route := range routes {
        m := route.Method
        e := route.Endpoint

        req, _ := http.NewRequest(m, e, bytes.NewBuffer([]byte("")))
        tryHandleTest(t, req, r)
    }
}