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
    "time"
    "errors"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/databases"
)

var (
    UnknownApplicationError = errors.New("applications: unknown application ID")
    UnknownDeploymentError = errors.New("applications: unknown deployment ID")
    DeploymentMismatchError = errors.New("applications: deployment does not belong to given application")
    NotEnoughDeploymentsError = errors.New("applications: no previous deployment to rollback to")
    ImageNotPulledError = errors.New("applications: deployment failed to pull an image")
    NotEnoughParametersError = errors.New("applications: not enough or invalid parameters given")
    ApplicationPermissionError = errors.New("applications: user not permitted to modify application")
)

var applications databases.TableInterface
var deployments databases.TableInterface

var appSchema = databases.Schema {
    "Id" : "serial primary key",
    "CurrentDeployment" : "bigint",
    "Name" : "text UNIQUE",
    "Instances" : "json",
}

var deploySchema = databases.Schema {
    "Id" : "serial primary key",
    "AppId" : "bigint",
    "Command" : "json",
    "Creator" : "text",
    "Date" : "datetime DEFAULT current_timestamp",
}

type applicationData struct {
    Id int64
    CurrentDeployment int64
    Name string
    Instances interface{}
}

type deploymentData struct {
    Id int64
    AppId int64
    Command map[string]interface{}
    Creator string
    Date time.Time
}

func Init(reload bool) {
    if applications == nil {
        applications = databases.NewLockingTable(nil, "applications", appSchema)
    }

    if deployments == nil {
        deployments = databases.NewLockingTable(nil, "deployments", deploySchema)
    }

    if reload {
        applications.Reload()
        deployments.Reload()
    }
}

func GetApplicationById(Id int64) (applicationData, error) {
    var app applicationData
    where := databases.Filter{"Id" : Id}

    err := applications.SelectRow(nil, where, nil, &app)

    if err == databases.NoRowsError {
        err = UnknownApplicationError
    }

    app.Instances, _ = convertInstanceList(app.Instances)

    return app, err
}

func GetApplicationByName(Name string) (applicationData, error) {
    var app applicationData
    where := databases.Filter{"Name" : Name}

    err := applications.SelectRow(nil, where, nil, &app)

    if err == databases.NoRowsError {
        err = UnknownApplicationError
    }

    app.Instances, _ = convertInstanceList(app.Instances)

    return app, err
}

func Handle(r *mux.Router) {
    r.HandleFunc("/create", handleCreateApplication).Methods("POST")

    r.HandleFunc("/list", handleListApplications).Methods("GET")

    r.HandleFunc("/list/{Id:.*}", handleGetApplicationHistory).Methods("GET")

    r.HandleFunc("/start/{Id:.*}", handleStartApplication).Methods("POST")

    r.HandleFunc("/stop/{Id:.*}", handleStopApplication).Methods("POST")

    r.HandleFunc("/revert/{Id:.*}", handleRevertApplication).Methods("PUT")

    r.HandleFunc("/update/{Id:.*}", handleUpdateApplication).Methods("PUT")
}