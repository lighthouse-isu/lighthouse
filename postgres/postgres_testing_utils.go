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
    "database/sql"
    "database/sql/driver"
)

/*
    The purpose of MockDatabase is to be able to create custom mocks for datatbases
    that can be directly used in a database.Database
*/

type MockDatabase struct {
    MockBegin           func()(*sql.Tx, error)
    MockClose           func() error
    MockDriver          func() driver.Driver
    MockExec            func(string, ...interface{}) (sql.Result, error)
    MockPing            func() error
    MockPrepare         func(string) (*sql.Stmt, error)
    MockQuery           func(string, ...interface{}) (*sql.Rows, error)
    MockQueryRow        func(string, ...interface{}) *sql.Row
    MockSetMaxIdleConns func(int)
}

func (t *MockDatabase) Begin()(*sql.Tx, error) {
    return t.MockBegin()
}

func (t *MockDatabase) Close() (error) {
    return t.MockClose()
}

func (t *MockDatabase) Driver() (driver.Driver) {
    return t.MockDriver()
}

func (t *MockDatabase) Exec(s string, i ...interface{}) (sql.Result, error) {
    return t.MockExec(s, i)
}

func (t *MockDatabase) Ping() (e error) {
    return t.MockPing()
}

func (t *MockDatabase) Prepare(s string) (*sql.Stmt, error) {
    return t.MockPrepare(s)
}

func (t *MockDatabase) Query(s string, i ...interface{}) (*sql.Rows, error) {
    return t.MockQuery(s, i)
}

func (t *MockDatabase) QueryRow(s string, i ...interface{}) (*sql.Row) {
    return t.MockQueryRow(s, i)
}

func (t *MockDatabase) SetMaxIdleConns(i int) () {
    t.MockSetMaxIdleConns(i)
}
