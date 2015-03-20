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
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

var containers databases.TableInterface

var schema = databases.Schema {
    "Id" : "serial primary key",
    "AppId" : "integer REFERENCES applications (Id)",
    "DockerInstance" : "text",
    "Name" : "text",
}

type Container struct {
    Id int64
    AppId int64
    DockerInstance string //TODO: normalize data (add IDs)
    Name string
}

func getDBSingleton() databases.TableInterface {
    if containers == nil {
        panic("Containers database not initialized")
    }
    return containers
}

func Init() {
    if containers == nil {
        containers = databases.NewSchemaTable(postgres.Connection(), "containers", schema)
    }
}

func CreateContainer(AppId int64, DockerInstance string, Name string) (int64, error) {
    values := make(map[string]interface{}, len(schema)-1)

    values["AppId"] = AppId
    values["DockerInstance"] = DockerInstance
    values["Name"] = Name

    containerId, err := getDBSingleton().InsertSchema(values, "Id")
    if err != nil {
        return -1, err
    }

    return containerId.(int64), err
}

func DeleteContainer(Id int64) (err error) {
    where := databases.Filter{"Id" : Id}
    return getDBSingleton().DeleteRowsSchema(where)
}

func GetContainerById(Id int64, container *Container) (err error) {
    where := databases.Filter{"Id" : Id}
    var columns []string

    for k, _ := range schema {
        columns = append(columns, k)
    }

    err = getDBSingleton().SelectRowSchema(columns, where, container)

    return
}
