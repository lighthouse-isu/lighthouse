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
	"database/sql/driver"
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

	Compiler(Schema) Compiler
}

type TableInterface interface {
	Insert(map[string]interface{}) error
	InsertReturn(map[string]interface{}, []string, *SelectOptions, interface{}) error
	Delete(Filter) error
	Update(map[string]interface{}, Filter) error
	SelectRow([]string, Filter, *SelectOptions, interface{}) error
	Select([]string, Filter, *SelectOptions) (ScannerInterface, error)
	Reload()
}

type ScannerInterface interface {
	Close() error
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(dest interface{}) error
}

type Compiler interface {
	ConvertInput(interface{}, string) interface{}
	ConvertOutput(interface{}, string) interface{}

	CompileCreate(string) string
	CompileDrop(string) string
	CompileInsert(string, map[string]interface{}) (string, []interface{})
	CompileDelete(string, Filter) (string, []interface{})
	CompileUpdate(string, map[string]interface{}, Filter) (string, []interface{})
	CompileSelect(string, []string, Filter, *SelectOptions) (string, []interface{})
}
