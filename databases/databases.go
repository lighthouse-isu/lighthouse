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
)

var (
    EmptyKeyError = errors.New("databases: given key was empty string")
    NoUpdateError = errors.New("databases: no rows updated")
    UnknownError = errors.New("databases: unknown error")
    KeyNotFoundError = errors.New("databases: given key not found")
    NoRowsError = errors.New("databases: result was empty")
)

type Table struct {
    db DBInterface
    table string
    schema Schema
}

type SelectOptions struct {
    Distinct bool
}

type Schema map[string]string
type Filter map[string]interface{}

const (
    keyColumn string = "keyColumn"
    valueColumn string = "valueColumn"
)

func NewTable(db DBInterface, table string) TableInterface {
    this := &Table{db, table, nil}
    this.drop()
    this.init()
    return this
}

func NewSchemaTable(db DBInterface, table string, schema Schema) TableInterface {
    this := &Table{db, table, schema}
    this.drop()
    this.initSchema()
    return this
}

func (this *Table) init() {
    query := fmt.Sprintf(`CREATE TABLE %s (%s text UNIQUE PRIMARY KEY, %s json);`,
        this.table, keyColumn, valueColumn)

    this.db.Exec(query)
}

func (this *Table) initSchema() {
    if len(this.schema) == 0 {
        panic("No schema given to database")
    }

    var buffer bytes.Buffer
    first := true

    for col, colType := range this.schema {
        if !first {
            buffer.WriteString(", ")
        }

        // JSON type doesn't have an equality operator which breaks queries
        if strings.Contains(colType, "json") {
            colType = "text"
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

func (this *Table) convertInput(orig interface{}, col string) interface{} {
    colType := this.schema[col]

    if strings.Contains(colType, "json") {
        b, _ := json.Marshal(orig)
        return string(b)
    }
    
    return orig
}

func (this *Table) convertOutput(orig interface{}, col string) interface{} {
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

        queryVals[i] = this.convertInput(val, col)

        i += 1
    }

    query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
        this.table, colBuf.String(), valBuf.String())
    res, err := this.db.Exec(query, queryVals...)

    if err == nil {
        cnt, err := res.RowsAffected()

        if err == nil && cnt < 1 {
            return NoUpdateError
        }
    }

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

        vals[i - 1] = this.convertInput(val, col)
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

    res, err := this.db.Exec(query, vals...)

    if err == nil {
        cnt, err := res.RowsAffected()

        if err == nil && cnt < 1 {
            return NoUpdateError
        }
    }

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

func buildQueryFrom(table string, columns []string, where Filter, opts SelectOptions) (string, []interface{})  {
    var buffer bytes.Buffer

    buffer.WriteString("SELECT ")

    if opts.Distinct {
        buffer.WriteString("DISTINCT ")
    }

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
    buffer.WriteString(table)

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

    return buffer.String(), whereVals
}

func (this *Table) SelectRowSchema(columns []string, where Filter, dest interface{}) error {

    if columns == nil || len(columns) == 0 {
        columns = make([]string, len(this.schema))

        i := 0
        for col, _ := range this.schema {
            columns[i] = col
            i += 1
        }
    }

    query, queryVals := buildQueryFrom(this.table, columns, where, SelectOptions{})
    row := this.db.QueryRow(query, queryVals...)

    if row == nil {
        return errors.New("unknown database error")
    }

    values := make([]interface{}, len(columns))
    valuePtrs := make([]interface{}, len(values))

    for i := 0; i < len(values); i++ {
        valuePtrs[i] = &values[i]
    }

    err := row.Scan(valuePtrs...)

    if err == sql.ErrNoRows {
        return NoRowsError
    }

    if err != nil {
        return err
    }

    rv := reflect.ValueOf(dest).Elem()
    for i, colName := range columns {
        setVal := this.convertOutput(values[i], colName)
        if setVal != nil {
            rv.FieldByName(colName).Set(reflect.ValueOf(setVal))
        }
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

func (this *Table) SelectSchema(columns []string, where Filter, opts SelectOptions) (ScannerInterface, error) {
    query, queryVals := buildQueryFrom(this.table, columns, where, opts)
    rows, err := this.db.Query(query, queryVals...)

    if err != nil {
        return nil, err
    }

    return &Scanner{*rows, this, columns}, nil
}
