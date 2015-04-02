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

package applications

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/lighthouse/lighthouse/databases"
)

func Test_CreateApplication(t *testing.T) {
    table := databases.CommonTestingTable(schema)
    SetupCustomTestingTable(table)
    defer TeardownTestingTable()

    CreateApplication("Test")

    assert.Equal(t, 1, len(table.Database),
        "Database should have new element after CreateApplication")

    assert.Equal(t, 0, table.Database[0][table.Schema["Id"]],
        "CreateApplication should auto-increment Id")

    assert.Equal(t, "Test", table.Database[0][table.Schema["Name"]],
        "CreateApplication should set Name")
}

func Test_GetApplicationName(t *testing.T) {
    table := databases.CommonTestingTable(schema)
    SetupCustomTestingTable(table)
    defer TeardownTestingTable()

    appA, _ := CreateApplication("appA")
    appB, _ := CreateApplication("appB")

    nameA, _ := GetApplicationName(appA)
    nameB, _ := GetApplicationName(appB)

    assert.Equal(t, "appA", nameA,
        "GetApplicationName should return the correct name.")

    assert.Equal(t, "appB", nameB,
        "GetApplicationName should return the correct name.")
}

func Test_GetApplicationId(t *testing.T) {
    table := databases.CommonTestingTable(schema)
    SetupCustomTestingTable(table)
    defer TeardownTestingTable()

    appA, _ := CreateApplication("appA")
    appB, _ := CreateApplication("appB")

    idA, _ := GetApplicationId("appA")
    idB, _ := GetApplicationId("appB")
    assert.Equal(t, appA, idA,
        "GetApplicationId should return the correct ID.")

    assert.Equal(t, appB, idB,
        "GetApplicationId should return the correct ID.")
}

func Test_Init(t *testing.T) {
    SetupTestingTable()
    defer TeardownTestingTable()

    // Basically just making sure this doesn't panic...
    Init(true)
}