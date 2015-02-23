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
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

var applications databases.TableInterface

var schema = databases.Schema {
    "Id" : "serial primary key",
    "Name" : "text",
}

type applicationData struct {
    Id int
    Name string
}

func getDBSingleton() databases.TableInterface {
    if applications == nil {
        panic("Applications database not initialized")
    }
    return applications
}

func Init() {
    if applications == nil {
        applications = databases.NewSchemaTable(postgres.Connection(), "applications", schema)
    }
}

func CreateApplication(Name string) (int, error) {
    values := make(map[string]interface{}, len(schema))

    values["Id"] = "DEFAULT"
    values["Name"] = Name

    appId, err := getDBSingleton().InsertSchema(values)
    if err != nil {
        return -1, err
    }

    return appId, err
}

func GetApplicationName(Id int) (string, error) {
    var application applicationData
    where := databases.Filter{"Id" : Id}
    var columns []string

    for k, _ := range schema {
        columns = append(columns, k)
    }

    err := getDBSingleton().SelectRowSchema(columns, where, &application)

    if err != nil {
        return "", err
    }

    return application.Name, err
}

func GetApplicationId(Name string) (int, error) {
    var application applicationData
    where := databases.Filter{"Name" : Name}
    var columns []string

    for k, _ := range schema {
        columns = append(columns, k)
    }

    err := getDBSingleton().SelectRowSchema(columns, where, &application)

    if err != nil {
        return -1, err
    }

    return application.Id, err
}
