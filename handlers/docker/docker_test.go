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

package docker

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/stretchr/testify/assert"

	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/beacons"
	"github.com/lighthouse/lighthouse/beacons/aliases"
	"github.com/lighthouse/lighthouse/handlers"
	"github.com/lighthouse/lighthouse/session"
)

func setup() string {
	beacons.SetupTestingTable()
	aliases.SetupTestingTable()
	auth.SetupTestingTable()
	auth.CreateUser("email", "", "")
	return "email"
}

func teardown() {
	beacons.TeardownTestingTable()
	aliases.TeardownTestingTable()
	auth.TeardownTestingTable()
}

/*
   These tests verify the functionality of dockerRequestHandler.go. In order to
   perform verifications, the tests each create a test server which runs a
   single handler function.  The purpose of this server is to emulate various
   types of real Docker responses and test proper handling of them.
*/

// Helper to perform test server setup.  Returns a *Server which will
// need to be closed at the end of the calling test
func SetupServer(f *func(http.ResponseWriter, *http.Request)) *httptest.Server {

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

/*
   Tests docker request forwarding for body-less requests
   Purpose: GET and DELETE requests
*/
func Test_DockerRequestHandler_GET(t *testing.T) {
	email := setup()
	defer teardown()

	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("success"))
	}

	defer SetupServer(&h).Close()

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	session.SetValue(r, "auth", "email", email)
	info := handlers.HandlerInfo{"", "localhost:8080", nil, r, nil}

	err := DockerRequestHandler(w, info)

	assert.Nil(t, err, "DockerRequestHandler should return nil error on valid request")

	assert.Equal(t, 200, w.Code,
		"DockerRequestHandler should output the forwarded request's response code.")

	body, _ := ioutil.ReadAll(w.Body)

	assert.Equal(t, "success", string(body),
		"DockerRequestHandler should output the forwarded request's response body.")
}

/*
   Tests docker request forwarding requests with query params
*/
func Test_DockerRequestHandler_query_params(t *testing.T) {
	email := setup()
	defer teardown()

	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		assert.Equal(t, "test=pass", r.URL.RawQuery,
			"DockerRequestHandler should forwarded request's query params.")

		w.Write([]byte("success"))
	}

	defer SetupServer(&h).Close()

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/?test=pass", nil)
	session.SetValue(r, "auth", "email", email)
	info := handlers.HandlerInfo{"/?test=pass", "localhost:8080", nil, r, nil}

	err := DockerRequestHandler(w, info)

	assert.Nil(t, err, "DockerRequestHandler should return nil error on valid request")

	assert.Equal(t, 200, w.Code,
		"DockerRequestHandler should output the forwarded request's response code.")

	body, _ := ioutil.ReadAll(w.Body)

	assert.Equal(t, "success", string(body),
		"DockerRequestHandler should output the forwarded request's response body.")
}

/*
   Tests docker request forwarding for bodied requests
   Purpose: POST and PUT requests
*/
func Test_DockerRequestHandler_POST(t *testing.T) {
	email := setup()
	defer teardown()

	testBody := []byte(`{"Payload" : { "TestBody" : 1 } }`)
	testPayload := map[string]interface{}{"TestBody": 1}

	jsonPayload, _ := json.Marshal(testPayload)

	h := func(w http.ResponseWriter, r *http.Request) {
		// Verify that the body is correctly transferred
		body, _ := ioutil.ReadAll(r.Body)
		assert.Equal(t, string(jsonPayload), string(body))

		w.WriteHeader(200)
		w.Write([]byte("success"))
	}

	defer SetupServer(&h).Close()

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/", bytes.NewBuffer(testBody))
	session.SetValue(r, "auth", "email", email)
	info := handlers.HandlerInfo{"", "localhost:8080", &handlers.RequestBody{testPayload}, r, nil}

	err := DockerRequestHandler(w, info)

	assert.Nil(t, err, "DockerRequestHandler should return nil error on valid request")

	assert.Equal(t, 200, w.Code,
		"DockerRequestHandler should output the forwarded request's response code.")

	body, _ := ioutil.ReadAll(w.Body)

	assert.Equal(t, "success", string(body),
		"DockerRequestHandler should output the forwarded request's response body.")
}

/*
   Tests error handling for bad requests
   Purpose: Ensuring that we handle either bad endpoints, or bad URLS
*/
func Test_DockerRequestHandler_BadEndpoint(t *testing.T) {
	email := setup()
	defer teardown()

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	session.SetValue(r, "auth", "email", email)
	info := handlers.HandlerInfo{"", "localhost:8080", nil, r, nil}

	err := DockerRequestHandler(w, info)

	assert.NotNil(t, err, "DockerRequestHandler should not return nil error on invalid request")

	assert.Equal(t, 500, err.StatusCode,
		"DockerRequestHandler should give a valid error code.")

	assert.Equal(t, err.Cause, "control",
		"DockerRequestHandler should correctly label error causes.")
}

/*
   Tests error handling of remote server errors
   Purpose: Ensuring that we forward remote error correctly
*/
func Test_DockerRequestHandler_ServerError(t *testing.T) {
	email := setup()
	defer teardown()

	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(504)
		w.Write([]byte("timeout"))
	}

	defer SetupServer(&h).Close()

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	session.SetValue(r, "auth", "email", email)
	info := handlers.HandlerInfo{"", "localhost:8080", nil, r, nil}

	err := DockerRequestHandler(w, info)

	assert.NotNil(t, err, "DockerRequestHandler should not return nil error on invalid request")

	assert.Equal(t, 504, err.StatusCode,
		"DockerRequestHandler should output the forwarded request's response code.")

	assert.Equal(t, err.Cause, "docker",
		"DockerRequestHandler should correctly label error causes.")
}

/*
   Tests error handling of remote server errors
   Purpose: Ensuring that we forward remote error correctly
*/
func Test_DockerRequestHandler_NilResponseBody(t *testing.T) {
	email := setup()
	defer teardown()

	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(nil)
	}

	defer SetupServer(&h).Close()

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	session.SetValue(r, "auth", "email", email)
	info := handlers.HandlerInfo{"", "localhost:8080", nil, r, nil}

	err := DockerRequestHandler(w, info)

	assert.Nil(t, err, "DockerRequestHandler should return nil error on valid request")

	assert.Equal(t, 200, w.Code,
		"DockerRequestHandler should output the forwarded request's response code.")
}

/*
   Tests data extraction for requests into a HandlerInfo.
   Purpose: To add ensure Handler get valid data.
*/
func Test_GetHandlerInfo(t *testing.T) {
	setup()
	defer teardown()

	aliases.AddAlias("TestHost", "AliasHost")

	router := mux.NewRouter()
	var info handlers.HandlerInfo

	router.HandleFunc("/{Endpoint:.*}",
		func(w http.ResponseWriter, r *http.Request) {
			info, _ = GetHandlerInfo(r)
		})

	r, _ := http.NewRequest("GET", "/TestHost/Test%2FEndpoint", nil)
	r.RequestURI = "/TestHost/Test%2FEndpoint"

	router.ServeHTTP(httptest.NewRecorder(), r)

	expected_map := make(map[string]interface{})
	expected := handlers.HandlerInfo{"Test/Endpoint", "AliasHost", nil, r, expected_map}

	assert.Equal(t, expected, info,
		"GetHandlerInfo did not extract data correctly")
}
