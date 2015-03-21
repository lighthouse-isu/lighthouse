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
    "encoding/json"

    "github.com/stretchr/testify/assert"

    "github.com/DATA-DOG/go-sqlmock"
)

var testSchema Schema = Schema{
    "Name" : "text UNIQUE PRIMARY KEY",
    "Age" : "integer",
    "Phone" : "text",
}

type testObject struct {
    Name string
    Age int
    Phone string
}

func Test_NewTable(t *testing.T) {
    db, _ := sqlmock.New()

    sqlmock.ExpectExec(`CREATE TABLE test_table (keyColumn text UNIQUE PRIMARY KEY, valueColumn json);`)

    var inter interface{}
    inter = NewTable(db, "test_table")
    table := inter.(*Table)
    table.Reload()

    assert.Nil(t, table.schema)
    assert.Equal(t, "test_table", table.table)
    assert.Equal(t, db, table.db)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_NewSchemaTable(t *testing.T) {
    db, _ := sqlmock.New()

    keyCols := `Name text UNIQUE PRIMARY KEY, Age integer, About json`

    sqlmock.ExpectExec(`CREATE TABLE test_table (` + keyCols + `);`)

    var inter interface{}
    inter = NewSchemaTable(db, "test_table", testSchema)
    table := inter.(*Table)
    table.Reload()

    assert.Equal(t, testSchema, table.schema)
    assert.Equal(t, "test_table", table.table)
    assert.Equal(t, db, table.db)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_NewSchemaTable_Panic(t *testing.T) {
    defer func() { recover() }()

    db, _ := sqlmock.New()
    NewSchemaTable(db, "test_table", nil)

    t.Errorf("Nil schema in NewSchemaTable should panic")
}

func Test_Insert(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", nil}

    value, _ := json.Marshal("VALUE")
    sqlmock.ExpectExec(`INSERT INTO test_table (.+) VALUES (.+);`).
        WithArgs("KEY", string(value)).
        WillReturnResult(sqlmock.NewResult(0, 1))

    err := table.Insert("KEY", "VALUE")
    if err != nil {
        t.Errorf(err.Error())
    }

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_InsertSchema_WithReturn(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    newData := map[string]interface{}{
        "Name" : "John Doe",
        "Age" : 42,
    }

    sqlmock.ExpectQuery(`INSERT INTO test_table (.+) VALUES (.+) RETURNING Age;`).
        WithArgs(42, "John Doe").
        WillReturnRows(sqlmock.NewRows([]string{"Age"}).AddRow(1))

    retval, err := table.InsertSchema(newData, "Age")

    assert.Nil(t, err)
    assert.Equal(t, 1, retval)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_InsertSchema_NoReturn(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    newData := map[string]interface{}{
        "Name" : "John Doe",
        "Age" : 42,
    }

    sqlmock.ExpectExec(`INSERT INTO test_table (.+) VALUES (.+);`).
        WithArgs(42, "John Doe").
        WillReturnResult(sqlmock.NewResult(0, 1))

    retval, err := table.InsertSchema(newData, "")

    assert.Nil(t, err)
    assert.Nil(t, retval)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_InsertSchema_NoInsert(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    newData := map[string]interface{}{
        "Name" : "John Doe",
        "Age" : 42,
    }

    sqlmock.ExpectExec(`INSERT INTO test_table (.+) VALUES (.+);`).
        WithArgs(42, "John Doe").
        WillReturnResult(sqlmock.NewResult(0, 0))

    _, err := table.InsertSchema(newData, "")

    assert.Equal(t, NoUpdateError, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_Update(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", nil}

    value, _ := json.Marshal("VALUE")
    sqlmock.ExpectExec(`UPDATE test_table SET (.+) WHERE keyColumn = (.+);`).
        WithArgs("KEY", string(value)).
        WillReturnResult(sqlmock.NewResult(0, 1))

    err := table.Update("KEY", "VALUE")
    assert.Nil(t, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_UpdateSchema(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    to := map[string]interface{} {
        "Name": "Jane Doe",
        "Age" : 42,
    }

    where := map[string]interface{} {
        "Age" : 41,
        "Phone" : "123-456-7890",
    }

    sqlmock.ExpectExec(`UPDATE test_table SET (.+) WHERE (.+);`).
        WithArgs(42, "Jane Doe", 41, "123-456-7890").
        WillReturnResult(sqlmock.NewResult(0, 1))

    err := table.UpdateSchema(to, where)
    assert.Nil(t, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_SelectRow(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", nil}

    key := []string{"PASS"}
    res := []string{}

    jsonBuff, _ := json.Marshal(key)

    sqlmock.ExpectQuery(`SELECT valueColumn FROM test_table WHERE keyColumn = .+;`).
        WithArgs("KEY").
        WillReturnRows(sqlmock.NewRows([]string{"valueColumn"}).AddRow(jsonBuff))

    err := table.SelectRow("KEY", &res)

    assert.Equal(t, key, res)
    assert.Nil(t, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_SelectRowSchema(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    columns := []string {"Phone", "Name"}
    filter := Filter {
        "Age" : 1,
    }

    sqlmock.ExpectQuery(`SELECT Phone, Name FROM test_table WHERE Age = .+;`).
        WithArgs(1).
        WillReturnRows(sqlmock.NewRows(columns).AddRow([]byte("123-456-7890"), []byte("Sam")))

    key := testObject{"Sam", 0, "123-456-7890"}

    var res testObject
    err := table.SelectRowSchema(columns, filter, &res)

    assert.Equal(t, key, res)
    assert.Nil(t, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_SelectSchema(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    columns := []string {"Phone", "Name"}
    filter := Filter {
        "Age" : 1,
    }

    key := []testObject{
        testObject{"Sam", 0, "123-456-7890"},
        testObject{"Sue", 0, "314-151-9285"},
        testObject{"Bob", 0, "319-256-7380"},
    }

    sqlmock.ExpectQuery(`SELECT Phone, Name FROM test_table WHERE Age = .+;`).
        WithArgs(1).
        WillReturnRows(sqlmock.NewRows(columns).
            AddRow([]byte(key[0].Phone), []byte(key[0].Name)).
            AddRow([]byte(key[1].Phone), []byte(key[1].Name)).
            AddRow([]byte(key[2].Phone), []byte(key[2].Name)))

    var res testObject
    scan, err := table.SelectSchema(columns, filter, SelectOptions{})

    assert.Nil(t, err)

    for i := 0; i < 3 && scan.Next(); i += 1 {
        scan.Scan(&res)
        assert.Equal(t, key[i], res)
    }

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_SelectSchema_Options(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    columns := []string {"Phone", "Name"}

    sqlmock.ExpectQuery(`SELECT DISTINCT .+;`).
        WillReturnRows(sqlmock.NewRows(columns))

    _, err := table.SelectSchema(columns, nil, SelectOptions{Distinct: true})

    assert.Nil(t, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_SelectSchema_Star(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    columns := []string {"Phone", "Name"}
    filter := Filter {
        "Age" : 1,
    }

    sqlmock.ExpectQuery(`SELECT \* .+;`).
        WithArgs(1).
        WillReturnRows(sqlmock.NewRows(columns))

    _, err := table.SelectSchema(nil, filter, SelectOptions{})

    assert.Nil(t, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_SelectRowSchema_Star(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    columns := []string {"Age", "Name", "Phone"}
    filter := Filter {
        "Age" : 1,
    }

    var data testObject

    sqlmock.ExpectQuery(`SELECT \* .+;`).
        WithArgs(1).
        WillReturnRows(sqlmock.NewRows(columns).AddRow(int64(1), []byte("Sam"), []byte("123-456-7890")))

    err := table.SelectRowSchema(nil, filter, &data)

    assert.Nil(t, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_DeleteRowsSchema(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    where := map[string]interface{} {
        "Age" : 41,
        "Phone" : "123-456-7890",
    }

    sqlmock.ExpectExec(`DELETE FROM test_table WHERE (.+);`).
        WithArgs(41, "123-456-7890").
        WillReturnResult(sqlmock.NewResult(0, 1))

    err := table.DeleteRowsSchema(where)
    assert.Nil(t, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_DeleteRowsNil(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    sqlmock.ExpectExec(`DELETE FROM test_table;`).
        WillReturnResult(sqlmock.NewResult(0, 1))

    err := table.DeleteRowsSchema(nil)
    assert.Nil(t, err)

    if err := db.Close(); err != nil {
        t.Errorf(err.Error())
    }
}

func Test_DeleteRowsNoUpdate(t *testing.T) {
    db, _ := sqlmock.New()
    table := &Table{db, "test_table", testSchema}

    sqlmock.ExpectExec(`.*`).
        WillReturnResult(sqlmock.NewResult(0, 0))

    err := table.DeleteRowsSchema(nil)

    assert.Equal(t, NoUpdateError, err)
}

func Test_BuildQueryFrom_Normal(t *testing.T) {
    columns := []string {"Phone", "Name"}
    filter := Filter {"Age" : 1, "Name" : "Sam"}

    res, vars := buildQueryFrom("TABLE", columns, filter, SelectOptions{})

    key := "SELECT Phone, Name FROM TABLE WHERE "

    assert.True(t, strings.HasPrefix(res, key))
    res = res[len(key):]

    assert.True(t, strings.Contains(res, "Age") && strings.Contains(res, "Name"))

    if strings.HasPrefix(res, "Age") {
        assert.Equal(t, 1, vars[0])
        assert.Equal(t, "Sam", vars[1])
    } else {
        assert.Equal(t, "Sam", vars[0])
        assert.Equal(t, 1, vars[1])
    } 
}

func Test_BuildQueryFrom_NilsAndOptions(t *testing.T) {
    res, vars := buildQueryFrom("TABLE", nil, nil, SelectOptions{Distinct: true})

    key := "SELECT DISTINCT * FROM TABLE;"

    assert.Equal(t, key, res)
    assert.Equal(t, 0, len(vars))
}

func Test_ConvertInput(t *testing.T) {
    type TestKeyPair struct{
        Test interface{}
        Key interface{}
    }

    tests := map[string]TestKeyPair {
        "text" : TestKeyPair{"STRING_TEST", "STRING_TEST"},
        "integer" : TestKeyPair{42, 42},
        "json" : TestKeyPair{[]string{"TEST"}, `["TEST"]`},
    }

    shema := map[string]string{}

    db, _ := sqlmock.New()
    table := &Table{db, "junk", shema}

    for trial, pair := range tests {
        shema["COLUMN"] = trial
        res := table.convertInput(pair.Test, "COLUMN")
        assert.Equal(t, pair.Key, res)
    }
}

func Test_ConvertOutput(t *testing.T) {
    type TestKeyPair struct{
        Test interface{}
        Key interface{}
    }

    tests := map[string]TestKeyPair {
        "text" : TestKeyPair{[]byte("STRING_TEST"), "STRING_TEST"},
        "integer" : TestKeyPair{int64(42), 42},
        "json" : TestKeyPair{[]byte(`["TEST"]`), []interface{}{"TEST"}, },
    }

    shema := map[string]string{}

    db, _ := sqlmock.New()
    table := &Table{db, "junk", shema}

    for trial, pair := range tests {
        shema["COLUMN"] = trial
        res := table.convertOutput(pair.Test, "COLUMN")
        assert.Equal(t, pair.Key, res)
    }
}
