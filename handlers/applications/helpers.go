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
	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/handlers"
    "github.com/lighthouse/lighthouse/databases"
)

func createApplication(name string, instances []string) (applicationData, error) {
	values := map[string]interface{} {
        "Name" : name,
        "Instances" : instances,
        "Active" : false
        "CurrentDeployment" : -1,
    }

    opts := databases.SelectOptions{Top: 1, OrderBy: []string{"Id"}, Desc : true}

    var app applicationData
    err := applications.InsertReturn(values, []string{"Id"}, &opts, &app)

    return app, err
}

func createDeployment(app int64, cmd interface{}, email string) (deploymentData, error) {
	values := map[string]interface{} {
        "AppId" : app,
        "Command" : cmd,
        "User" : email,
    }

    opts := databases.SelectOptions{Top: 1, OrderBy: []string{"Id"}, Desc : true}

    var deploy deploymentData
    err := deployments.InsertReturn(values, []string{"Id"}, &opts, &deploy)

    return deploy, err
}

func removeApplication(app int64) error {
    where := database.Filter{"Id" : app}
    return applications.Delete(where)
}

func removeDeployment(deploy int64) error {
    where := database.Filter{"Id" : deploy}
    return deployments.Delete(where)
}

func createApplication(w http.ResponseWriter, user *auth.User, name string, cmd interface{}, instances []string, start, pullImages bool) error {
	application, err := createApplication(name, instances)
    if err != nil {
    	return err
    }

    deployment, err := createDeployment(application.Id, name, cmd, user.Email)
    if err != nil {
    	removeApplication(application.Id)
        return err
    }

    err = doDeployment(application, deployment, pullImages)
    if err == nil && start {
    	err = startApplication(application, w)
    }

    if err != nil {
        removeDeployment(deployment.Id)
        removeApplication(application.Id)
        return err
    }

    auth.SetUserApplicationAuthLevel(user, application.Id, auth.OwnerAuthLevel)

    return nil
}

func startApplication(app int64, w http.ResponseWriter) error {
	application, err := getApplicationById(app)
	if err != nil {
		return err
	}

	if application.Active {
		return StateNotChangedError
	}

	if !toggleApplicationState(app, w) {
		return StateNotChangedError
	}

	return nil
}

func stopApplication(app int64, w http.ResponseWriter) error {
	application, err := getApplicationById(app)
	if err != nil {
		return err
	}

	if !application.Active {
		return StateNotChangedError
	}

	if !toggleApplicationState(app, w) {
		return StateNotChangedError
	}

	return nil
}

func doDeployment(app applicationData, deploy deploymentData, start, pullImages bool) error {
	if forcePull {
		image := struct {
			Image string
		}{}

		jsonCmd, err := json.Marshal(deploy.Command)
		if err == nil {
			json.Unmarshal(jsonCmd, &image)
		}
		
		if err != nil {
			return err
		}

		completed := handlers.NewBatchProcess(w, app.Instances).
			AddStep("POST", nil, fmt.Sprintf("images/create?fromImage?%s", image.Image)).
			Run()

		if len(completed) != len(app.Instances) {
			return ImageNotPulledError
		}
	}

	batch := handlers.NewBatchProcess(w, app.Instances).
		AddStep("DELETE", nil, fmt.Sprintf("containers/%s?force=true", app.Name)).
		AddStep("POST", deploy.Command, "containers/create")

	if app.Active {
		batch.AddStep("POST", nil, fmt.Sprintf("containers/%s/start", app.Name))
	}

	completed := batch.Run()

	if len(completed) != len(app.Instances) {
		return DeploymentFailedError
	}

	return nil
}

func getApplicationList(user *auth.User) ([]applicationData, error) {
	cols := []string{"Id", "CurrentDeployment", "Name", "Active"}
    scanner, err := applications.Select(cols, nil, nil)

    if err != nil {
        return nil, err
    }

    apps := make([]applicationData, 0)
    var app applicationData

    for scanner.Next() {
        err = scanner.Scan(&app)
        if err != nil {
	        return nil, err
	    }

        if user.CanAccessApplication(app.Id) {
            apps = append(apps, app)
        }
    }
   
    return apps, nil
}

func toggleApplicationState(app applicationData, w http.ResponseWriter) boolean {
	var targetState, rollbackState string

	if app.Active {
		targetState, rollbackState := "stop", "start"
	} else {
		targetState, rollbackState := "start", "stop"
	}

	completed := handlers.NewBatchProcess(w, app.Instances).
		AddStep("POST", nil, fmt.Sprintf("containers/%s/%s", app.Name, targetState)).
		Run()

	if len(completed) != len(application.Instances) {
		handlers.NewBatchProcess(w, completed).
			AddStep("POST", nil, fmt.Sprintf("containers/%s/%s", app.Name, rollbackState)).
			Run()

		return false
	}

	to := map[string]interface{} {"Active" : !app.Active}
	where := databases.Filter{"Id" : app.Id}
	applications.Update(to, where)

	return true
}

func getRevertDeployment(app int64, target int64) (deploymentData, error) {
	var deployment deploymentData
	var err error

	if target >= 0 {
		where := databases.Filter{"Id" : target, "AppId" : app}
		err = deployments.SelectRow(nil, where, nil, &deployment)
	} else {
		priorCnt := int(-1 * target)
		where := databases.Filter{"AppId" : app}
			opts := &databases.SelectOptions {
			OrderBy : []string{"Date"},
			Top : priorCnt + 1,
			Desc : true,
		}

		scan, err := deployments.Select(nil, where, opts)
		if err != nil {
			return nil, err
		}

		if !scan.Next() {
			return nil, UnknownApplicationError
		}

		for i := 0; i < priorCnt; i += 1 {
			if !scan.Next() {
				return nil, NotEnoughDeploymentsError
			}
		}

		err = scan.Scan(&deployment)
	}

	if err != nil {
		return nil, err
	}

	return deployment, nil
}