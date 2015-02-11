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

import ()

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
    MockSelectSchema    func([]string, Filter)(*Scanner, error)
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

func (t *MockTable) SelectSchema(c []string, w Filter)(s *Scanner, e error) {
    if t.MockSelectSchema != nil { return t.MockSelectSchema(c, w) }
    return
}

func CommonTestingTable(schema Schema) *databases.MockTable {
    table := &databases.MockTable{make([][]interface{}), make(map[string]int)}

    i := 0
    for k, _ := range schema {
        table.Schema[k] = i
    }

    table.MockInsertSchema = func(values map[string]interface{})(error) {
        addition := make([]interface{}, len(table.Schema))

        for k, v := range values {
            addition[table.Schema[k]] = v
        }

        table.Database = append(table.Database, addition)

        return nil
    }

    table.MockUpdateSchema = func(to, where map[string]interface{})(error) {
        for _, row := range table.Database {

            applies := true
            for col, val := range where {
                if table.Database[table.Schema[col]] != val {
                    applies = false
                    break
                }
            }

            if applies {
                for col, val := range to {
                    row[table.Schema[col]] = val
                }
            }
        }
        return nil
    }

    table.MockSelectRowSchema = func(cols []string, where databases.Filter, dest interface{})(error) {

        for _, row := range table.Database {

            applies := true
            for col, val := range where {
                if table.Database[table.Schema[col]] != val {
                    applies = false
                    break
                }
            }

            if applies {
                rv := reflect.ValueOf(dest).Elem()
                for _, col := range cols {
                    rv.FieldByName(col).Set(reflect.ValueOf(row[table.Schema[col]]))
                }
            }
        }

        return nil
    }

    return table
}