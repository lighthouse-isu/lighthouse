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
)

type MockScanner struct {
	Rows        [][]interface{}
	ColumnNames []string
	Index       int

	MockClose   func() error
	MockColumns func() ([]string, error)
	MockErr     func() error
	MockNext    func() bool
	MockScan    func(interface{}) error
}

func (t *MockScanner) Close() (e error) {
	if t.MockClose != nil {
		return t.MockClose()
	}
	return
}

func (t *MockScanner) Columns() (s []string, e error) {
	if t.MockColumns != nil {
		return t.MockColumns()
	}
	return
}

func (t *MockScanner) Err() (e error) {
	if t.MockErr != nil {
		return t.MockErr()
	}
	return
}

func (t *MockScanner) Next() (b bool) {
	if t.MockNext != nil {
		return t.MockNext()
	}
	return
}

func (t *MockScanner) Scan(dest interface{}) (e error) {
	if t.MockScan != nil {
		return t.MockScan(dest)
	}
	return
}

func CommonTestingScanner(rows [][]interface{}, columns []string) *MockScanner {
	scanner := &MockScanner{Rows: rows, ColumnNames: columns, Index: -1}

	scanner.MockNext = func() bool {
		scanner.Index += 1
		return scanner.Index < len(scanner.Rows)
	}

	scanner.MockColumns = func() ([]string, error) {
		return scanner.ColumnNames, nil
	}

	scanner.MockScan = func(dest interface{}) error {

		if scanner.Index >= len(scanner.Rows) {
			return errors.New("out of bounds")
		}

		row := scanner.Rows[scanner.Index]

		rv := reflect.ValueOf(dest).Elem()
		for i, colName := range scanner.ColumnNames {
			rv.FieldByName(colName).Set(reflect.ValueOf(row[i]))
		}

		return nil
	}

	return scanner
}
