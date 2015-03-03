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

func Test_WhoAmI_NotLoggedIn(t *testing.T) {
	r, _ := http.NewRequest("GET", "/whoami", nil)

    m := mux.NewRouter()
    Handle(m)	
    w := httptest.NewRecorder()

    m.ServeHTTP(w, r)

    assert.Equal(t, 404, w.Code)
}

func Test_WhoAmI_LoggedIn(t *testing.T) {
	r, _ := http.NewRequest("GET", "/whoami", nil)
    session.SetValue(r, "auth", "email", "TEST_EMAIL")

    m := mux.NewRouter()
    Handle(m)	
    w := httptest.NewRecorder()

    m.ServeHTTP(w, r)

    assert.Equal(t, 200, w.Code)
    assert.Equal(t, "TEST_EMAIL", w.Body.String())
}

func Test_WhoAmI_NotLoggedIn(t *testing.T) {
	r, _ := http.NewRequest("GET", "/whoami", nil)

    m := mux.NewRouter()
    Handle(m)	
    w := httptest.NewRecorder()

    m.ServeHTTP(w, r)

    assert.Equal(t, 404, w.Code)
}