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
    "errors"
    "github.com/lighthouse/lighthouse/databases"
)

var (
    UnknownApplicationError := errors.New("applications: unknown application ID")
    NotEnoughDeploymentsError := errors.New("applications: no previous deployment to rollback to")
)

var applications databases.TableInterface
var deployments databases.TableInterface

var appSchema = databases.Schema {
    "Id" : "serial primary key",
    "Name" : "text",
}

var deploySchema = databases.Schema {
    "AppId" : "integer",
    "Command" : "json",
    "Date" : "datetime DEFAULT current_timestamp",
}

type applicationData struct {
    Id int64
    Name string
}

type deployData struct {
    AppId int64
    Command interface{}
    Date string
}

func Init(reload bool) {
    if applications == nil {
        applications = databases.NewLockingTable(nil, "applications", appSchema)
    }

    if applications == nil {
        deployments = databases.NewLockingTable(nil, "deployments", deploySchema)
    }

    if reload {
        applications.Reload()
        deployments.Reload()
    }
}

func CreateApplication(name string, cmd interface{) (int64, error) {
    values := map[string]interface{} {
        "Name" : name,
    }

    cols := []string{"Id"}
    opts := databases.SelectOptions{Top: 1, OrderBy: []string{"Id"}, Desc : true}

    var app applicationData

    err := applications.InsertReturn(values, cols, &opts, &app)
    if err != nil {
        return -1, err
    }

    err = Deploy(app.Id, cmd)
    if err != nil {
        deleteApplication(app.Id)
        return -1, err
    }

    return app.Id, err
}

func GetApplicationName(Id int64) (string, error) {
    var application applicationData
    where := databases.Filter{"Id" : Id}

    err := applications.SelectRow(nil, where, nil, &application)

    if err != nil {
        return "", err
    }

    return application.Name, err
}

func GetApplicationId(Name string) (int64, error) {
    var application applicationData
    where := databases.Filter{"Name" : Name}

    err := applications.SelectRow(nil, where, nil, &application)

    if err != nil {
        return -1, err
    }

    return application.Id, err
}
