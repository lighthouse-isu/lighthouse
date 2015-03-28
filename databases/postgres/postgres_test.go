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

    "strings"

    "github.com/stretchr/testify/assert"

    "github.com/lighthouse/lighthouse/databases"
)

var testSchema databases.Schema = databases.Schema{
    "Name" : "text UNIQUE PRIMARY KEY",
    "Age" : "integer",
    "Phone" : "text",
}

func Test_CompileSelect_Default(t *testing.T) {
    columns := []string {"Phone", "Name"}
    where := databases.Filter {"Age" : 1, "Name" : "Sam"}

    comp := &postgresCompiler{testSchema}
    res, vars := comp.CompileSelect("TABLE", columns, where, nil)

    keyQuery := "SELECT Phone, Name FROM TABLE WHERE "

    assert.True(t, strings.HasPrefix(res, keyQuery))
    res = res[len(keyQuery):]

    assert.True(t, strings.Contains(res, "Age") && strings.Contains(res, "Name"))

    if strings.HasPrefix(res, "Age") {
        assert.Equal(t, 1, vars[0])
        assert.Equal(t, "Sam", vars[1])
    } else {
        assert.Equal(t, "Sam", vars[0])
        assert.Equal(t, 1, vars[1])
    }
}

func Test_BuildQueryFrom_Options(t *testing.T) {
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