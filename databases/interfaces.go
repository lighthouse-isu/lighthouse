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
}

type TableInterface interface {
    Insert(string, interface{})(error)
    Update(string, interface{})(error)
    SelectRow(string, interface{})(error)
    InsertSchema(map[string]interface{})(int, error)
    DeleteRowsSchema(Filter) (error)
    UpdateSchema(map[string]interface{}, map[string]interface{})(error)
    SelectRowSchema([]string, Filter, interface{})(error)
    SelectSchema([]string, Filter, SelectOptions)(ScannerInterface, error)
}

type ScannerInterface interface {
	Close()(error)
    Columns()([]string, error)
    Err()(error)
    Next()(bool)
    Scan(dest interface{})(error)
}
