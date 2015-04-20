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
    "database/sql"
)

type Scanner struct {
	sql.Rows
    table *Table
    columns []string
}

func (this *Scanner) Scan(dest interface{}) error {
	row := make([]interface{}, len(this.columns))
    rowPtrs := make([]interface{}, len(this.columns))

    for i := 0; i < len(row); i++ {
        rowPtrs[i] = &row[i]
    }

	this.Rows.Scan(rowPtrs...)

    rv := reflect.ValueOf(dest).Elem()
    for i, colName := range this.columns {
        setVal := this.table.compiler.ConvertOutput(row[i], colName)
        if setVal != nil {
            rv.FieldByName(colName).Set(reflect.ValueOf(setVal))
        }
    }

    return nil
}
