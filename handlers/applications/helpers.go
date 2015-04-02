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

func createApplication(user *auth.User, name string, cmd interface{}, instances []string) error {
	application, err := createApplication(name, instances)
    if err != nil {
    	return err
    }

    deployment, err := createDeployment(application.Id, name, cmd, user.Email)
    if err != nil {
    	removeApplication(application.Id)
        return err
    }

    err = doDeployment(application, deployment)
    if err != nil {
        removeDeployment(deployment.Id)
        removeApplication(application.Id)
        return err
    }

    auth.SetUserApplicationAuthLevel(user, application.Id, auth.OwnerAuthLevel)

    // TODO
    // - Check query params to start or force pull

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

func doDeployment(app applicationData, deploy deploymentData, pullImages bool) error {

}

func toggleApplicationState(app applicationData, w http.ResponseWriter) boolean {
	var targetState, rollbackState string

	if app.Active {
		targetState, rollbackState := "stop", "start"
	} else {
		targetState, rollbackState := "start", "stop"
	}

	toRun := make([]string, len(application.Instances))
	for i, inst := range application.Instances {
		toRun[i] = fmt.Sprintf("%s/containers/%s/%s", 
			inst, application.Name, targetState)
	}

	completed := runBatch(toRun, changeContainerState)

	if len(completed) != len(application.Instances) {
		rollback := make([]string, len(completed))
		for i, dest := range completed {
			prefix := dest[:strings.LastIndex(dest, "/")]
			rollback[i] = fmt.Sprintf("%s/%s", prefix, rollbackState)
		}

		runBatch(rollback, changeContainerState)

		return false
	}

	to := map[string]interface{} {"Active" : !app.Active}
	where := databases.Filter{"Id" : app.Id}
	applications.Update(to, where)

	return true
}

func changeContainerState(dest string) (*http.Response, error) {
	req, err := http.NewRequest("POST", dest, nil)

	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

func getRollbackCommand(app int64, target int64) (interface{}, error) {
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

		scan, err := deployments.Select(cols, where, opts)
		if err != nil {
			return nil, err
		}

		// No deployments known, application doesn't exist
		if !scan.Next() {
			return nil, UnknownApplicationError
		}

		// Skip the previous deployments (and prep the target deployment)
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

	return deployment.Command, nil
}

func runBatch(items []string, f func(string)(*http.Response, error), w http.ResponseWriter) []string {
	var completed []string
	total := len(items)

	wg := sync.WaitGroup{}
	wg.Add(total)
	for i, item := range items {
		go func(item string, number int) {
			resp, err := f(item)
			update, ok := getStreamStatus(resp, err)

			update.Progress = number
			update.Total = total

			body, _ := json.Marshal(update)
			w.Write(body)

			if ok {
				completed = append(completed, item)
			}
		}(item, i)
	}
	wg.Wait()

	return completed
}

func getStreamStatus(resp *http.Response, err error) (streamUpdate, bool) {
	if err != nil {
		return streamUpdate{Status : "Error", Message : err.Error()}, false
	}

	switch resp.Status {
	case 201, 204:
		return streamUpdate{Status : "OK", Message : ""}, true
	case 304:
		return streamUpdate{Status : "Warning", Message : "Container was already in requested state"}, true
	case 404:
		return streamUpdate{Status : "Error", Message : "Resource missing at instance"}, false
	case 500, 504:
		return streamUpdate{Status : "Error", Message : "Server error at instance"}, false
	default:
		return streamUpdate{Status : "Error", Message : fmt.Sprintf("Unknown status code: %d", resp.Status)}, false
	}
}