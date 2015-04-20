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
    "net/http"

    "github.com/gorilla/sessions"
    "github.com/gorilla/securecookie"
)

var cookieStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(32))

func GetValueOK(r *http.Request, sessionKey string, key interface{}) (interface{}, bool) {
    session := GetSession(r, sessionKey)
    val, ok := session.Values[key]
    return val, ok
}

func GetValueOrDefault(r *http.Request, sessionKey string, key, def interface{}) interface{} {
    session := GetSession(r, sessionKey)
    val, ok := session.Values[key]
    if ok {
        return val
    }
    return def
}

func SetValue(r *http.Request, sessionKey string, key, value interface{}) {
    session := GetSession(r, sessionKey)
    session.Values[key] = value
}

func Save(sessionKey string, r *http.Request, w http.ResponseWriter) {
    GetSession(r, sessionKey).Save(r, w)
}

func GetSession(r *http.Request, sessionKey string) *sessions.Session {
    session, _ := cookieStore.Get(r, sessionKey)
    return session
}
