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
    "testing"
    "strings"
    "fmt"
    "errors"
    "database/sql"

    "github.com/stretchr/testify/assert"
)

type testParameterPair struct {
    query string
    args []interface{}
}

type testCallData map[string]testParameterPair

func makeTestingDatabase(t *testing.T) (*MockDatabase, testCallData, Schema) {
    db := &MockDatabase{}
    call := make(testCallData)
    schema := Schema {
        "Id" : "serial primary key",
        "Name" : "text",
        "Age" : "integer",
    }

    db.MockExec = func(s string, i ...interface{}) (sql.Result, error) {
        call["exec"] = testParameterPair{s, i}
        return nil, errors.New("junk")
    }
    db.MockQueryRow = func(s string, i ...interface{}) (*sql.Row) {
        call["query_row"] = testParameterPair{s, i}
        return nil
    }
    db.MockQuery = func(s string, i ...interface{}) (*sql.Rows, error) {
        call["query"] = testParameterPair{s, i}
        return nil, errors.New("junk")
    }

    return db, call, schema
}

func Test_NewTable(t *testing.T) {
    db, call, _ := makeTestingDatabase(t)
    NewTable(db, "test_table")

    q := call["exec"].query

    assert.True(t, strings.Contains(q, "CREATE"),
        "Database setup should CREATE table")
    assert.True(t, strings.Contains(q, "test_table"),
        "Table creation Exec call should contain table name")
}

func Test_NewSchemaTable(t *testing.T) {
    db, call, schema := makeTestingDatabase(t)
    NewSchemaTable(db, "test_table", schema)

    q := call["exec"].query

    assert.True(t, strings.Contains(q, "CREATE"), 
        "Database setup should CREATE table")
    assert.True(t, strings.Contains(q, "test_table"),
        "Table creation Exec call should contain table name")

    for col, dataType := range schema {
        assert.True(t, strings.Contains(q, col),
            fmt.Sprintf("Table should be created with schema column %s", col))
        assert.True(t, strings.Contains(q, dataType),
            fmt.Sprintf("Table should create %s with type %s", col, dataType))
    }
}

func Test_Insert(t *testing.T) {
    db, call, _ := makeTestingDatabase(t)
    table := NewTable(db, "test_table")

    table.Insert("TEST_KEY", "TEST_VAL")

    q := call["exec"].query
    args := call["exec"].args[0].([]interface{})

    assert.True(t, strings.Contains(q, "INSERT"),
        "Insert query should contain INSERT")
    assert.True(t, strings.Contains(q, "test_table"),
        "Insertion Exec call should contain table name")

    assert.Equal(t, "TEST_KEY", args[0].(string),
        "Insertion call should add given key")
    assert.Equal(t, "\"TEST_VAL\"", args[1].(string),
        "Insertion call should encode given value as JSON")
}

//TODO
//Will not run because we can't mock sql.Row
//We need to write a database driver...
//Or use someone else's
/*
func Test_InsertSchema_WithReturn(t *testing.T) {
    db, call, schema := makeTestingDatabase(t)
    table := NewSchemaTable(db, "test_table", schema)

    newData := map[string]interface{}{
        "Name" : "John Doe",
        "Age" : 42,
    }

    retval, _ := table.InsertSchema(newData, "Id")

    q := call["query_row"].query
    args := call["query_row"].args[0].([]interface{})

    assert.True(t, strings.Contains(q, "INSERT"),
        "Insert query should contain INSERT")
    assert.True(t, strings.Contains(q, "test_table"),
        "Insertion Exec call should contain table name")

    var firstCol, lastCol string

    assert.True(t, strings.Index(q, "Id") < 0, "Query should not contain 'Id' column")
    assert.True(t, strings.Index(q, "Name") >= 0, "Query should contain 'Name' column")
    assert.True(t, strings.Index(q, "Age") >= 0, "Query should contain 'Age' column")

    firstCol = firstSubstr(q, "Age", "Name")
    lastCol = lastSubstr(q, "Age", "Name")

    assert.Equal(t, newData[firstCol], args[0],
        fmt.Sprintf("Query param for %s is incorrect", firstCol))
    assert.Equal(t, newData[lastCol], args[1],
        fmt.Sprintf("Query param for %s is incorrect", lastCol))
}
*/

func Test_InsertSchema_Plain(t *testing.T) {
    db, call, schema := makeTestingDatabase(t)
    table := NewSchemaTable(db, "test_table", schema)
    
    newData := map[string]interface{}{
        "Id" : "0",
        "Name" : "John Doe",
        "Age" : 42,
    }
    
    revData := map[interface{}]string {
        "0" : "Id",
        "John Doe" : "Name",
        42 : "Age",
    }

    table.InsertSchema(newData, "")

    q := call["exec"].query
    args := call["exec"].args[0].([]interface{})

    assert.True(t, strings.Contains(q, "INSERT"),
        "Insert query should contain INSERT")
    assert.True(t, strings.Contains(q, "test_table"),
        "Insertion Exec call should contain table name")
    
    var firstCol, lastCol string
    
    assert.True(t, strings.Index(q, "Id") >= 0, "Query should contain 'Id' column")
    assert.True(t, strings.Index(q, "Name") >= 0, "Query should contain 'Name' column")
    assert.True(t, strings.Index(q, "Age") >= 0, "Query should contain 'Age' column")
    
    firstCol = firstSubstr(q, firstSubstr(q, "Id", "Name"), "Age")
    lastCol = lastSubstr(q, lastSubstr(q, "Id", "Name"), "Age")

    assert.Equal(t, newData[firstCol], args[0],
        fmt.Sprintf("Query param for %s is incorrect", firstCol))
    assert.Equal(t, newData[lastCol], args[2],
        fmt.Sprintf("Query param for %s is incorrect", lastCol))

    midCol := revData[args[1]] != firstCol &&
              revData[args[1]] != lastCol &&
              revData[args[1]] != ""

    assert.True(t, midCol,
        fmt.Sprintf("Middle query param %v is incorrect", args[1]))
}

func firstSubstr(src, a, b string) string {
    if strings.Index(src, a) <= strings.Index(src, b) {
        return a
    }
    return b
}

func lastSubstr(src, a, b string) string {
    if strings.Index(src, a) >= strings.Index(src, b) {
        return a
    }
    return b
}

func Test_Update(t *testing.T) {
    db, call, _ := makeTestingDatabase(t)
    table := NewTable(db, "test_table")

    table.Update("TEST_KEY", "TEST_VAL")

    q := call["exec"].query
    args := call["exec"].args[0].([]interface{})

    assert.True(t, strings.Contains(q, "UPDATE"),
        "Update query should contain UPDATE")
    assert.True(t, strings.Contains(q, "test_table"),
        "Update Exec call should contain table name")

    assert.Equal(t, "TEST_KEY", args[0].(string),
        "Insertion call should add given key")
    assert.Equal(t, "\"TEST_VAL\"", args[1].(string),
        "Insertion call should encode given value as JSON")
}

func Test_UpdateSchema(t *testing.T) {
    db, call, schema := makeTestingDatabase(t)
    table := NewSchemaTable(db, "test_table", schema)
    
    to := map[string]interface{} {
        "Name": "Jane Doe",
    }

    where := map[string]interface{} {
        "Id" : 1,
    }

    table.UpdateSchema(to, where)
    q := call["exec"].query
    args := call["exec"].args[0].([]interface{})

    assert.True(t, strings.Contains(q, "UPDATE"),
        "Update query should contain UPDATE")
    assert.True(t, strings.Contains(q, "test_table"),
        "Update Exec call should contain table name")
    assert.True(t, strings.Contains(q, "Name"),
        "Update query should to contain column name")
    assert.True(t, strings.Contains(q, "Id"),
        "Update query should contain where column name")
    assert.True(t, strings.Index(q, "Name") < strings.Index(q, "Id"),
        "Update query should contain to column before where column")

    assert.Equal(t, "Jane Doe", args[0].(string),
        "Update query should update name")
    assert.Equal(t, 1, args[1].(int),
        "Update query should look for Id")

}

func Test_SelectRow(t *testing.T) {
    db, call, _ := makeTestingDatabase(t)
    table := NewTable(db, "test_table")

    table.SelectRow("TEST_KEY", nil)

    q := call["query_row"].query
    args := call["query_row"].args[0].([]interface{})

    assert.True(t, strings.Contains(q, "SELECT"),
        "Row query should contain SELECT")
    assert.True(t, strings.Contains(q, "test_table"),
        "Query should contain table name")

    assert.Equal(t, "TEST_KEY", args[0].(string),
        "Query should be given correct key")
}

func Test_SelectRowSchema(t *testing.T) {
    db, call, schema := makeTestingDatabase(t)
    table := NewSchemaTable(db, "test_table", schema)

    columns := []string {"Id", "Name", "Age"}
    filter := Filter {
        "Id" : 1,
    }

    table.SelectRowSchema(columns, filter, nil)

    q := call["query_row"].query
    args := call["query_row"].args[0].([]interface{})

    assert.True(t, strings.Contains(q, "SELECT"),
        "Row query should contain SELECT")
    assert.True(t, strings.Contains(q, "test_table"),
        "Query should contain table name")

    for _, col := range columns {
        assert.True(t, strings.Contains(q, col),
            "Row query should contain column name")
    }

    assert.True(t, strings.Contains(q, "WHERE"),
        "Row query should contain WHERE")
    assert.True(t, strings.Contains(q, "Id"),
        "Row query should contain filter column")
    assert.Equal(t, 1, args[0].(int),
        "Row query parameters should contain filter values")
}

func Test_SelectSchema(t *testing.T) {
    db, call, schema := makeTestingDatabase(t)
    table := NewSchemaTable(db, "test_table", schema)

    columns := []string {"Id", "Name", "Age"}
    filter := Filter {
        "Id" : 1,
    }

    table.SelectSchema(columns, filter, SelectOptions{})

    q := call["query"].query
    args := call["query"].args[0].([]interface{})

    assert.True(t, strings.Contains(q, "SELECT"),
        "Row query should contain SELECT")
    assert.True(t, strings.Contains(q, "test_table"),
        "Query should contain table name")

    for _, col := range columns {
        assert.True(t, strings.Contains(q, col),
            "Row query should contain column name")
    }

    assert.True(t, strings.Contains(q, "WHERE"),
        "Row query should contain WHERE")
    assert.True(t, strings.Contains(q, "Id"),
        "Row query should contain filter column")
    assert.Equal(t, 1, args[0].(int),
        "Row query parameters should contain filter values")
}
