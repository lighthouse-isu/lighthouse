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

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/databases"
)

var (
    UnknownApplicationError = errors.New("applications: unknown application ID")
    UnknownDeploymentError = errors.New("applications: unknown deployment ID")
    DeploymentMismatchError = errors.New("applications: deployment does not belong to given application")
    NotEnoughDeploymentsError = errors.New("applications: no previous deployment to rollback to")
    StateNotChangedError = errors.New("applications: application already in requested state")
    ImageNotPulledError = errors.New("applications: deployment failed to pull an image")
)

var applications databases.TableInterface
var deployments databases.TableInterface

var appSchema = databases.Schema {
    "Id" : "serial primary key",
    "CurrentDeployment" : "integer",
    "Name" : "text UNIQUE",
    "Active" : "boolean",
    "Instances" : "json",
}

var deploySchema = databases.Schema {
    "Id" : "serial primary key",
    "AppId" : "integer",
    "Command" : "json",
    "User" : "text",
    "Date" : "datetime DEFAULT current_timestamp",
}

type applicationData struct {
    Id int64
    CurrentDeployment int64
    Name string
    Active bool
    Instances []string
}

type deploymentData struct {
    Id int64
    AppId int64
    Command interface{}
    User string
    Date string
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
    var application applicationData
    where := databases.Filter{"Id" : Id}

    err := applications.SelectRow(nil, where, nil, &application)

    if err == databases.NoRowsError {
        err = UnknownApplicationError
    }

    return application, err
}

func GetApplicationByName(Name string) (applicationData, error) {
    var application applicationData
    where := databases.Filter{"Name" : Name}

    err := applications.SelectRow(nil, where, nil, &application)

    if err == databases.NoRowsError {
        err = UnknownApplicationError
    }

    return application, err
}

func Handle(r *mux.Router) {
    // r.HandleFunc("/create", handleCreateApplication).Methods("POST")

    // r.HandleFunc("/list", handleListApplications).Methods("GET")

    // r.HandleFunc("/list/{Id:.*}", handleGetApplicationHistory).Methods("GET")

    // r.HandleFunc("/start/{Id:.*}", handleStartApplication).Methods("POST")

    // r.HandleFunc("/stop/{Id:.*}", handleStopApplication).Methods("POST")

    // r.HandleFunc("/revert/{Id:.*}", handleRevertApplication).Methods("PUT")
}