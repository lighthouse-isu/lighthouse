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

package permissions

import (
    "os"
    "fmt"
    "database/sql"
)

type Permissions struct {
    db *sql.DB
}

var permissions *Permissions = nil

type Permission struct {
    Email string
    Providers []string
}

func connect() *Permissions {
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

    return &Permissions{postgres}
}

func (this *Permissions) init() *Permissions {
    this.db.Exec("CREATE TABLE permissions (email varchar(40), providers text[])")
    return this
}

func (this *Permissions) drop() *Permissions {
    this.db.Exec("DROP TABLE permissions")
    return this
}

func Setup() {
    if users == nil {
        users = connect().drop().init()
    }
}

func AddPermission(email string, providers []string) {
    Setup()

    permissions.db.Exec("INSERT INTO permissions (email, providers) VALUES (($1), ($2))",
        email, providers) // Probably need to format this
}

func GetPermissions(email string) *User {
    Setup()

    row := permissions.db.QueryRow(
        "SELECT email, providers FROM permissions WHERE email = ($1)", email)

    permission := &Permission{}
    err := row.Scan(&permission.Email, &permission.Providers)

    if err != nil {
        fmt.Println(err.Error())
    }

    return permission
}
