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

package postgres

import (
    "os"
    "fmt"

    "database/sql"

    _ "github.com/lib/pq"
)

var connection *sql.DB

func Connection() *sql.DB {
    if connection == nil {
        connection = setup()
    }
    return connection
}

func setup() *sql.DB {
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

    return postgres
}
