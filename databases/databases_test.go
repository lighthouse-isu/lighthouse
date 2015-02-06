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
    "errors"
    "database/sql"

    "github.com/stretchr/testify/assert"
)

type testParameterPair struct {
    query string
    args []interface{}
}

type testCallData map[string]testParameterPair

func makeTestingDatabase(t *testing.T) (*MockDatabase, testCallData) {
    db := &MockDatabase{}
    call := make(testCallData)

    db.MockExec = func(s string, i ...interface{}) (sql.Result, error) {
        call["exec"] = testParameterPair{s, i}
        return nil, errors.New("junk")
    }
    db.MockQueryRow = func(s string, i ...interface{}) (*sql.Row) {
        call["query_row"] = testParameterPair{s, i}
        return nil
    }

    return db, call
}

func Test_NewTable(t *testing.T) {
    db, call := makeTestingDatabase(t)
    table := NewTable(db, "test_table")

    assert.Equal(t, db, table.db, "Database pointer not stored properly")
    assert.Equal(t, "test_table", table.table, "Table name not stored properly")

    q := call["exec"].query

    assert.True(t, strings.Contains(q, "CREATE"),
        "Database setup should CREATE table")
    assert.True(t, strings.Contains(q, "test_table"),
        "Table creation Exec call should contain table name")
}

func Test_Insert(t *testing.T) {
    db, call := makeTestingDatabase(t)
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

func Test_Update(t *testing.T) {
    db, call := makeTestingDatabase(t)
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

func Test_SelectRow(t *testing.T) {
    db, call := makeTestingDatabase(t)
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
