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

package handlers

import (
    "testing"
    "net"
    "net/http"
    "net/http/httptest"
    "io/ioutil"
    "bytes"

    "github.com/stretchr/testify/assert"
)

// Helper to perform test server setup.  Returns a *Server which will
// need to be closed at the end of the calling test
func SetupServer(f *func(http.ResponseWriter, *http.Request)) *httptest.Server {
    var useFunc func(http.ResponseWriter, *http.Request)

    if f != nil {
        useFunc = *f
    } else {
        useFunc = func(http.ResponseWriter, *http.Request) {}
    }

    server := httptest.NewUnstartedServer(http.HandlerFunc(useFunc))
    server.Config.Addr = "/"

    go func() {
        l, _ := net.Listen("tcp", ":8080")
        server.Listener = l
        server.Start()
    }()

    return server
}

/*
    Tests docker request forwarding for body-less requests
    Purpose: GET and DELETE requests
*/
func Test_DockerRequestHandler_GET(t *testing.T) {
    h :=  func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte("success"))
    }

    defer SetupServer(&h).Close()

    w := httptest.NewRecorder()
    r, _ := http.NewRequest("GET", "/", nil)
    info := HandlerInfo{"", "localhost:8080", nil, r}

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

    testBody := []byte("TestBody")

    h :=  func(w http.ResponseWriter, r *http.Request) {
        body, _ := ioutil.ReadAll(r.Body)
        assert.Equal(t, testBody, body)
        w.WriteHeader(200)
        w.Write([]byte("success"))
    }

    defer SetupServer(&h).Close()

    w := httptest.NewRecorder()
    r, _ := http.NewRequest("POST", "/", bytes.NewBuffer(testBody))
    info := HandlerInfo{"", "localhost:8080", &RequestBody{string(testBody)}, r}

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
    w := httptest.NewRecorder()
    r, _ := http.NewRequest("GET", "/", nil)
    info := HandlerInfo{"", "localhost:8080", nil, r}

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
    h :=  func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(504)
        w.Write([]byte("timeout"))
    }

    defer SetupServer(&h).Close()

    w := httptest.NewRecorder()
    r, _ := http.NewRequest("GET", "/", nil)
    info := HandlerInfo{"", "localhost:8080", nil, r}

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
    h :=  func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write(nil)
    }

    defer SetupServer(&h).Close()

    w := httptest.NewRecorder()
    r, _ := http.NewRequest("GET", "/", nil)
    info := HandlerInfo{"", "localhost:8080", nil, r}

    err := DockerRequestHandler(w, info)

    assert.Nil(t, err, "DockerRequestHandler should not return nil error on invalid request")

    assert.Equal(t, 200, w.Code,
        "DockerRequestHandler should output the forwarded request's response code.")
}
