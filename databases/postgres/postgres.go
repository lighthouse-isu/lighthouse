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
    "strings"

    "database/sql"

    "github.com/lib/pq"

    "github.com/lighthouse/lighthouse/logging"

    "github.com/lighthouse/lighthouse/databases"
)

type postgresConnection struct {
    sql.DB
}

var connection *postgresConnection

func Connection() databases.DBInterface {
    if connection == nil {
        connection = setup()
    }
    return connection
}

func (this *postgresConnection) Exec(cmd string, params ...interface{}) (sql.Result, error) {
    res, err := this.DB.Exec(cmd, params...)
    
    err = transformError(err)

    return res, err
}

func transformError(err error) error {
    var pqErr *pq.Error = nil
    var ok bool = false

    if err != nil {
        pqErr, ok = err.(*pq.Error)
    } 

    if !ok {
        return nil
    }

    switch pqErr.Code {
    case "23505": 
        return databases.DuplicateKeyError
    }

    return nil
}

func setup() *postgresConnection {
    postgresHost := os.Getenv("POSTGRES_PORT_5432_TCP_ADDR")
    dockerHost := os.Getenv("DOCKER_HOST")

    var postgresOptions string

    if postgresHost != "" {
        logging.Info("connecting to a linked container running postgres")

        postgresOptions = fmt.Sprintf(
            "host=%s sslmode=disable user=postgres", postgresHost)

    } else if dockerHost != "" {
        logging.Info("connecting to postgres server inside a docker container")

        dockerHost = strings.Replace(dockerHost, "tcp://", "", 1)
        dockerHost = strings.Split(dockerHost, ":")[0]

        postgresOptions = fmt.Sprintf(
            "host=%s sslmode=disable user=postgres", dockerHost)
    } else {
        logging.Info("connecting to localhost running postgres")
        postgresOptions = "sslmode=disable"
    }

    postgres, err := sql.Open("postgres", postgresOptions)

    if err != nil {
        panic(err.Error())
    }

    if err := postgres.Ping(); err != nil {
        panic(err.Error())
    }

    return &postgresConnection{*postgres}
}
