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

package hosts

import (
    "fmt"
    "net/http"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/users"
)

func Handle(r *mux.Router) {
    r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
        session, _ := CookieStore.Get(r, "auth")

        loginForm := &LoginForm{}
        body, _ := ioutil.ReadAll(r.Body)
        json.Unmarshal(body, &loginForm)

        user := users.GetUser(loginForm.Email)
        password := SaltPassword(loginForm.Password, user.Salt)

        if (password == user.Password) {
            session.Values["logged_in"] = true
            session.Values["user_name"] = user
        } else {
            session.Values["logged_in"] = false
        }

        session.Save(r, w)

        fmt.Fprintf(w, "%t", session.Values["logged_in"].(bool))
    }).Methods("POST")
}
