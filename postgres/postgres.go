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
    "errors"
    "encoding/json"

    "database/sql"
    "database/sql/driver"

    _ "github.com/lib/pq"

    "github.com/lighthouse/lighthouse/logging"
)

type DBInterface interface {
    Begin() (*sql.Tx, error)
    Close() error
    Driver() driver.Driver
    Exec(string, ...interface{}) (sql.Result, error)
    Ping() error
    Prepare(string) (*sql.Stmt, error)
    Query(string, ...interface{}) (*sql.Rows, error)
    QueryRow(string, ...interface{}) *sql.Row
    SetMaxIdleConns(int)
}

type Database struct {
    db DBInterface
    table string
}

const (
    keyColumn string = "keyColumn"
    valueColumn string = "valueColumn"
)

func New(table string) *Database {
    db := connect()
    return NewFromDB(table, db)
}

func NewFromDB(table string, db DBInterface) *Database {
    this := &Database{db, table}
    this.drop()
    this.init()
    return this
}

func connect() *sql.DB {
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

    return postgres
}

func (this *Database) init() {
    query := fmt.Sprintf(`CREATE TABLE %s (%s text UNIQUE PRIMARY KEY, %s json);`,
        this.table, keyColumn, valueColumn)

    this.db.Exec(query)
}

func (this *Database) drop() {
    query := fmt.Sprintf(`DROP TABLE %s;`,
        this.table)

    this.db.Exec(query)
}

func (this *Database) Insert(key string, value interface{}) error {
    json, _ := json.Marshal(value)
    query := fmt.Sprintf(`INSERT INTO %s (%s, %s) VALUES (($1), ($2));`,
        this.table, keyColumn, valueColumn)

    _, err := this.db.Exec(query, key, string(json))

    if err != nil {
        fmt.Println(err.Error())
    }
    return err
}

func (this *Database) Update(key string, newValue interface{}) (error) {
    json, _ := json.Marshal(newValue)
    query := fmt.Sprintf(`UPDATE %s SET %s = ($1) WHERE %s = ($2);`,
        this.table, valueColumn, keyColumn)

    _, err := this.db.Exec(query, string(json), key)

    if err != nil {
        fmt.Println(err.Error())
    }
    return err
}

func (this *Database) SelectRow(key string, value interface{}) error {
    query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ($1);`,
        valueColumn, this.table, keyColumn)

    row := this.db.QueryRow(query, key)

    var data interface{}
    err := row.Scan(&data)

    switch {
    case err == sql.ErrNoRows:
        return errors.New("key not found")

    case err != nil:
        return err

    default:
        return json.Unmarshal(data.([]byte), &value)
    }
}
