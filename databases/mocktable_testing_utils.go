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
    "reflect"
    "strings"
)

/*
    The purpose of MockTable is to be able to create custom mocks for datatbases
    that can be directly used in a database.Database
*/

type MockTable struct {
    Database [][]interface{}
    Schema map[string]int
    lastUpdateRow int64

    MockInsert     func(map[string]interface{}, string)(interface{}, error)
    MockDelete     func(Filter)(error)
    MockUpdate     func(map[string]interface{}, Filter)(error)
    MockSelectRow  func([]string, Filter, interface{})(error)
    MockSelect     func([]string, Filter, SelectOptions)(ScannerInterface, error)

    MockReload           func()
}

func (t *MockTable) Insert(v map[string]interface{}, returnCol string)(i interface{}, e error) {
    if t.MockInsert != nil { return t.MockInsert(v, returnCol) }
    return
}

func (t *MockTable) Delete(w Filter)(e error) {
    if t.MockDelete != nil { return t.MockDelete(w) }
    return
}

func (t *MockTable) Update(to map[string]interface{}, w Filter)(e error) {
    if t.MockUpdate != nil { return t.MockUpdate(to, w) }
    return
}

func (t *MockTable) SelectRow(c []string, w Filter, d interface{})(e error) {
    if t.MockSelectRow != nil { return t.MockSelectRow(c, w, d) }
    return
}

func (t *MockTable) Select(c []string, w Filter, opts SelectOptions)(s ScannerInterface, e error) {
    if t.MockSelect != nil { return t.MockSelect(c, w, opts) }
    return
}

func (t *MockTable) Reload() {
    if t.MockReload != nil { t.MockReload() }
    return
}

func CommonTestingTable(schema Schema) *MockTable {
    table := &MockTable{Database: make([][]interface{}, 0), Schema: make(map[string]int), lastUpdateRow: 0}

    uniqueCols := []int{}

    i := 0
    for k, t := range schema {
        table.Schema[k] = i

        if strings.Contains(strings.ToLower(t), "unique") {
            uniqueCols = append(uniqueCols, i)
        }

        i += 1
    }

    table.MockInsert = func(values map[string]interface{}, returnCol string)(interface{}, error) {
        addition := make([]interface{}, len(table.Schema))

        for k, orig := range values {
            addition[table.Schema[k]] = orig
        }

        //because "DEFAULT" doesn't work with pq, need to add left-out columns
        for k, v := range table.Schema {
            if _, ok := values[k]; !ok {
                addition[v] = table.lastUpdateRow
            }
        }

        for _, row := range table.Database {
            for _, col := range uniqueCols {
                if reflect.DeepEqual(row[col], addition[col]) {
                    return -1, DuplicateKeyError
                }
            }
        }

        table.lastUpdateRow++
        table.Database = append(table.Database, addition)

        if returnCol != "" {
            ret := addition[table.Schema[returnCol]]
            return ret, nil
        }

        return nil, nil
    }

    table.MockDelete = func(where Filter)(error) {
        updated := false
        var toDelete []int

        i := 0
        for _, row := range table.Database {
            applies := true
            for col, val := range where {
                if row[table.Schema[col]] != val {
                    applies = false
                    break
                }
            }

            if applies {
                toDelete = append(toDelete, i)
                updated = true
            }

            i = i + 1
        }

        //cut the appropriate rows from the database
        for _, rowId := range toDelete {
            copy(table.Database[rowId:], table.Database[rowId+1:])
            for j, end := len(table.Database) - 1, len(table.Database); j < end; j++ {
                table.Database[j] = nil
            }
            table.Database = table.Database[:len(table.Database) - 1]
        }

        if !updated {
            return NoUpdateError
        }

        return nil
    }

    table.MockUpdate = func(to map[string]interface{}, where Filter)(error) {
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
            return NoUpdateError
        }

        return nil
    }

    table.MockSelectRow = func(cols []string, where Filter, dest interface{})(error) {

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

        return NoRowsError
    }

    table.MockSelect = func(cols []string, where Filter, opts SelectOptions)(ScannerInterface, error) {

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
