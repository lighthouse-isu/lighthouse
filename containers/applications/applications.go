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
    "fmt"
    "errors"

    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

var applications *databases.Table

var columns = []string {"Id", "Name"}
var types = []string {
    "serial primary key",
    "text",
}

type Application struct {
    Id int
    Name string
}

func getDBSingleton() *databases.Table {
    if applications == nil {
        panic("Applications database not initialized")
    }
    return applications
}

func Init() {
    if applications == nil {
        applications = databases.NewSchemaTable(postgres.Connection(), "applications", columns, types)
    }
}

func CreateApplication(Name string) error {
    var values []string = make([]string, len(columns))
    values[0] = "DEFAULT"
    values[1] = Name

    return getDBSingleton().InsertSchema(columns, values)
}

func getApplication(Id int, application *Application) (err error) {
    //See GetContainer in containers.go for note
    query := "SELECT * FROM applications WHERE Id = ($1)"
    var queryParams = make([]string, 1)
    queryParams[0] = fmt.Sprintf("%d", Id)
    row := getDBSingleton().CustomSelect(query, queryParams)

    if row == nil {
        return errors.New("unknown database error")
    }

    err = row.Scan(&(application.Id), &(application.Name))

    return
}
