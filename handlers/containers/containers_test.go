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

package containers

import (
    "testing"
    "fmt"
    "github.com/stretchr/testify/assert"
    "github.com/lighthouse/lighthouse/databases"
)

func Test_CreateContainer(t *testing.T) {
    table := databases.CommonTestingTable(schema)
    SetupCustomTestingTable(table)
    defer TeardownTestingTable()

    CreateContainer(1, "localhost", "Test")

    assert.Equal(t, 1, len(table.Database),
        "Database should have new element after CreateContainer")

    assert.Equal(t, 0, table.Database[0][table.Schema["Id"]],
        "CreateContainer should auto-increment Id")

    assert.Equal(t, 1, table.Database[0][table.Schema["AppId"]],
        "CreateContainer should set AppId")

    assert.Equal(t, "localhost", table.Database[0][table.Schema["DockerInstance"]],
        "CreateContainer should set DockerInstance")

    assert.Equal(t, "Test", table.Database[0][table.Schema["Name"]],
        "CreateContainer should set Name")
}

func Test_DeleteContainer(t *testing.T) {
    table := databases.CommonTestingTable(schema)
    SetupCustomTestingTable(table)
    defer TeardownTestingTable()

    containerId, _ := CreateContainer(1, "localhost", "Test")
    CreateContainer(1, "not localhost", "Test")

    err := DeleteContainer(containerId)
    if err != nil {
        fmt.Println(err)
    }

    assert.Equal(t, 1, len(table.Database),
        "Database should have one element after delete")

    assert.Equal(t, "not localhost", table.Database[0][table.Schema["DockerInstance"]],
        "DeleteContainer should delete the correct container.")
}

func Test_GetContainerById(t *testing.T) {
    table := databases.CommonTestingTable(schema)
    SetupCustomTestingTable(table)
    defer TeardownTestingTable()

    containerId, _ := CreateContainer(1, "localhost", "Test")
    CreateContainer(1, "not localhost", "Test")

    var container Container
    GetContainerById(containerId, &container)

    assert.Equal(t, 0, container.Id,
        "Container ID should match ID in database.")

    assert.Equal(t, 1, container.AppId,
        "Container application should match application in database.")

    assert.Equal(t, "localhost", container.DockerInstance,
        "Container Docker instance should match instance in database.")

    assert.Equal(t, "Test", container.Name,
        "Container name should match name in database.")
}
