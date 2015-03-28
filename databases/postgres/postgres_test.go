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
    "testing"

    "github.com/lib/pq"

    "github.com/stretchr/testify/assert"
    "github.com/DATA-DOG/go-sqlmock"

    "github.com/lighthouse/lighthouse/databases"
)

var testSchema databases.Schema = databases.Schema{
    "Name" : "text UNIQUE PRIMARY KEY",
    "Age" : "integer",
    "Phone" : "text",
}

func Test_Connection(t *testing.T) {
	db, _ := sqlmock.New()
	connection = &postgresConn{*db}

	res := Connection()

	assert.Equal(t, connection, res)
}

func Test_Compiler(t *testing.T) {
	db, _ := sqlmock.New()
	conn := &postgresConn{*db}

	var inter interface{}

	schema := databases.Schema{"this" : "junk"}
	inter = conn.Compiler(schema)

	comp, ok := inter.(*postgresCompiler)

	assert.True(t, ok)
	assert.Equal(t, schema, comp.schema)
}

func Test_TransformError(t *testing.T) {
	tests := map[error]error {
		nil : nil,
		databases.EmptyKeyError : databases.EmptyKeyError,
		&pq.Error{Code : "23505"} : databases.DuplicateKeyError,
	}

	for test, key := range tests {
		res := transformError(test)
		assert.Equal(t, key, res)
	}
}

func Test_Exec(t *testing.T) {
	db, _ := sqlmock.New()
	conn := &postgresConn{*db}

	exec := `TEST EXEC`
	args := []interface{}{42, "TEST", false}

	sqlmock.ExpectExec(exec).
		WithArgs(42, "TEST", false).
		WillReturnResult(sqlmock.NewResult(42, 73))

	res, err := conn.Exec(exec, args...)

	assert.Nil(t, err)

	id, _ := res.LastInsertId()
	assert.Equal(t, 42, id)

	cnt, _ := res.RowsAffected()
	assert.Equal(t, 73, cnt)
}

func Test_CompileInsert(t *testing.T) {
    values := map[string]interface{}{"Age" : 1, "Name" : "Sam"}

	comp := &postgresCompiler{testSchema}
	exec, vars := comp.CompileInsert("TABLE", values)

    key := "INSERT INTO TABLE (Age, Name) VALUES (($1), ($2));"
    assert.Equal(t, key, exec)
    assert.Equal(t, 1, vars[0])
    assert.Equal(t, "Sam", vars[1])
}

func Test_CompileDelete(t *testing.T) {
    where := map[string]interface{}{"Age" : 1, "Name" : "Sam"}

	comp := &postgresCompiler{testSchema}
	delete, vars := comp.CompileDelete("TABLE", where)

    key := "DELETE FROM TABLE WHERE Age = ($1) AND Name = ($2);"
    
    assert.Equal(t, key, delete)
    assert.Equal(t, 1, vars[0])
    assert.Equal(t, "Sam", vars[1])
}

func Test_CompileDelete_All(t *testing.T) {
	comp := &postgresCompiler{testSchema}
	delete, vars := comp.CompileDelete("TABLE", nil)

    key := "DELETE FROM TABLE;"
    
    assert.Equal(t, key, delete)
    assert.Equal(t, 0, len(vars))
}

func Test_CompileUpdate(t *testing.T) {
    to := map[string]interface{}{"Phone" : "123-456-7890", "Name" : "Pete"}
    where := databases.Filter{"Age" : 1, "Name" : "Sam"}

    comp := &postgresCompiler{testSchema}
    update, vars := comp.CompileUpdate("TABLE", to, where)

    key := "UPDATE TABLE SET Name = ($1), Phone = ($2) WHERE Age = ($3) AND Name = ($4);"

    assert.Equal(t, key, update)
    assert.Equal(t, "Pete", vars[0])
    assert.Equal(t, "123-456-7890", vars[1])
    assert.Equal(t, 1, vars[2])
    assert.Equal(t, "Sam", vars[3])
}

func Test_CompileSelect_Default(t *testing.T) {
    columns := []string {"Phone", "Name"}
    where := databases.Filter {"Age" : 1, "Name" : "Sam"}

    comp := &postgresCompiler{testSchema}
    query, vars := comp.CompileSelect("TABLE", columns, where, nil)

    key := "SELECT Phone, Name FROM TABLE WHERE Age = ($1) AND Name = ($2);"

    assert.Equal(t, key, query)
    assert.Equal(t, 1, vars[0])
    assert.Equal(t, "Sam", vars[1])
}

func Test_CompileQuery_Options(t *testing.T) {
    opts := databases.SelectOptions{
        Distinct: true,
        Top: 42,
        OrderBy: []string{"C1", "C2"},
        Desc: true,
    }

    comp := &postgresCompiler{testSchema}
    res, vars := comp.CompileSelect("TABLE", []string{"Name"}, nil, &opts)

    key := "SELECT DISTINCT Name FROM TABLE ORDER BY C1, C2 DESC LIMIT 42;"

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
    comp := &postgresCompiler{shema}

    for trial, pair := range tests {
        shema["COLUMN"] = trial
        res := comp.ConvertInput(pair.Test, "COLUMN")
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
    comp := &postgresCompiler{shema}

    for trial, pair := range tests {
        shema["COLUMN"] = trial
        res := comp.ConvertOutput(pair.Test, "COLUMN")
        assert.Equal(t, pair.Key, res)
    }
}