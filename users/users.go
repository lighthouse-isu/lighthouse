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

package users

import (
    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/database"
    "github.com/lighthouse/lighthouse/users/permissions"
)

var users *database.Database

type User struct {
    Email string
    Salt string
    Password string
}

func getDBSingleton() *database.Database {
    if users == nil {
        users = database.New("users")
    }
    return users
}

func CreateUser(email, salt, password string) error {
    return getDBSingleton().Insert(email, User{email, salt, password})
}

func GetUser(email string) (user User, err error) {
    err = getDBSingleton().Select(email, &user)
    return
}

func Handle(r *mux.Router) {
    permissions.Handle(r.PathPrefix("/permissions").Subrouter())
}