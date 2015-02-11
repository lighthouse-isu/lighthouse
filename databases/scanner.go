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
	rows *sql.Rows
    table *Table
}

func (this *Scanner) Scan(dest interface{}) error {
    columns, err := this.rows.Columns()

    if err != nil {
        return err
    }

	row := make([]interface{}, len(columns))
    rowPtrs := make([]interface{}, len(columns))

    for i := 0; i < len(row); i++ {
        rowPtrs[i] = &row[i]
    }

    canScan := this.rows.Next()

    if canScan {
	    this.rows.Scan(rowPtrs...)

        rv := reflect.ValueOf(dest).Elem()
        for i, colName := range columns {
            setVal := this.table.convertOutput(row[i], colName)
            if setVal != nil {
                rv.FieldByName(colName).Set(reflect.ValueOf(setVal))
            }
        }
    } else {
        return this.rows.Err()
    }

    return nil
}

func (this *Scanner) Close() error {
    return this.rows.Close()
}
