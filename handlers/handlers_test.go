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
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// Helper for GetRequestBody tests
func makeRequestAndGetBody(body string) *RequestBody {
	byteBody := []byte(body)
	req, _ := http.NewRequest("", "", bytes.NewBuffer(byteBody))
	return GetRequestBody(req)
}

/*
   Tests body extraction for body-less requests
   Purpose: GET and DELETE requests
*/
func Test_GetRequestBody_Null(t *testing.T) {
	req, _ := http.NewRequest("", "", nil)

	assert.Nil(t, GetRequestBody(req),
		"GetResponseBody should return nil on nil body")
}

/*
   Tests body extraction for bodied requests
   Purpose: [most] POST and PUT requests
*/
func Test_GetRequestBody_Normal(t *testing.T) {
	outBody := makeRequestAndGetBody(`{"Payload":{"TestPayload":1}}`)

	assert.NotNil(t, outBody,
		"GetResponseBody should not return nil on non-nil body")

	key := map[string]interface{}{"TestPayload": float64(1)}

	assert.Equal(t, &RequestBody{key}, outBody,
		"GetResponseBody did not extract body correctly")
}

/*
   Tests body extraction for requests with extra field.
   Purpose: To add niche fields to requests without bogging
       down the RequestBody type.
*/
func Test_GetRequestBody_ExtraPayload(t *testing.T) {
	outBody := makeRequestAndGetBody(`{"Payload":{"TestPayload":1},"Extra":"ExtraField"}`)

	assert.NotNil(t, outBody,
		"GetResponseBody should not return nil on non-nil body")

	key := map[string]interface{}{"TestPayload": float64(1)}

	assert.Equal(t, &RequestBody{key}, outBody,
		"GetResponseBody did not extract Payload correctly with an extra field")
}

/*
   Tests body extraction for requests without a Payload.
   Purpose: To add fields to requests without bogging
       down the RequestBody type.
*/
func Test_GetRequestBody_NoPayload(t *testing.T) {
	outBody := makeRequestAndGetBody(`{"NotAPayload":"TotallyNotAPayload"}`)

	assert.NotNil(t, outBody,
		"GetResponseBody should not return nil on non-nil body")

	assert.Equal(t, &RequestBody{nil}, outBody,
		"GetResponseBody did not extract Payload correctly with an extra field")
}

/*
   Validates that error messages are correctly generated for the user.
   Purpose: Ensuring handler errors reach the user correctly.
*/
func Test_WriteError(t *testing.T) {
	router := mux.NewRouter()

	router.HandleFunc("/",
		func(w http.ResponseWriter, r *http.Request) {
			WriteError(w, 500, "TestCause", "TestMessage")
		})

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, r)

	// Header check
	assert.Equal(t, 500, w.Code,
		"WriteError did not set status code correctly")

	assert.Equal(t, "application/json", w.Header().Get("Content-type"),
		"WriteError did not set content type correctly")

	// Body check
	body, _ := ioutil.ReadAll(w.Body)
	sBody := string(body)

	assert.True(t, strings.Contains(sBody, "TestCause"),
		"WriteError did not add cause to the output")

	assert.True(t, strings.Contains(sBody, "TestMessage"),
		"WriteError did not add message to the output")
}

/*
   Validates custom handler calling.
   Purpose: Ensuring custom handlers are called
*/
func Test_RunCustomHandlers_Normal(t *testing.T) {
	handlers := make(CustomHandlerMap)

	hitCount := 0
	testInfo := HandlerInfo{"testendpoint", "", nil, nil, nil}

	doCall := func(info HandlerInfo, rollback bool) *HandlerError {
		assert.False(t, rollback, "RunCustomHandlers instructed to rollback")
		assert.Equal(t, testInfo, info)
		hitCount += 1
		return nil
	}

	dontCall := func(info HandlerInfo, rollback bool) *HandlerError {
		t.Error("RunCustomHandlers ran an unexpected handler")
		return nil
	}

	handlers[regexp.MustCompile(".*")] = doCall
	handlers[regexp.MustCompile("test")] = doCall
	handlers[regexp.MustCompile("testendpoint")] = doCall
	handlers[regexp.MustCompile("tes.*int")] = doCall
	handlers[regexp.MustCompile("test$")] = dontCall
	handlers[regexp.MustCompile("completelywrong")] = dontCall

	runHandlers, err := RunCustomHandlers(testInfo, handlers)

	assert.Nil(t, err, "RunCustomHandlers returned an unexpected error")

	assert.Equal(t, 4, hitCount,
		"RunCustomHandlers did not run all expected handlers")

	assert.Equal(t, 4, len(runHandlers),
		"RunCustomHandlers did not return all run handlers")
}

/*
   Validates custom handler calling during errors
   Purpose: Ensuring errors custom handlers are not lost
*/
func Test_RunCustomHandlers_Error(t *testing.T) {
	handlers := make(CustomHandlerMap)

	testError := HandlerError{500, "TestCause", "TestMessage"}
	doError := func(info HandlerInfo, rollback bool) *HandlerError {
		return &testError
	}

	handlers[regexp.MustCompile(".*")] = doError
	runHandlers, err := RunCustomHandlers(HandlerInfo{}, handlers)

	assert.NotNil(t, err, "RunCustomHandlers did not return an expected error")

	assert.NotNil(t, runHandlers,
		"RunCustomHandlers did not return a list of run handlers during an error")

	assert.Equal(t, &testError, err,
		"RunCustomHandlers did not a generated error correctly")
}

/*
   Validates custom handler rollback functionality
   Purpose: Ensuring handlers are called correctly during a rollback
*/
func Test_Rollback(t *testing.T) {
	hitCount := 0

	doRollback := func(info HandlerInfo, rollback bool) *HandlerError {
		assert.True(t, rollback, "Rollback did not instruct to rollback")
		hitCount += 1
		return nil
	}

	handlers := []CustomHandlerFunc{doRollback, doRollback, doRollback}
	Rollback(httptest.NewRecorder(), HandlerError{}, HandlerInfo{}, handlers)

	assert.Equal(t, 3, hitCount, "Rollback did not call all handlers")
}

/*
   Validates parameter extraction of wildcarded endpoints
   Purpose: Ensuring handlers get proper endpoint parameters
*/
func Test_GetEndpointParams_Normal(t *testing.T) {
	params := []string{"P1", "P2", "P3"}

	keys := map[string]string{
		"P1": "K1",
		"P2": "K2",
		"P3": "K3",
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		res, ok := GetEndpointParams(r, params)

		assert.True(t, ok)
		assert.Equal(t, keys, res)
	}

	m := mux.NewRouter()
	m.HandleFunc("/{Endpoint:.*}", h)

	r, _ := http.NewRequest("GET", "/K1/K2/K3", nil)
	r.RequestURI = "/K1/K2/K3"

	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
}

/*
   Validates error handling when mux doesn't have an "Endpoint" route variable
*/
func Test_GetEndpointParams_Invalid(t *testing.T) {
	params := []string{"P1"}

	h := func(w http.ResponseWriter, r *http.Request) {
		res, ok := GetEndpointParams(r, params)

		assert.False(t, ok)
		assert.Nil(t, res)
	}

	m := mux.NewRouter()
	m.HandleFunc("/BAD_ROUTE", h)

	r, _ := http.NewRequest("GET", "/K1", nil)
	r.RequestURI = "/K1"

	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
}

/*
   Validates parameter extraction when not enough parameters exist
*/
func Test_GetEndpointParams_TooFew(t *testing.T) {
	params := []string{"P1", "P2", "P3"}

	keys := map[string]string{
		"P1": "K1",
		"P2": "K2",
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		res, ok := GetEndpointParams(r, params)

		assert.True(t, ok)
		assert.Equal(t, keys, res)
	}

	m := mux.NewRouter()
	m.HandleFunc("/{Endpoint:.*}", h)

	r, _ := http.NewRequest("GET", "/K1/K2", nil)
	r.RequestURI = "/K1/K2"

	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
}

/*
   Validates parameter extraction when extra parameters exist
*/
func Test_GetEndpointParams_TooMany(t *testing.T) {
	params := []string{"P1", "P2"}

	keys := map[string]string{
		"P1": "K1",
		"P2": "K2/K3",
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		res, ok := GetEndpointParams(r, params)

		assert.True(t, ok)
		assert.Equal(t, keys, res)
	}

	m := mux.NewRouter()
	m.HandleFunc("/{Endpoint:.*}", h)

	r, _ := http.NewRequest("GET", "/K1/K2/K3", nil)
	r.RequestURI = "/K1/K2/K3"

	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
}
