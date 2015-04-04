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
	"net/http"
	"encoding/json"
	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/handlers/batch"
    "github.com/lighthouse/lighthouse/databases"
)

func addApplication(name string, instances []string) (applicationData, error) {
	values := map[string]interface{} {
        "Name" : name,
        "Instances" : instances,
        "Active" : false,
        "CurrentDeployment" : int64(-1),
    }

    opts := databases.SelectOptions{Top: 1, OrderBy: []string{"Id"}, Desc : true}

    var app applicationData
    err := applications.InsertReturn(values, nil, &opts, &app)

    return app, err
}

func addDeployment(app int64, cmd interface{}, email string) (deploymentData, error) {
	values := map[string]interface{} {
        "AppId" : app,
        "Command" : cmd,
        "User" : email,
    }

    opts := databases.SelectOptions{Top: 1, OrderBy: []string{"Id"}, Desc : true}

    var deploy deploymentData
    err := deployments.InsertReturn(values, nil, &opts, &deploy)

    return deploy, err
}

func removeApplication(app int64) error {
    where := databases.Filter{"Id" : app}
    return applications.Delete(where)
}

func removeDeployment(deploy int64) error {
    where := databases.Filter{"Id" : deploy}
    return deployments.Delete(where)
}

// func createApplication(user *auth.User, name string, cmd interface{}, instances []string, startApp, pullImages bool, w http.ResponseWriter) error {
// 	application, err := addApplication(name, instances)
//     if err != nil {
//     	return err
//     }

//     deployment, err := addDeployment(application.Id, cmd, user.Email)
//     if err != nil {
//     	removeApplication(application.Id)
//         return err
//     }

//     err = doDeployment(application, deployment, startApp, pullImages, w)

//     if err != nil {
//         removeDeployment(deployment.Id)
//         removeApplication(application.Id)
//         return err
//     }

//     auth.SetUserApplicationAuthLevel(user, application.Name, auth.OwnerAuthLevel)

//     return nil
// }

func startApplication(app int64, w http.ResponseWriter) error {
	return setApplicationStateTo(app, true, w)
}

func stopApplication(app int64, w http.ResponseWriter) error {
	return setApplicationStateTo(app, false, w)
}

func doDeployment(app applicationData, deployment deploymentData, startApp, pullImages bool, w http.ResponseWriter) error {
	deploy := batch.NewProcessor(w, app.Instances)

	if pullImages {
		image := struct {
			Image string
		}{}

		jsonCmd, err := json.Marshal(deployment.Command)
		if err == nil {
			err = json.Unmarshal(jsonCmd, &image)
		}
		
		if err != nil {
			return err
		}

		pullTarget := fmt.Sprintf("images/create?fromImage?%s", image.Image)
		err = deploy.Do("POST", nil, pullTarget, nil)

		if err != nil {
			return err
		}
	}

	tmpName := fmt.Sprintf("%s_tmp", app.Name)

	createTarget := fmt.Sprintf("containers/create?name=%s", tmpName)
	err := deploy.Do("POST", deployment.Command, createTarget, nil)

	if err != nil {
		deleteTarget := fmt.Sprintf("containers/%s?force=true", tmpName)
		deploy.Do("DELETE", nil, deleteTarget, nil)
		return err
	}

	deleteTarget := fmt.Sprintf("containers/%s?force=true", app.Name)
	err = deploy.Do("DELETE", nil, deleteTarget, nil)
	if err != nil {
		return err
	}

	renameTarget := fmt.Sprintf("containers/%s/rename?name=%s", tmpName, app.Name)
	err = deploy.Do("POST", nil, renameTarget, nil)

	if err != nil {
		// In a weird case where some containers have the temp name and some have the real
		// name. Have to rollback each case individually.
		
		deleteAppTarget := fmt.Sprintf("containers/%s?force=true", app.Name)
		deploy.Do("DELETE", nil, deleteAppTarget, nil)

		deleteTmpTarget := fmt.Sprintf("containers/%s?force=true", tmpName)
		batch.NewProcessor(w, deploy.Failures()).Do("DELETE", nil, deleteTmpTarget, nil)
		return err
	}

	if app.Active || startApp {
		startTarget := fmt.Sprintf("containers/%s/start", app.Name)
		err = deploy.Do("POST", nil, startTarget, nil)

		if err != nil {
			return err
		}
	}

	to := map[string]interface{} {"CurrentDeployment" : deployment.Id}
	where := databases.Filter{"Id" : app.Id}
	applications.Update(to, where)

	return nil
}

func getApplicationList(user *auth.User) ([]applicationData, error) {
    scanner, err := applications.Select(nil, nil, nil)

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

        if user.CanAccessApplication(app.Name) {
            apps = append(apps, app)
        }
    }
   
    return apps, nil
}

func getApplicationHistory(user *auth.User, app applicationData) ([]map[string]interface{}, error) {
	if !user.CanAccessApplication(app.Name) {
		return []map[string]interface{}{}, nil
	}

	cols := []string{"Id", "User", "Date"}
	where := databases.Filter{"AppId" : app.Id}
    scanner, err := deployments.Select(cols, where, nil)

    if err != nil {
        return nil, err
    }

    deploys := make([]map[string]interface{}, 0)
    var deploy deploymentData

    for scanner.Next() {
        err = scanner.Scan(&deploy)
        if err != nil {
	        return nil, err
	    }

	    deploys = append(deploys, map[string]interface{}{
            "Id" : deploy.Id,
            "Creator" : deploy.User,
            "Date" : deploy.Date,
        })
    }
   
    return deploys, nil
}

func setApplicationStateTo(id int64, state bool, w http.ResponseWriter) error {
	app, err := GetApplicationById(id)
	if err != nil {
		return err
	}

	if app.Active == state {
		return StateNotChangedError
	}

	var target, rollback string

	if state == false {
		target = fmt.Sprintf("containers/%s/stop", app.Name)
		rollback = fmt.Sprintf("containers/%s/start", app.Name)
	} else {
		target = fmt.Sprintf("containers/%s/start", app.Name)
		rollback = fmt.Sprintf("containers/%s/stop", app.Name)
	}

	toggle := batch.NewProcessor(w, app.Instances)

	err = toggle.Do("POST", nil, target, nil)

	if err == nil {
		to := map[string]interface{} {"Active" : state}
		where := databases.Filter{"Id" : id}
		err = applications.Update(to, where)
	}

	if err != nil {
		toggle.Do("POST", nil, rollback, nil)
		return err
	}

	return nil
}

func getRevertDeployment(app int64, target int64) (deploymentData, error) {
	var deployment deploymentData
	var err error

	if target >= 0 {
		where := databases.Filter{"Id" : target}
		err = deployments.SelectRow(nil, where, nil, &deployment)

		if err == databases.NoRowsError {
			err = UnknownDeploymentError
		} else if deployment.AppId != app {
			err = DeploymentMismatchError
			deployment = deploymentData{}
		}

	} else {
		priorCnt := int(-1 * target)
		where := databases.Filter{"AppId" : app}
		opts := &databases.SelectOptions {
			OrderBy : []string{"Id"},
			Top : priorCnt + 1,
			Desc : true,
		}

		scan, err := deployments.Select(nil, where, opts)
		if err != nil {
			return deployment, err
		}

		if !scan.Next() {
			return deployment, UnknownApplicationError
		}

		for i := 0; i < priorCnt; i += 1 {
			if !scan.Next() {
				return deployment, NotEnoughDeploymentsError
			}
		}

		err = scan.Scan(&deployment)
	}

	return deployment, err
}