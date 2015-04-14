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
	"io/ioutil"
	"net/http"
	"strconv"
	"sort"
	"math"
	"fmt"

	"github.com/gorilla/mux"

	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/handlers"
	"github.com/lighthouse/lighthouse/handlers/batch"
	"github.com/lighthouse/lighthouse/databases"
)

func getBoolParamOrDefault(r *http.Request, param string, def bool) bool {
    val := r.URL.Query().Get(param)
    ret, err := strconv.ParseBool(val)
    if err != nil {
    	return def
    }
    return ret
}

func getInt64ParamOrDefault(r *http.Request, param string, def int64) int64 {
    val := r.URL.Query().Get(param)
    ret, err := strconv.ParseInt(val, 10, 64)
    if err != nil {
    	return def
    }
    return int64(ret)
}

func getAppIdByIdentifier(identifier string) (int64, error) {
	if asInt, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		return int64(asInt), nil
	} else {
		app, err := GetApplicationByName(identifier)
		if err != nil {
			return -1, err
		}
		return app.Id, nil
	}
}

func writeError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	code, ok := map[error]int {
		UnknownApplicationError : 404,
		UnknownDeploymentError : 404,
		DeploymentMismatchError : 400,
		NotEnoughDeploymentsError : 400,
		NotEnoughParametersError : 400,
		ApplicationPermissionError : 403,
		databases.NoUpdateError : 400,
	}[err]

	if !ok {
		code = 500
	}

	handlers.WriteError(w, code, "applications", err.Error())
}

func getDifferenceOf(orig, remove []string) []string {
	if len(remove) == 0 {
		return orig
	}

	ret := []string{}

	sort.Strings(orig)
	sort.Strings(remove)

	i, j := 0, 0
	for i < len(orig) {
		for j < len(remove) && remove[j] < orig[i] {
			j++
		}

		if j >= len(remove) || remove[j] > orig[i] {
			ret = append(ret, orig[i])
		}

		i++
	}
	
	return ret
}

func handleCreateApplication(w http.ResponseWriter, r *http.Request) {
   	var err error
	defer func() { writeError(w, err) }()

	w.Header().Set("Content-Type", "application/json")

    start := getBoolParamOrDefault(r, "start", false)
    pull := getBoolParamOrDefault(r, "forcePull", false)
    user := auth.GetCurrentUser(r)

    body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	create := struct {
		Name string
		Command map[string]interface{}
		Instances []interface{}
	}{}

	err = json.Unmarshal(body, &create)
	if err != nil {
		return
	}

	instanceList, ok := convertInstanceList(create.Instances)
	if !ok {
		err = NotEnoughParametersError
		return
	}

	if create.Name == "" || len(create.Command) == 0 || len(instanceList) == 0 {
		err = NotEnoughParametersError
		return
	}

	application, err := addApplication(create.Name, instanceList)
    if err != nil {
    	return
    }

    deployment, err := addDeployment(application.Id, create.Command, user.Email)
    if err != nil {
    	removeApplication(application.Id)
        return
    }

    deployErr, ok := doDeployment(user, application, deployment, start, pull, w)
    if !ok {
        removeDeployment(deployment.Id)
        removeApplication(application.Id)

        if deployErr == NotEnoughParametersError {
        	err = NotEnoughParametersError
        }
        
        return
    }

    auth.SetUserApplicationAuthLevel(user, application.Name, auth.OwnerAuthLevel)

	return
}

func handleListApplications(w http.ResponseWriter, r *http.Request) {
   	var err error
	defer func() { writeError(w, err) }()

	apps, err := getApplicationList(auth.GetCurrentUser(r))
	if err != nil {
		return
	}

	jsonApps, err := json.Marshal(apps)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "%v", string(jsonApps))
}

func handleGetApplicationHistory(w http.ResponseWriter, r *http.Request) {
   	var err error
	defer func() { writeError(w, err) }()

	countParam := getInt64ParamOrDefault(r, "count", math.MaxInt32)
	if countParam < 0 {
		err = NotEnoughParametersError
		return
	}

	count := int(countParam)

    id, err := getAppIdByIdentifier(mux.Vars(r)["Id"])
	if err != nil {
		return
	}

	app, err := GetApplicationById(id)
	if err != nil {
		return
	}

	hist, err := getApplicationHistory(auth.GetCurrentUser(r), app)
	if err != nil {
		return
	}

	if count < len(hist) {
		hist = hist[:count]
	}

	jsonHist, err := json.Marshal(hist)
	if err != nil {
		return
	}

	fmt.Fprint(w, string(jsonHist))
}

func handleStartApplication(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() { writeError(w, err) }()
	
	w.Header().Set("Content-Type", "application/json")

	user := auth.GetCurrentUser(r)

	id, err := getAppIdByIdentifier(mux.Vars(r)["Id"])
	if err != nil {
		return
	}

	app, err := GetApplicationById(id)
	if err != nil {
		return
	}

	if !user.CanModifyApplication(app.Name) {
		err = ApplicationPermissionError
		return
	}

	startApplication(user, app.Id, w)
}

func handleStopApplication(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() { writeError(w, err) }()
	
	w.Header().Set("Content-Type", "application/json")

	user := auth.GetCurrentUser(r)

	id, err := getAppIdByIdentifier(mux.Vars(r)["Id"])
	if err != nil {
		return
	}

	app, err := GetApplicationById(id)
	if err != nil {
		return
	}

	if !user.CanModifyApplication(app.Name) {
		err = ApplicationPermissionError
		return
	}

	stopApplication(user, app.Id, w)
}

func handleRevertApplication(w http.ResponseWriter, r *http.Request) {
   	var err error
	defer func() { writeError(w, err) }()

	w.Header().Set("Content-Type", "application/json")

	target := getInt64ParamOrDefault(r, "target", -1)
    pull := getBoolParamOrDefault(r, "forcePull", false)
    user := auth.GetCurrentUser(r)

    id, err := getAppIdByIdentifier(mux.Vars(r)["Id"])
	if err != nil {
		return
	}

	app, err := GetApplicationById(id)
	if err != nil {
		return
	}

	if !user.CanModifyApplication(app.Name) {
		err = ApplicationPermissionError
		return
	}

    deploy, err := getRevertDeployment(id, target)
    if err != nil {
		return
	}

	doDeployment(user, app, deploy, false, pull, w)
}

func handleUpdateApplication(w http.ResponseWriter, r *http.Request) {
   	var err error
	defer func() { writeError(w, err) }()

	w.Header().Set("Content-Type", "application/json")

	restart := getBoolParamOrDefault(r, "restart", false)
	user := auth.GetCurrentUser(r)

	fmt.Println(user)

	id, err := getAppIdByIdentifier(mux.Vars(r)["Id"])
	if err != nil {
		return
	}

	app, err := GetApplicationById(id)
	if err != nil {
		return
	}

	if !user.CanModifyApplication(app.Name) {
		err = ApplicationPermissionError
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	update := struct {
		Command map[string]interface{}
		Add []interface{}
		Remove []interface{}
	}{}

	err = json.Unmarshal(body, &update)
	if err != nil {
		return
	}

	addList, addOK := convertInstanceList(update.Add)
	removeList, removeOK := convertInstanceList(update.Remove)
	if !addOK || !removeOK {
		err = NotEnoughParametersError
		return
	}

	var appToDeploy applicationData = app
	var deployment deploymentData

	// Prep latest deployment in case of instance addition or a restart
	where := databases.Filter{"Id" : app.CurrentDeployment}
    err = deployments.SelectRow(nil, where, nil, &deployment)
	if err != nil {
		return
	}

	willDeploy := restart || len(update.Command) > 0 

	if len(addList) > 0 || len(removeList) > 0 {
		app.Instances = getDifferenceOf(app.Instances.([]string), removeList)

		if len(addList) > 0 {
			app.Instances = append(app.Instances.([]string), addList...)
		}

		to := map[string]interface{}{"Instances" : app.Instances}
		where := databases.Filter{"Id" : app.Id}
		err = applications.Update(to, where)
		if err != nil {
			return
		}
	}

	if len(update.Command) > 0 {
		deployment, err = addDeployment(app.Id, update.Command, auth.GetCurrentUser(r).Email)
		if err != nil {
			return
		}
	}

	if len(addList) > 0 && !willDeploy {
		// A temporary app that only contains the added instances
		appToDeploy = applicationData {
			app.Id, app.CurrentDeployment, app.Name, addList,
		}

		willDeploy = true
	}

	if len(removeList) > 0 {
		proc := batch.NewProcessor(user, w, removeList)
		batchDeleteContainersByName(proc, app.Name, false)
	}
	
	if willDeploy {
		doDeployment(user, appToDeploy, deployment, false, true, w)
	} else {
		w.WriteHeader(204)
	}
}