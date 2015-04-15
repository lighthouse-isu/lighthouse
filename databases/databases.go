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
	"errors"
	"reflect"
	"sort"
	"sync"

	"database/sql"
)

var (
	EmptyKeyError     = errors.New("databases: given key was empty string")
	NoUpdateError     = errors.New("databases: no rows updated")
	UnknownError      = errors.New("databases: unknown error")
	KeyNotFoundError  = errors.New("databases: given key not found")
	NoRowsError       = errors.New("databases: result was empty")
	DuplicateKeyError = errors.New("databases: key already exists")
)

type Table struct {
	db       DBInterface
	table    string
	schema   Schema
	compiler Compiler
	mutex    *sync.Mutex
}

type SelectOptions struct {
	Distinct bool
	Top      int
	OrderBy  []string
	Desc     bool
}

type Schema map[string]string
type Filter map[string]interface{}

var (
	defaultConnection DBInterface
)

func SetDefaultConnection(conn DBInterface) {
	defaultConnection = conn
}

func DefaultConnection() DBInterface {
	return defaultConnection
}

func DefaultSelectOptions() *SelectOptions {
	return &SelectOptions{}
}

func NewTable(db DBInterface, table string, schema Schema) *Table {
	if db == nil {
		db = defaultConnection
	}

	if len(schema) == 0 {
		panic("No schema given to database")
	}

	this := &Table{db, table, schema, db.Compiler(schema), nil}
	return this
}

func NewLockingTable(db DBInterface, table string, schema Schema) *Table {
	var this *Table
	this = NewTable(db, table, schema)
	this.mutex = &sync.Mutex{}
	return this
}

func (this *Table) Reload() {
	this.drop()
	this.init()
}

func (this *Table) init() {
	exec := this.compiler.CompileCreate(this.table)
	this.db.Exec(exec)
}

func (this *Table) drop() {
	exec := this.compiler.CompileDrop(this.table)
	this.db.Exec(exec)
}

func (this *Table) allColumns() []string {
	columns := make([]string, len(this.schema))

	i := 0
	for col, _ := range this.schema {
		columns[i] = col
		i += 1
	}

	sort.Strings(columns)
	return columns
}

func (this *Table) Insert(values map[string]interface{}) error {
	if this.mutex != nil {
		this.mutex.Lock()
		defer this.mutex.Unlock()
	}

	query, queryVals := this.compiler.CompileInsert(this.table, values)
	res, err := this.db.Exec(query, queryVals...)

	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err == nil && cnt < 1 {
		return NoUpdateError
	}

	return nil
}

func (this *Table) InsertReturn(values map[string]interface{}, cols []string, opts *SelectOptions, dest interface{}) error {
	if this.mutex == nil {
		panic("Only LockingTables can perform InsertReturns")
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	query, queryVals := this.compiler.CompileInsert(this.table, values)
	res, err := this.db.Exec(query, queryVals...)

	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err == nil && cnt < 1 {
		return NoUpdateError
	}

	return this.SelectRow(cols, values, opts, dest)
}

func (this *Table) Delete(where Filter) error {

	query, vals := this.compiler.CompileDelete(this.table, where)
	res, err := this.db.Exec(query, vals...)

	if err == nil {
		cnt, err := res.RowsAffected()

		if err == nil && cnt < 1 {
			return NoUpdateError
		}
	}

	return err
}

func (this *Table) Update(to map[string]interface{}, where Filter) error {
	query, vals := this.compiler.CompileUpdate(this.table, to, where)
	res, err := this.db.Exec(query, vals...)

	if err == nil {
		cnt, err := res.RowsAffected()

		if err == nil && cnt < 1 {
			return NoUpdateError
		}
	}

	return err
}

func (this *Table) SelectRow(columns []string, where Filter, opts *SelectOptions, dest interface{}) error {
	if len(columns) == 0 {
		columns = this.allColumns()
	}

	query, queryVals := this.compiler.CompileSelect(this.table, columns, where, opts)
	row := this.db.QueryRow(query, queryVals...)

	if row == nil {
		return UnknownError
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
		setVal := this.compiler.ConvertOutput(values[i], colName)
		if setVal != nil {
			rv.FieldByName(colName).Set(reflect.ValueOf(setVal))
		}
	}

	return err
}

func (this *Table) Select(columns []string, where Filter, opts *SelectOptions) (ScannerInterface, error) {
	if len(columns) == 0 {
		columns = this.allColumns()
	}

	query, queryVals := this.compiler.CompileSelect(this.table, columns, where, opts)
	rows, err := this.db.Query(query, queryVals...)

	if err != nil {
		return nil, err
	}

	return &Scanner{*rows, this, columns}, nil
}
