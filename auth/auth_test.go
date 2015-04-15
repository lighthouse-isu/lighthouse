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

package auth

import (
	"testing"

	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"

	"github.com/lighthouse/lighthouse/session"
)

func Test_SaltPassword(t *testing.T) {
	SECRET_HASH_KEY = "i'm the secret hash key"
	expectedResult := "00a987631776516d7dd00e8ace06c5c5a83739dbf95742ddf5f39eeda1f26c346f235131b4bc1a1eb244d479f899610f420e23cefb139d47c0d9a07ed1bf909c"
	assert.Equal(t, expectedResult, SaltPassword("12345", "i'm the salt"))
}

func Test_LoginOK(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	r := mux.NewRouter()
	Handle(r)

	CreateUser("TEST", "SALT", SaltPassword("PASSWORD", "SALT"))

	form := LoginForm{"TEST", "PASSWORD"}
	body, _ := json.Marshal(form)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))

	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func Test_LoginInvalid(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	r := mux.NewRouter()
	Handle(r)

	form := LoginForm{"TEST", "PASSWORD"}
	body, _ := json.Marshal(form)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))

	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func Test_MiddlewareOK(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	m := mux.NewRouter()
	Handle(m)
	r := AuthMiddleware(m, nil)

	CreateUser("TEST", "SALT", SaltPassword("PASSWORD", "SALT"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/list", nil)

	session.SetValue(req, "auth", "logged_in", true)
	session.SetValue(req, "auth", "email", "TEST")

	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func Test_MiddlewareInvalid_NonAPI(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	m := mux.NewRouter()
	Handle(m)
	r := AuthMiddleware(m, nil)

	CreateUser("TEST", "SALT", SaltPassword("PASSWORD", "SALT"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/list", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, 302, w.Code)
}

func Test_MiddlewareInvalid_API(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	m := mux.NewRouter().PathPrefix("/api").Subrouter()
	Handle(m)
	r := AuthMiddleware(m, nil)

	CreateUser("TEST", "SALT", SaltPassword("PASSWORD", "SALT"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users/list", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
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
		{"POST", "/login"},
		{"GET", "/logout"},
		{"GET", "/users/list"},
		{"GET", "/users/TEST"},
		{"PUT", "/users/TEST"},
		{"POST", "/users/create"},
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
