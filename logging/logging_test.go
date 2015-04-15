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

package logging

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockWriter struct {
	LogInput string
}

func (this *MockWriter) Write(b []byte) (int, error) {
	this.LogInput = string(b)
	return len(b), nil
}

func Test_LoggingMiddleware(t *testing.T) {
	mw := &MockWriter{}
	duration, method, url := "", "", ""

	oldLogger := logger
	defer func() {
		// restore patched logger after done testing
		logger = oldLogger
	}()

	// patch logger with a mock logger
	logger = log.New(mw, "", 0)

	handler := Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/some", nil)
	handler.ServeHTTP(w, r)

	fmt.Sscanf(mw.LogInput, "%12s %s %s\n", &duration, &method, &url)

	assert.Equal(t, url, "/some",
		"Logging Middleware didn't correctly log the url.")
	assert.Equal(t, method, "GET",
		"Logging Middleware didn't correctly log the method.")
}

func Test_Info(t *testing.T) {
	mw := &MockWriter{}
	key, res := "TEST_INFO", ""

	oldLogger := logger
	defer func() {
		// restore patched logger after done testing
		logger = oldLogger
	}()

	// patch logger with a mock logger
	logger = log.New(mw, "", 0)

	Info(key)

	fmt.Sscanf(mw.LogInput, "%s\n", &res)

	assert.Equal(t, key, res)
}
