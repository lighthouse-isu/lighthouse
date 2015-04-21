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
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
)

func SetupTestingDefaultConnection() {
	defaultConnection = testDB()
}

func TeardownTestingDefaultConnection() {
	defaultConnection = nil
}

type testDatabase struct {
	sql.DB
}

func (this *testDatabase) Compiler(schema Schema) Compiler {
	return &testCompiler{}
}

type testCompiler struct{}

func (t *testCompiler) ConvertInput(o interface{}, c string) interface{} {
	return o
}

func (t *testCompiler) ConvertOutput(o interface{}, c string) interface{} {
	return o
}

func (t *testCompiler) CompileCreate(tab string) string {
	return "CREATE"
}

func (t *testCompiler) CompileDrop(tab string) string {
	return "DROP"
}

func (t *testCompiler) CompileInsert(tab string, val map[string]interface{}) (string, []interface{}) {
	return "INSERT", []interface{}{}
}

func (t *testCompiler) CompileDelete(tab string, f Filter) (string, []interface{}) {
	return "DELETE", []interface{}{}
}

func (t *testCompiler) CompileUpdate(tab string, to map[string]interface{}, w Filter) (string, []interface{}) {
	return "UPDATE", []interface{}{}
}

func (t *testCompiler) CompileSelect(tab string, c []string, f Filter, o *SelectOptions) (string, []interface{}) {
	return "SELECT", []interface{}{}
}

func testDB() DBInterface {
	db, _ := sqlmock.New()
	return &testDatabase{*db}
}
