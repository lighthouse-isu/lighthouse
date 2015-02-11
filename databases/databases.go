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
    "strings"
    "errors"
    "reflect"
    "encoding/json"

    "database/sql"
    "database/sql/driver"
)

var (
    EmptyKeyError = errors.New("databases: given key was empty string")
    NoUpdateError = errors.New("databases: no rows updated")
    UnknownError = errors.New("databases: unknown error")
    KeyNotFoundError = errors.New("databases: given key not found")
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
    schema Schema
}

type Schema map[string]string
type Filter map[string]interface{}

const (
    keyColumn string = "keyColumn"
    valueColumn string = "valueColumn"
)

func NewTable(db DBInterface, table string) *Table {
    this := &Table{db, table, nil}
    this.drop()
    this.init()
    return this
}

func NewSchemaTable(db DBInterface, table string, schema Schema) *Table {
    this := &Table{db, table, schema}
    this.drop()
    this.InitSchema()
    return this
}

func (this *Table) init() {
    query := fmt.Sprintf(`CREATE TABLE %s (%s text UNIQUE PRIMARY KEY, %s json);`,
        this.table, keyColumn, valueColumn)

    this.db.Exec(query)
}

func (this *Table) InitSchema() {
    if len(this.schema) == 0 {
        panic("No schema given to database")
    }

    var buffer bytes.Buffer
    first := true

    for col, colType := range this.schema {
        if !first {
            buffer.WriteString(", ")
        }

        buffer.WriteString(col)
        buffer.WriteString(" ")
        buffer.WriteString(colType)

        first = false
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

func (this *Table) getValueOf(orig interface{}, col string) interface{} {
    colType := this.schema[col]

    if strings.Contains(colType, "text") {
        return string(orig.([]byte))
    }

    if strings.Contains(colType, "json") {
        var read interface{}

        err := json.Unmarshal(orig.([]byte), &read)
        if err != nil {
            return orig
        }

        return read
    }
    
    return orig
}

func (this *Table) InsertSchema(values map[string]interface{}) error {
    var colBuf, valBuf bytes.Buffer
    queryVals := make([]interface{}, len(values))
    i := 0

    for col, val := range values {
        if i != 0 {
            colBuf.WriteString(",")
            valBuf.WriteString(",")
        }
        colBuf.WriteString(col)
        s := fmt.Sprintf(`($%d)`, i + 1)
        valBuf.WriteString(s)

        queryVals[i] = val

        i += 1
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

func (this *Table) UpdateSchema(to, where map[string]interface{}) (error) {
    var buffer bytes.Buffer

    buffer.WriteString("UPDATE ")
    buffer.WriteString(this.table)
    buffer.WriteString(" SET ")

    vals := make([]interface{}, len(where) + len(to))

    i := 1

    for col, val := range to {
        if i != 1 {
            buffer.WriteString(", ")
        }

        buffer.WriteString(col)
        buffer.WriteString(" = ")
        buffer.WriteString(fmt.Sprintf("($%d)", i))

        vals[i - 1] = val
        i += 1
    }

    buffer.WriteString(" WHERE ")

    for col, val := range where {
        if i != len(to) + 1 {
            buffer.WriteString(" && ")
        }

        buffer.WriteString(col)
        buffer.WriteString(" = ")
        buffer.WriteString(fmt.Sprintf("($%d)", i))

        vals[i - 1] = val
        i += 1
    }

    buffer.WriteString(";")

    query := buffer.String()

    _, err := this.db.Exec(query, vals...)

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

func (this *Table) SelectRowSchema(columns []string, where Filter, dest interface{}) error {
    var buffer bytes.Buffer

    buffer.WriteString("SELECT ")

    if columns == nil {
        buffer.WriteString("*")
    } else {
        first := true
        for _, col := range columns {
            if !first {
                buffer.WriteString(", ")
            }
            buffer.WriteString(col)

            first = false
        }
    }

    buffer.WriteString(" FROM ") 
    buffer.WriteString(this.table)

    whereVals := make([]interface{}, len(where))
    if where != nil {
        i := 1
        buffer.WriteString(" WHERE ")

        for col, val := range where {
            if i != 1 {
                buffer.WriteString(" && ")
            }

            buffer.WriteString(col)
            buffer.WriteString(" = ")
            buffer.WriteString(fmt.Sprintf("($%d)", i))

            whereVals[i - 1] = val
            i += 1
        }
    }

    buffer.WriteString(";")
    query := buffer.String()
    row := this.db.QueryRow(query, whereVals...)

    if row == nil {
        return errors.New("unknown database error")
    }

    colCount := len(this.schema)
    if columns != nil {
        colCount = len(columns)
    }

    values := make([]interface{}, colCount)
    valuePtrs := make([]interface{}, colCount)

    for i := 0; i < len(values); i++ {
        valuePtrs[i] = &values[i]
    }

    err := row.Scan(valuePtrs...)
    if err != nil {
        return err
    }

    rv := reflect.ValueOf(dest).Elem()
    for i, colName := range columns {
        setVal := this.getValueOf(values[i], colName)
        rv.FieldByName(colName).Set(reflect.ValueOf(setVal))
    }

    if err != nil {
        return err
    }

    return err
}

//Deprecated (Really, don't ever use this. It's going away in a week)
func (this *Table) CustomSelect(query string, queryParams []string) (row *sql.Row) {
    var vals = make([]interface{}, len(queryParams))

    for i, param := range queryParams {
        vals[i] = param
    }

    row = this.db.QueryRow(query, vals)

    return
}

func (this *Table) SelectSchema(columns []string, filter Filter) (*Scanner, error) {
    var buffer bytes.Buffer

    buffer.WriteString("SELECT ")

    if columns == nil {
        buffer.WriteString("*")
    } else {
        first := true
        for _, col := range columns {
            if !first {
                buffer.WriteString(", ")
            }
            buffer.WriteString(col)

            first = false
        }
    }

    buffer.WriteString(" FROM ") 
    buffer.WriteString(this.table)

    var vals = make([]interface{}, len(filter))

    if len(filter) > 0 {
        buffer.WriteString(" WHERE ")

        i := 0

        for col, val := range filter {
            if i != 0 {
                buffer.WriteString(" && ")
            }

            buffer.WriteString(col)
            buffer.WriteString("=")
            buffer.WriteString(fmt.Sprintf("($%d)", i + 1))

            vals[i] = val
        }
    }

    buffer.WriteString(";")

    query := buffer.String()
    rows, err := this.db.Query(query, vals...)

    if err != nil {
        return nil, err
    }

    return &Scanner{rows, this}, nil
}
