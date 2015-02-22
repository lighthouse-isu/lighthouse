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
    "fmt"
    "errors"

    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

var containers *databases.Table

var columns = []string {"Id", "AppId", "DockerInstance", "Name"}
var types = []string {
        "serial primary key",
        "integer REFERENCES applications (Id)",
        "text REFERENCES aliases (keyColumn)",
        "text",
    }

type Container struct {
    Id int
    AppId int
    DockerInstance string //TODO: normalize data (add IDs)
    Name string
}

func getDBSingleton() *databases.Table {
    if containers == nil {
        panic("Containers database not initialized")
    }
    return containers
}

func Init() {
    if containers == nil {
        containers = databases.NewSchemaTable(postgres.Connection(), "containers", columns, types)
    }
}

func CreateContainer(AppId int, DockerInstance string, Name string) error {
    var values []string = make([]string, len(columns))
    
    values[0] = "DEFAULT"
    values[1] = fmt.Sprintf("%d", AppId)
    values[2] = DockerInstance
    values[3] = Name

    return getDBSingleton().InsertSchema(columns, values)
}

func GetContainer(Id int, container *Container) (err error) {
    //temporary. need to figure out how best to add to databases package
    query := "SELECT * FROM containers WHERE Id = ($1)"
    var queryParams = make([]string, 1)
    queryParams[0] = fmt.Sprintf("%d", Id)
    row := getDBSingleton().CustomSelect(query, queryParams)

    if row == nil {
        return errors.New("unknown database error")
    }
    
    err = row.Scan(&(container.Id),
                    &(container.AppId),
                    &(container.DockerInstance),
                    &(container.Name))
    return
}
