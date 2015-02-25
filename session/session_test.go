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

package session

import (
    "testing"
    "net/http"

    "github.com/stretchr/testify/assert"
)

const (
    sessionKey = "TEST_KEY"
)

func Test_GetValueOK_Found(t *testing.T) {
    r, _ := http.NewRequest("GET", "/", nil)

    s := GetSession(r, sessionKey)
    s.Values["key"] = "right_value"

    value, ok := GetValueOK(r, sessionKey, "key")

    assert.True(t, ok)
    assert.Equal(t, "right_value", value.(string))
}

func Test_GetValueOK_NotFound(t *testing.T) {
    r, _ := http.NewRequest("GET", "/", nil)

    _, ok := GetValueOK(r, sessionKey, "key")

    assert.False(t, ok)
}

func Test_GetValue_Valid(t *testing.T) {
    r, _ := http.NewRequest("GET", "/", nil)

    s := GetSession(r, sessionKey)
    s.Values["key"] = "right_value"

    value := GetValueOrDefault(r, sessionKey, "key", "wrong_value").(string)

    assert.Equal(t, "right_value", value)
}

func Test_GetValue_Invalid(t *testing.T) {
    r, _ := http.NewRequest("GET", "/", nil)

    s := GetSession(r, sessionKey)
    s.Values["key"] = "wrong_value"

    value := GetValueOrDefault(r, sessionKey, "unknown_key", "right_value").(string)

    assert.Equal(t, "right_value", value)
}

func Test_SetValue_New(t *testing.T) {
    r, _ := http.NewRequest("GET", "/", nil)

    SetValue(r, sessionKey, "key", "right_value")

    s := GetSession(r, sessionKey)
    value := s.Values["key"]

    assert.Equal(t, "right_value", value)
}

func Test_SetValue_Overwrite(t *testing.T) {
    r, _ := http.NewRequest("GET", "/", nil)

    s := GetSession(r, sessionKey)
    s.Values["key"] = "wrong_value"

    SetValue(r, sessionKey, "key", "right_value")

    value := s.Values["key"]

    assert.Equal(t, "right_value", value)
}
