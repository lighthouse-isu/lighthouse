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

func (t *MockDatabase) Begin()(tx *sql.Tx, e error) {
    if t.MockBegin != nil { return t.MockBegin() }
    return
}

func (t *MockDatabase) Close() (e error) {
    if t.MockClose != nil { return t.MockClose() }
    return
}

func (t *MockDatabase) Driver() (d driver.Driver) {
    if t.MockDriver != nil { return t.MockDriver() }
    return
}

func (t *MockDatabase) Exec(s string, i ...interface{}) (r sql.Result, e error) {
    if t.MockExec != nil { return t.MockExec(s, i) }
    return
}

func (t *MockDatabase) Ping() (e error) {
    if t.MockPing != nil { return t.MockPing() }
    return
}

func (t *MockDatabase) Prepare(s string) (st *sql.Stmt, e error) {
    if t.MockPrepare != nil { return t.MockPrepare(s) }
    return
}

func (t *MockDatabase) Query(s string, i ...interface{}) (r *sql.Rows, e error) {
    if t.MockQuery != nil { return t.MockQuery(s, i) }
    return
}

func (t *MockDatabase) QueryRow(s string, i ...interface{}) (r *sql.Row) {
    if t.MockQueryRow != nil { return t.MockQueryRow(s, i) }
    return
}

func (t *MockDatabase) SetMaxIdleConns(i int) () {
    if t.MockSetMaxIdleConns != nil { t.MockSetMaxIdleConns(i) }
}

func CommonTestingDatabase() (*MockDatabase) {
    db := &MockDatabase{}

    db.MockExec = func(s string, i ...interface{}) (sql.Result, error) {
        return nil, errors.New("junk")
    }
    db.MockQueryRow = func(s string, i ...interface{}) (*sql.Row) {
        return nil
    }

    return db
}