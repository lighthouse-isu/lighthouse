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

package database

import (
    "os"
    "fmt"
    "database/sql"
    "encoding/json"
)

type Database struct {
    db *sql.DB
    table string
}

const (
    keyColumn string = "keyColumn"
    valueColumn string = "valueColumn"
)

func New(table string) *Database {
    sql := connect()
    this := &Database{sql, table}
    this.drop()
    this.init()
    return this
}

func connect() *sql.DB {
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

func (this *Database) init() {
    query := fmt.Sprintf(`CREATE TABLE %s (%s text PRIMARY KEY, %s json);`,
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
    query := fmt.Sprintf(`INSERT INTO %s (%s, %s) VALUES ('%s', '%s');`,
        this.table, keyColumn, valueColumn, key, string(json))

    _, err := this.db.Exec(query)

    if err != nil {
        fmt.Println(err.Error())
    }
    return err
}

func (this *Database) Update(key string, newValue interface{}) (error) {
    json, _ := json.Marshal(newValue)
    query := fmt.Sprintf(`UPDATE %s SET %s = '%s' WHERE %s = '%s';`,
        this.table, valueColumn, string(json), keyColumn, key)

    _, err := this.db.Exec(query)

    if err != nil {
        fmt.Println(err.Error())
    }
    return err
}

func (this *Database) Select(key string, value interface{}) error {
    query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = '%s';`,
        valueColumn, this.table, keyColumn, key)

    row := this.db.QueryRow(query)

    var data interface{}
    err := row.Scan(&data)

    if err != nil {
        fmt.Println(err.Error())
        return err
    }

    json.Unmarshal(data.([]byte), &value)

    return nil
}