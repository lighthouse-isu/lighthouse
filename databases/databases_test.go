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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DATA-DOG/go-sqlmock"
)

var testSchema Schema = Schema{
	"Name":  "text UNIQUE PRIMARY KEY",
	"Age":   "integer",
	"Phone": "text",
}

type testObject struct {
	Name  string
	Age   int
	Phone string
}

func Test_SetAndGetDefaultConnection(t *testing.T) {
	db := testDB()

	SetDefaultConnection(db)
	assert.Equal(t, db, DefaultConnection())
}

func Test_DefaultSelectOptions(t *testing.T) {
	opts := DefaultSelectOptions()

	key := &SelectOptions{
		Distinct: false,
		Top:      0,
		OrderBy:  nil,
		Desc:     false,
	}

	assert.Equal(t, key, opts)
}

func Test_NewTableDefault(t *testing.T) {
	db := testDB()
	SetDefaultConnection(db)

	var inter interface{}
	inter = NewTable(nil, "test_table", testSchema)
	table := inter.(*Table)

	assert.Equal(t, db, table.db)
}

func Test_NewTable_Panic(t *testing.T) {
	defer func() { recover() }()

	db := testDB()
	NewTable(db, "test_table", nil)

	t.Errorf("Nil schema in NewTable should panic")
}

func Test_NewLockingTableDefault(t *testing.T) {
	db := testDB()
	SetDefaultConnection(db)

	var inter interface{}
	inter = NewLockingTable(nil, "test_table", testSchema)
	table := inter.(*Table)

	assert.Equal(t, db, table.db)
}

func Test_NewLockingTable_Panic(t *testing.T) {
	defer func() { recover() }()

	db := testDB()
	NewLockingTable(db, "test_table", nil)

	t.Errorf("Nil schema in NewTable should panic")
}

func Test_Reload(t *testing.T) {
	db := testDB()

	sqlmock.ExpectExec(`DROP`)
	sqlmock.ExpectExec(`CREATE`)

	NewTable(db, "test_table", testSchema).Reload()

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_InsertReturn(t *testing.T) {
	db := testDB()
	table := NewLockingTable(db, "test_table", testSchema)

	newData := map[string]interface{}{
		"Name": "John Doe",
		"Age":  42,
	}

	sqlmock.ExpectExec(`INSERT`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	sqlmock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{"Age"}).
		AddRow(42))

	var res testObject

	cols := []string{"Age"}
	err := table.InsertReturn(newData, cols, nil, &res)

	assert.Nil(t, err)
	assert.Equal(t, 42, res.Age)

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_InsertReturn_NoInsert(t *testing.T) {
	db := testDB()
	table := NewLockingTable(db, "test_table", testSchema)

	newData := map[string]interface{}{
		"Name": "John Doe",
		"Age":  42,
	}

	sqlmock.ExpectExec(`INSERT`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	var res testObject

	cols := []string{"Age"}
	err := table.InsertReturn(newData, cols, nil, &res)

	assert.Equal(t, NoUpdateError, err)
}

func Test_InsertReturn_Panic(t *testing.T) {
	defer func() { recover() }()

	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	newData := map[string]interface{}{
		"Name": "John Doe",
		"Age":  42,
	}

	var res testObject

	cols := []string{"Age"}
	table.InsertReturn(newData, cols, nil, &res)

	t.Fail()
}

func Test_Insert(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	newData := map[string]interface{}{
		"Name": "John Doe",
		"Age":  42,
	}

	sqlmock.ExpectExec(`INSERT`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := table.Insert(newData)

	assert.Nil(t, err)

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_Insert_NoInsert(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	newData := map[string]interface{}{
		"Name": "John Doe",
		"Age":  42,
	}

	sqlmock.ExpectExec(`INSERT`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := table.Insert(newData)

	assert.Equal(t, NoUpdateError, err)
}

func Test_Insert_Locking(t *testing.T) {
	db := testDB()
	table := NewLockingTable(db, "test_table", testSchema)

	newData := map[string]interface{}{
		"Name": "John Doe",
		"Age":  42,
	}

	sqlmock.ExpectExec(`INSERT`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := table.Insert(newData)

	assert.Nil(t, err)

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_Update(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	to := map[string]interface{}{
		"Name": "Jane Doe",
		"Age":  42,
	}

	where := map[string]interface{}{
		"Age":   41,
		"Phone": "123-456-7890",
	}

	sqlmock.ExpectExec(`UPDATE`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := table.Update(to, where)
	assert.Nil(t, err)

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_Update_NoUpdate(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	to := map[string]interface{}{
		"Name": "Jane Doe",
		"Age":  42,
	}

	where := map[string]interface{}{
		"Age":   41,
		"Phone": "123-456-7890",
	}

	sqlmock.ExpectExec(`UPDATE`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := table.Update(to, where)
	assert.Equal(t, NoUpdateError, err)
}

func Test_SelectRow(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	columns := []string{"Phone", "Name"}
	filter := Filter{
		"Age": 1,
	}

	sqlmock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("123-456-7890", "Sam"))

	key := testObject{"Sam", 0, "123-456-7890"}

	var res testObject
	err := table.SelectRow(columns, filter, nil, &res)

	assert.Equal(t, key, res)
	assert.Nil(t, err)

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_SelectRow_NoRows(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	columns := []string{"Phone", "Name"}
	filter := Filter{
		"Age": 1,
	}

	sqlmock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows(columns))

	var res testObject
	err := table.SelectRow(columns, filter, nil, &res)

	assert.Equal(t, NoRowsError, err)
}

func Test_Select(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	columns := []string{"Phone", "Name"}
	filter := Filter{
		"Age": 1,
	}

	key := []testObject{
		testObject{"Sam", 0, "123-456-7890"},
		testObject{"Sue", 0, "314-151-9285"},
		testObject{"Bob", 0, "319-256-7380"},
	}

	sqlmock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows(columns).
		AddRow(key[0].Phone, key[0].Name).
		AddRow(key[1].Phone, key[1].Name).
		AddRow(key[2].Phone, key[2].Name))

	var res testObject
	scan, err := table.Select(columns, filter, nil)

	assert.Nil(t, err)

	for i := 0; i < 3 && scan.Next(); i += 1 {
		scan.Scan(&res)
		assert.Equal(t, key[i], res)
	}

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_Select_Star(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", Schema{"Name": "text"})

	sqlmock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{"Name"}))

	_, err := table.Select(nil, nil, nil)

	assert.Nil(t, err)

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_SelectRow_Star(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", Schema{"Name": "text"})

	var data testObject

	sqlmock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{"Name"}).AddRow("Sam"))

	err := table.SelectRow(nil, nil, nil, &data)

	assert.Nil(t, err)

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_Delete(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	where := map[string]interface{}{
		"Age":   41,
		"Phone": "123-456-7890",
	}

	sqlmock.ExpectExec(`DELETE`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := table.Delete(where)
	assert.Nil(t, err)

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_Delete_Nil(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	sqlmock.ExpectExec(`DELETE`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := table.Delete(nil)
	assert.Nil(t, err)

	if err := db.Close(); err != nil {
		t.Errorf(err.Error())
	}
}

func Test_Delete_NoUpdate(t *testing.T) {
	db := testDB()
	table := NewTable(db, "test_table", testSchema)

	sqlmock.ExpectExec(`DELETE`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := table.Delete(nil)

	assert.Equal(t, NoUpdateError, err)
}
