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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/databases"
	"github.com/lighthouse/lighthouse/handlers/batch"
)

func addApplication(name string, instances []string) (applicationData, error) {
	values := map[string]interface{}{
		"Name":              name,
		"Instances":         instances,
		"CurrentDeployment": int64(-1),
	}

	opts := databases.SelectOptions{Top: 1, OrderBy: []string{"Id"}, Desc: true}

	var app applicationData
	err := applications.InsertReturn(values, nil, &opts, &app)

	app.Instances, _ = convertInstanceList(app.Instances)

	return app, err
}

func addDeployment(app int64, cmd interface{}, email string) (deploymentData, error) {
	values := map[string]interface{}{
		"AppId":   app,
		"Command": cmd,
		"Creator": email,
	}

	opts := databases.SelectOptions{Top: 1, OrderBy: []string{"Id"}, Desc: true}

	var deploy deploymentData
	err := deployments.InsertReturn(values, nil, &opts, &deploy)

	return deploy, err
}

func removeApplication(app int64) error {
	where := databases.Filter{"Id": app}
	return applications.Delete(where)
}

func removeDeployment(deploy int64) error {
	where := databases.Filter{"Id": deploy}
	return deployments.Delete(where)
}

func startApplication(user *auth.User, app int64, w http.ResponseWriter) error {
	return setApplicationStateTo(user, app, true, w)
}

func stopApplication(user *auth.User, app int64, w http.ResponseWriter) error {
	return setApplicationStateTo(user, app, false, w)
}

func doDeployment(user *auth.User, app applicationData, deployment deploymentData, startApp, pullImages bool, w http.ResponseWriter) (error, bool) {
	deploy := batch.NewProcessor(user, w, app.Instances.([]string))

	if pullImages {
		image, ok := deployment.Command["Image"]
		if !ok || image == "" {
			return NotEnoughParametersError, false
		}

		pullTarget := fmt.Sprintf("images/create?fromImage=%s", image)
		err := deploy.Do("Pulling required image", "POST", nil, pullTarget, interpretPullImage)

		if err != nil {
			return nil, false
		}
	}

	tmpName := fmt.Sprintf("%s_tmp", app.Name)

	createTarget := fmt.Sprintf("containers/create?name=%s", tmpName)
	err := deploy.Do("Creating new container", "POST", deployment.Command, createTarget, nil)

	if err != nil {
		batchDeleteContainersByName(deploy, tmpName, true)
		return nil, false
	}

	err = batchDeleteContainersByName(deploy, app.Name, false)
	if err != nil {
		return nil, false
	}

	renameTarget := fmt.Sprintf("containers/%s/rename?name=%s", tmpName, app.Name)
	err = deploy.Do("Setting up new container", "POST", nil, renameTarget, nil)

	if err != nil {
		// In a weird state where some containers have the temp name and some have the real name
		batchDeleteContainersByName(deploy, app.Name, true)
		batchDeleteContainersByName(deploy.FailureProcessor(), tmpName, true)

		return nil, false
	}

	if startApp {
		startTarget := fmt.Sprintf("containers/%s/start", app.Name)
		err = deploy.Do("Starting application", "POST", nil, startTarget, nil)

		if err != nil {
			return nil, false
		}
	}

	to := map[string]interface{}{"CurrentDeployment": deployment.Id}
	where := databases.Filter{"Id": app.Id}
	applications.Update(to, where)

	return nil, true
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

	cols := []string{"Id", "Creator", "Date", "Command"}
	where := databases.Filter{"AppId": app.Id}
	opts := databases.SelectOptions{OrderBy: []string{"Id"}, Desc: true}
	scanner, err := deployments.Select(cols, where, &opts)

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
			"Id":      deploy.Id,
			"Creator": deploy.Creator,
			"Date":    deploy.Date,
			"Image":   deploy.Command["Image"],
			"Command": deploy.Command,
		})
	}

	versionCnt := len(deploys)
	for i, deploy := range deploys {
		deploy["Version"] = versionCnt - i
	}

	return deploys, nil
}

func setApplicationStateTo(user *auth.User, id int64, state bool, w http.ResponseWriter) error {
	app, err := GetApplicationById(id)
	if err != nil {
		return err
	}

	var target, rollback, msg string

	if state == false {
		target = fmt.Sprintf("containers/%s/stop", app.Name)
		rollback = fmt.Sprintf("containers/%s/start", app.Name)
		msg = "Stopping application"
	} else {
		target = fmt.Sprintf("containers/%s/start", app.Name)
		rollback = fmt.Sprintf("containers/%s/stop", app.Name)
		msg = "Starting application"
	}

	w.WriteHeader(200)

	toggle := batch.NewProcessor(user, w, app.Instances.([]string))

	err = toggle.Do(msg, "POST", nil, target, nil)
	if err != nil {
		toggle.Do("Rolling back", "POST", nil, rollback, nil)
		return err
	}

	return nil
}

func getRevertDeployment(app int64, target int64) (deploymentData, error) {
	var deployment deploymentData
	var err error

	if target >= 0 {
		where := databases.Filter{"Id": target}
		err = deployments.SelectRow(nil, where, nil, &deployment)

		if err == databases.NoRowsError {
			err = UnknownDeploymentError
		} else if deployment.AppId != app {
			err = DeploymentMismatchError
			deployment = deploymentData{}
		}

	} else {
		priorCnt := int(-1 * target)

		where := databases.Filter{"AppId": app}
		opts := &databases.SelectOptions{
			OrderBy: []string{"Id"},
			Top:     priorCnt + 1,
			Desc:    true,
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

func batchDeleteContainersByName(proc *batch.Processor, name string, isRollback bool) error {
	var msg string
	if isRollback {
		msg = "Rolling back"
	} else {
		msg = "Deleting old containers"
	}

	deleteTarget := fmt.Sprintf("containers/%s?force=true", name)
	return proc.Do(msg, "DELETE", nil, deleteTarget, interpretDeleteContainer)
}

func convertInstanceList(inter interface{}) ([]string, bool) {
	if inter == nil {
		return []string{}, true
	}

	strList, ok := inter.([]string)
	if ok {
		return strList, true
	}

	list, ok := inter.([]interface{})
	if !ok {
		return nil, false
	}

	ret := make([]string, len(list))

	for i, item := range list {
		str, ok := item.(string)

		if !ok {
			return nil, false
		}

		ret[i] = str
	}

	return ret, true
}

func interpretDeleteContainer(code int, body io.Reader, err error) (batch.Result, error) {
	if err != nil {
		return batch.Result{"Error", err.Error(), 500}, err
	}

	switch {
	case code == 404:
		return batch.Result{"Warning", "Container did not exist", code}, nil

	case 200 <= code && code <= 299:
		return batch.Result{"OK", "", code}, nil

	default:
		return batch.Result{"Error", "", code}, errors.New(fmt.Sprintf("Instance request returned code %d", code))
	}
}

func interpretPullImage(code int, body io.Reader, err error) (batch.Result, error) {
	if err != nil {
		return batch.Result{"Error", err.Error(), 500}, err
	}

	if code == 500 {
		return batch.Result{"Error", "Docker internal error", 500}, errors.New("Image pull returned code 500")
	}

	var errorCheck = struct {
		Error string `json:"error"`
	}{}

	decoder := json.NewDecoder(body)

	for {
		err := decoder.Decode(&errorCheck)

		if err == io.EOF {
			return batch.Result{"OK", "", code}, nil
		} else if err != nil {
			return batch.Result{"Error", err.Error(), 500}, err
		}

		if errorCheck.Error != "" {
			return batch.Result{"Error", errorCheck.Error, 500}, errors.New(errorCheck.Error)
		}
	}
}
