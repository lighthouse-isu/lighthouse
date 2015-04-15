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
	"net/http"
	"testing"

	"github.com/lighthouse/lighthouse/databases"
	"github.com/lighthouse/lighthouse/handlers"
	"github.com/lighthouse/lighthouse/handlers/containers/applications"
	"github.com/stretchr/testify/assert"
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

	DeleteContainer(containerId)

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

func Test_containerCreate_Passed(t *testing.T) {
	SetupTestingTable()
	applications.SetupTestingTable()
	defer TeardownTestingTable()
	defer applications.TeardownTestingTable()

	var info handlers.HandlerInfo
	info.Host = "HOST_PASSED"
	r, _ := http.NewRequest("POST", "/containers/create?name=NAME_PASSED", nil)
	info.Request = r
	info.HandlerData = make(map[string]interface{})

	ret := containerCreate(info)

	assert.Nil(t, ret)
	assert.Equal(t, 0, info.HandlerData["ContainerCreate"])

	var container Container
	err := GetContainerById(info.HandlerData["ContainerCreate"].(int64), &container)
	assert.Nil(t, err)
	assert.Equal(t, 0, container.Id)
	assert.Equal(t, 0, container.AppId)
	assert.Equal(t, "HOST_PASSED", container.DockerInstance)
	assert.Equal(t, "NAME_PASSED", container.Name)

	name, err := applications.GetApplicationName(container.AppId)
	assert.Nil(t, err)
	assert.Equal(t, "NAME_PASSED", name)
}

func Test_containerCreate_NoName(t *testing.T) {
	var info handlers.HandlerInfo
	info.Host = "HOST_PASSED"
	r, _ := http.NewRequest("POST", "/containers/create", nil)
	info.Request = r
	info.HandlerData = make(map[string]interface{})

	ret := containerCreate(info)
	assert.NotNil(t, ret)
	assert.Equal(t, http.StatusBadRequest, ret.StatusCode)
	assert.Equal(t, "control", ret.Cause)
	assert.Equal(t, "Containers must be created with a name", ret.Message)
}

func Test_containerDelete(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	var container Container
	var info handlers.HandlerInfo
	info.HandlerData = make(map[string]interface{})

	containerId, err := CreateContainer(0, "HOST", "NAME")
	assert.Nil(t, err)

	info.HandlerData["ContainerCreate"] = containerId

	containerDelete(info)

	err = GetContainerById(containerId, &container)
	assert.Equal(t, databases.NoRowsError, err)
}

func Test_containerHandler(t *testing.T) {
	SetupTestingTable()
	applications.SetupTestingTable()
	defer TeardownTestingTable()
	defer applications.TeardownTestingTable()

	var info handlers.HandlerInfo
	info.Host = "HOST_PASSED"
	r, _ := http.NewRequest("POST", "/containers/create?name=NAME_PASSED", nil)
	info.Request = r
	info.HandlerData = make(map[string]interface{})

	handlerErr := ContainerCreateHandler(info, false)
	assert.Nil(t, handlerErr)

	var container Container
	err := GetContainerById(0, &container)
	assert.Nil(t, err)

	handlerErr = ContainerCreateHandler(info, true)
	assert.Nil(t, handlerErr)

	err = GetContainerById(0, &container)
	assert.Equal(t, databases.NoRowsError, err)
}

func Test_Init(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	// Basically just making sure this doesn't panic...
	Init(true)
}
