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
    "sort"
    "bytes"
    "strings"
    "encoding/json"
    "database/sql"

    "github.com/lib/pq"

    "github.com/lighthouse/lighthouse/logging"
    "github.com/lighthouse/lighthouse/databases"
)

type postgresConn struct {
    *sql.DB
}

type postgresCompiler struct {
    schema databases.Schema
}

var connection *postgresConn

func Connection() databases.DBInterface {
    if connection == nil {
        connection = setup()
    }
    return connection
}

func (this *postgresConn) Exec(cmd string, params ...interface{}) (sql.Result, error) {
    res, err := this.DB.Exec(cmd, params...)   
    err = transformError(err)

    return res, err
}

func (this *postgresConn) Compiler(schema databases.Schema) (databases.Compiler) {
    return &postgresCompiler{schema}
}

func transformError(err error) error {
    var pqErr *pq.Error = nil
    var ok bool = false

    if err != nil {
        pqErr, ok = err.(*pq.Error)
    } 

    if !ok {
        return err
    }

    // Code listed at https://github.com/lib/pq/blob/master/error.go
    switch pqErr.Code {
    case "23505": 
        return databases.DuplicateKeyError
    }

    return nil
}

func setup() *postgresConn {
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

    return &postgresConn{postgres}
}

func (this *postgresCompiler) CompileCreate(table string) string {
    var cols []string

    for col, colType := range this.schema {
        // JSON type doesn't have an equality operator which breaks queries
        if strings.Contains(colType, "json") {
            colType = "text"
        }

        cols = append(cols, fmt.Sprintf("%s %s", col, colType))
    }

    sort.Strings(cols)

    colStr := strings.Join(cols, ", ")

    return fmt.Sprintf(`CREATE TABLE %s (%s);`, table, colStr)
}

func (this *postgresCompiler) CompileDrop(table string) string {
    return fmt.Sprintf(`DROP TABLE %s;`, table)
}

func (this *postgresCompiler) CompileInsert(table string, values map[string]interface{}) (string, []interface{}) {
    var valBuf bytes.Buffer
    queryVals := make([]interface{}, len(values))
    i := 0
    var keys []string

    for col, _ := range values {
        keys = append(keys, col)
    }

    sort.Strings(keys)

    colBuf := strings.Join(keys, ", ")

    for _, col := range keys {
        val := values[col]
        if i != 0 {
            valBuf.WriteString(", ")
        }

        s := fmt.Sprintf(`($%d)`, i + 1)
        valBuf.WriteString(s)

        queryVals[i] = this.ConvertInput(val, col)

        i += 1
    }

    query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s);`,
        table, colBuf, valBuf.String())

    return query, queryVals
}

func (this *postgresCompiler) CompileDelete(table string, where databases.Filter) (string, []interface{}) {
    var buffer bytes.Buffer
    vals := make([]interface{}, len(where))

    buffer.WriteString("DELETE FROM ")
    buffer.WriteString(table)

    if len(where) > 0 {
        buffer.WriteString(" WHERE ")

        var whereKeys []string
        for col, _ := range where {
            whereKeys = append(whereKeys, col)
        }
        sort.Strings(whereKeys)

        for i, col := range whereKeys {
            val := where[col]
            if i != 0 {
                buffer.WriteString(" AND ")
            }

            buffer.WriteString(fmt.Sprintf("%s = ($%d)", col, i + 1))

            vals[i] = this.ConvertInput(val, col)
        }
    }

    buffer.WriteString(";")

    return buffer.String(), vals
}

func (this *postgresCompiler) CompileUpdate(table string, to map[string]interface{}, where databases.Filter) (string, []interface{}) {
    var buffer bytes.Buffer

    buffer.WriteString("UPDATE ")
    buffer.WriteString(table)
    buffer.WriteString(" SET ")

    vals := make([]interface{}, len(where) + len(to))
    var toKeys, whereKeys []string
    i := 1

    for col, _ := range to {
        toKeys = append(toKeys, col)
    }

    for col, _ := range where {
        whereKeys = append(whereKeys, col)
    }

    sort.Strings(toKeys)
    sort.Strings(whereKeys)

    for _, col := range toKeys {
        val := to[col]
        if i != 1 {
            buffer.WriteString(", ")
        }

        buffer.WriteString(fmt.Sprintf("%s = ($%d)", col, i))

        vals[i - 1] = this.ConvertInput(val, col)
        i += 1
    }

    buffer.WriteString(" WHERE ")

    for _, col := range whereKeys {
        val := where[col]
        if i != len(to) + 1 {
            buffer.WriteString(" AND ")
        }

        buffer.WriteString(fmt.Sprintf("%s = ($%d)", col, i))

        vals[i - 1] = this.ConvertInput(val, col)
        i += 1
    }

    buffer.WriteString(";")

    return buffer.String(), vals
}

func (this *postgresCompiler) CompileSelect(table string, cols []string, where databases.Filter, opts *databases.SelectOptions) (string, []interface{}) {
    if opts == nil {
        opts = databases.DefaultSelectOptions()
    }

    var buffer bytes.Buffer

    buffer.WriteString("SELECT ")

    if opts.Distinct {
        buffer.WriteString("DISTINCT ")
    }

    buffer.WriteString(strings.Join(cols, ", "))

    buffer.WriteString(" FROM ") 
    buffer.WriteString(table)

    whereVals := make([]interface{}, len(where))
    if where != nil {

        buffer.WriteString(" WHERE ")

        var whereKeys []string
        for col, _ := range where {
            whereKeys = append(whereKeys, col)
        }

        sort.Strings(whereKeys)

        for i, col := range whereKeys {
            val := where[col]
            if i != 0 {
                buffer.WriteString(" AND ")
            }

            buffer.WriteString(fmt.Sprintf("%s = ($%d)", col, i + 1))

            whereVals[i] = val
        }
    }

    if opts.OrderBy != nil {
        buffer.WriteString(" ORDER BY ")
        buffer.WriteString(strings.Join(opts.OrderBy, ", "))
    }

    if opts.Desc {
        buffer.WriteString(" DESC")
    }

    if opts.Top > 0 {
        buffer.WriteString(fmt.Sprintf(" LIMIT %d", opts.Top))
    }

    buffer.WriteString(";")

    return buffer.String(), whereVals
}

func (this *postgresCompiler) ConvertInput(orig interface{}, col string) interface{} {
    colType := this.schema[col]

    if strings.Contains(colType, "json") {
        b, _ := json.Marshal(orig)
        return string(b)
    }
    
    return orig
}

func (this *postgresCompiler) ConvertOutput(orig interface{}, col string) interface{} {
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

    if strings.Contains(colType, "int") {
        return int(orig.(int64))
    }
    
    return orig
}