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
    "os"
    "fmt"
    "database/sql"
)

type Users struct {
    db *sql.DB
}

var users *Users = nil

type User struct {
    Email string
    Salt string
    Password string
}

func connect() *Users {
    host := os.Getenv("POSTGRES_PORT_5432_TCP_ADDR")
    var postgresOptions string

    if host == "" { // if running locally
        postgresOptions = "sslmode=disable"
    } else { // if running in docker
        postgresOptions = fmt.Sprintf(
            "host=%s sslmode=disable user=postgres", host)
    }

    postgres, err := sql.Open("postgres", postgresOptions)

    if err != nil {
        fmt.Println(err.Error())
    }

    return &Users{postgres}
}

func (this *Users) init() *Users {
    this.db.Exec("CREATE TABLE users (email varchar(40), salt char(32), password char(128))")
    return this
}

func (this *Users) drop() *Users {
    this.db.Exec("DROP TABLE users")
    return this
}

func Setup() {
    if users == nil {
        users = connect().drop().init()
    }
}

func CreateUser(email, salt, password string) {
    Setup()

    users.db.Exec("INSERT INTO users (email, salt, password) VALUES (($1), ($2), ($3))",
        email, salt, password)
}

func GetUser(email string) *User {
    Setup()

    row := users.db.QueryRow(
        "SELECT email, salt, password FROM users WHERE email = ($1)", email)

    user := &User{}
    err := row.Scan(&user.Email, &user.Salt, &user.Password)

    if err != nil {
        fmt.Println(err.Error())
    }

    return user
}
