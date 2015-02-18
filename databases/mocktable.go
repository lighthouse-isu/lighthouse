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

/*
    The purpose of MockTable is to be able to create custom mocks for datatbases
    that can be directly used in a database.Database
*/

type MockTable struct {
    Database [][]interface{}
    Schema map[string]int

    MockInsert          func(string, interface{})(error)
    MockUpdate          func(string, interface{})(error)
    MockSelectRow       func(string, interface{})(error)
    MockInsertSchema    func(map[string]interface{})(error)
    MockUpdateSchema    func(map[string]interface{}, map[string]interface{})(error)
    MockSelectRowSchema func([]string, Filter, interface{})(error)
    MockSelectSchema    func([]string, Filter, SelectOptions)(ScannerInterface, error)
}

func (t *MockTable) Insert(s string, i interface{})(e error) {
    if t.MockInsert != nil { return t.MockInsert(s, i) }
    return
}

func (t *MockTable) Update(s string, i interface{})(e error) {
    if t.MockUpdate != nil { return t.MockUpdate(s, i) }
    return
}

func (t *MockTable) SelectRow(s string, i interface{})(e error) {
    if t.MockSelectRow != nil { return t.MockSelectRow(s, i) }
    return
}

func (t *MockTable) InsertSchema(v map[string]interface{})(e error) {
    if t.MockInsertSchema != nil { return t.MockInsertSchema(v) }
    return
}

func (t *MockTable) UpdateSchema(to, w map[string]interface{})(e error) {
    if t.MockUpdateSchema != nil { return t.MockUpdateSchema(to, w) }
    return
}

func (t *MockTable) SelectRowSchema(c []string, w Filter, d interface{})(e error) {
    if t.MockSelectRowSchema != nil { return t.MockSelectRowSchema(c, w, d) }
    return
}

func (t *MockTable) SelectSchema(c []string, w Filter, opts SelectOptions)(s ScannerInterface, e error) {
    if t.MockSelectSchema != nil { return t.MockSelectSchema(c, w, opts) }
    return
}

func CommonTestingTable(schema Schema) *MockTable {
    table := &MockTable{Database: make([][]interface{}, 0), Schema: make(map[string]int)}

    i := 0
    for k, _ := range schema {
        table.Schema[k] = i
        i += 1
    }

    table.MockInsertSchema = func(values map[string]interface{})(error) {
        addition := make([]interface{}, len(table.Schema))

        for k, v := range values {
            addition[table.Schema[k]] = v
        }

        for _, row := range table.Database {
            if reflect.DeepEqual(row, addition) {
                return errors.New("duplicate row")
            }
        }

        table.Database = append(table.Database, addition)

        return nil
    }

    table.MockUpdateSchema = func(to, where map[string]interface{})(error) {
        updated := false
        for _, row := range table.Database {

            applies := true
            for col, val := range where {
                if row[table.Schema[col]] != val {
                    applies = false
                    break
                }
            }

            if applies {
                for col, val := range to {
                    row[table.Schema[col]] = val
                }
                updated = true
            }
        }

        if !updated {
            return errors.New("no update")
        }

        return nil
    }

    table.MockSelectRowSchema = func(cols []string, where Filter, dest interface{})(error) {

        if cols == nil {
            cols = make([]string, len(table.Schema))
            for col, i := range table.Schema {
                cols[i] = col
            }
        }

        for _, row := range table.Database {

            applies := true
            for col, val := range where {
                if row[table.Schema[col]] != val {
                    applies = false
                    break
                }
            }

            if applies {
                rv := reflect.ValueOf(dest).Elem()
                for _, col := range cols {
                    rv.FieldByName(col).Set(reflect.ValueOf(row[table.Schema[col]]))
                }
                return nil
            }
        }

        return errors.New("not found")
    }

    table.MockSelectSchema = func(cols []string, where Filter, opts SelectOptions)(ScannerInterface, error) {

        if cols == nil {
            cols = make([]string, len(table.Schema))
            for col, i := range table.Schema {
                cols[i] = col
            }
        }

        entries := make([][]interface{}, 0)

        for _, row := range table.Database {

            applies := true
            for col, val := range where {
                if row[table.Schema[col]] != val {
                    applies = false
                    break
                }
            }

            newEntry := make([]interface{}, len(cols))
            for i, col := range cols {
                newEntry[i] = row[table.Schema[col]]
            }

            if applies && opts.Distinct {
                for _, oldEntry := range entries {
                    if reflect.DeepEqual(newEntry, oldEntry) {
                        applies = false
                        break
                    }
                }
            }

            if applies {
                entries = append(entries, newEntry)
            }
        }

        return CommonTestingScanner(entries, cols), nil
    }

    return table
}