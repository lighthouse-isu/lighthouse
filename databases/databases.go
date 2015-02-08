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

package databases

import (
    "fmt"
    "bytes"
    "errors"
    "encoding/json"

    "database/sql"
    "database/sql/driver"

    _ "github.com/lib/pq"
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

type Table struct {
    db DBInterface
    table string
}

const (
    keyColumn string = "keyColumn"
    valueColumn string = "valueColumn"
)

func NewTable(db DBInterface, table string) *Table {
    this := &Table{db, table}
    this.drop()
    this.init()
    return this
}

func NewSchemaTable(db DBInterface, table string, columns, types []string) *Table {
    this := &Table{db, table} //store columns?
    this.drop() //check if table already exists, instead of drop?
    this.InitSchema(columns, types)
    return this
}

func (this *Table) init() {
    query := fmt.Sprintf(`CREATE TABLE %s (%s text UNIQUE PRIMARY KEY, %s json);`,
        this.table, keyColumn, valueColumn)

    this.db.Exec(query)
}

func (this *Table) InitSchema(columns, types []string) {
    var buffer bytes.Buffer
    for i :=  0; i < len(columns) && i < len(types); i++ {
        buffer.WriteString(columns[i])
        buffer.WriteString(" ")
        buffer.WriteString(types[i])

        if i + 1 < len(columns) && i + 1 < len(types) {
            buffer.WriteString(",")
        }
    }

    query := fmt.Sprintf(`CREATE TABLE %s (%s);`, this.table, buffer.String())
    this.db.Exec(query)
}

func (this *Table) drop() {
    query := fmt.Sprintf(`DROP TABLE %s;`,
        this.table)

    this.db.Exec(query)
}

func (this *Table) Insert(key string, value interface{}) error {
    json, _ := json.Marshal(value)
    query := fmt.Sprintf(`INSERT INTO %s (%s, %s) VALUES (($1), ($2));`,
        this.table, keyColumn, valueColumn)

    _, err := this.db.Exec(query, key, string(json))

    if err != nil {
        fmt.Println(err.Error())
    }
    return err
}

func (this *Table) InsertSchema(columns, values []string) error {
    var colBuf, valBuf bytes.Buffer
    queryVals := make([]interface{}, len(values))

    if len(columns) != len(values) {
        return errors.New(`Column/Value mismatch`)
    }

    for i, col := range columns {
        if i != 0 {
            colBuf.WriteString(",")
            valBuf.WriteString(",")
        }
        colBuf.WriteString(col)
        s := fmt.Sprintf(`($%d)`, i)
        valBuf.WriteString(s)

        queryVals[i] = values[i]
    }

    query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
        this.table, colBuf.String(), valBuf.String())
    _, err := this.db.Exec(query, queryVals...)

    if err != nil {
        fmt.Println(err.Error())
    }
    return err
}

func (this *Table) Update(key string, newValue interface{}) (error) {
    json, _ := json.Marshal(newValue)
    query := fmt.Sprintf(`UPDATE %s SET %s = ($2) WHERE %s = ($1);`,
        this.table, valueColumn, keyColumn)

    _, err := this.db.Exec(query, key, string(json))

    if err != nil {
        fmt.Println(err.Error())
    }
    return err
}

func (this *Table) SelectRow(key string, value interface{}) error {
    query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ($1);`,
        valueColumn, this.table, keyColumn)

    row := this.db.QueryRow(query, key)

    if row == nil {
        return errors.New("unknown database error")
    }

    var data interface{}
    err := row.Scan(&data)

    if err == nil {
        err = json.Unmarshal(data.([]byte), value)
    }

    return err
}
