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
    "net/http"

    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/handlers"
    "github.com/lighthouse/lighthouse/handlers/containers/applications"
)

func ContainerCreateHandler(info handlers.HandlerInfo, rollback bool) *handlers.HandlerError {
    if rollback {
        containerDelete(info)
        return nil
    } else {
        return containerCreate(info)
    }

}

func containerCreate(info handlers.HandlerInfo) *handlers.HandlerError {
    var errInfo handlers.HandlerError

    //get name
    name := info.Request.FormValue("name")
    if name == "" {
        errInfo.StatusCode = http.StatusBadRequest
        errInfo.Cause = "control"
        errInfo.Message = "Containers must be created with a name"
        return &errInfo
    }

    //get or create application
    appId, err := applications.GetApplicationId(name)
    if err == databases.NoRowsError {
        appId, err = applications.CreateApplication(name)
    }

    if err != nil {
        errInfo.StatusCode = http.StatusInternalServerError
        errInfo.Cause = "control"
        errInfo.Message = "Failed to read application from database"
        return &errInfo
    }

    //create container
    containerId, err := CreateContainer(appId, info.Host, name)

    if err != nil {
        errInfo.StatusCode = http.StatusInternalServerError
        errInfo.Cause = "control"
        errInfo.Message = "Failed to insert new container into database."
        return &errInfo
    }

    info.HandlerData["ContainerCreate"] = containerId
    return nil
}

func containerDelete(info handlers.HandlerInfo) {
    containerId := info.HandlerData["ContainerCreate"]
    DeleteContainer(containerId.(int64))
}
