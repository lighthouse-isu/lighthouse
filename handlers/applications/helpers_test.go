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
    "testing"

    "fmt"
    "errors"
    "strings"
    "net/http"
    "net/http/httptest"

    "github.com/stretchr/testify/assert"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/handlers/batch"
)

func Test_AddApplication_New(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	keyApp := applicationData {
		Id : 0,
		Name : "TestApp", 
		Active : false,
		CurrentDeployment : -1,
		Instances : []string{"instance"},
	}

	retApp, err := addApplication("TestApp", []string{"instance"})

	assert.Nil(t, err)
	assert.Equal(t, keyApp, retApp)

	var selectApp applicationData
	applications.SelectRow(nil, nil, nil, &selectApp)

	assert.Equal(t, keyApp, selectApp)
}

func Test_AddApplication_Dup(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	app := applicationData {
		Id : 0,
		Name : "TestApp", 
		Active : false,
		CurrentDeployment : -1,
		Instances : []string{"instance"},
	}

	applications.Insert(makeDatabaseEntryFor(app))

	retApp, err := addApplication("TestApp", []string{"instance"})

	assert.NotNil(t, err)
	assert.Equal(t, applicationData{}, retApp)
}

func Test_AddDeployment(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	key := deploymentData {
		Id : 0,
		AppId : 314, 
		Command : map[string]interface{}{"Image" : "image"},
		User : "user",
	}

	retDeploy, err := addDeployment(key.AppId, key.Command, key.User)

	// The time might have changed during insert
	key.Date = retDeploy.Date

	assert.Nil(t, err)
	assert.Equal(t, key, retDeploy)

	var selectDeploy deploymentData
	deployments.SelectRow(nil, nil, nil, &selectDeploy)

	assert.Equal(t, key, selectDeploy)
}

func Test_RemoveApplication(t *testing.T) {
	err := errors.New("TestError")

	applications = &databases.MockTable{
		MockDelete: func(where databases.Filter)(error) {
			assert.Equal(t, int64(42), where["Id"])
			return err
		},
	}
	
	retErr := removeApplication(42)
	assert.Equal(t, err, retErr)

	TeardownTestingTable()
}

func Test_RemoveDeployment(t *testing.T) {
	err := errors.New("TestError")

	deployments = &databases.MockTable{
		MockDelete: func(where databases.Filter)(error) {
			assert.Equal(t, int64(42), where["Id"])
			return err
		},
	}
	
	retErr := removeDeployment(42)
	assert.Equal(t, err, retErr)

	TeardownTestingTable()
}

func Test_StartStopApplication_Normal(t *testing.T) {
	type testCase struct {
		Id int64 // Valid app is Id 0
		Active bool
		ControlStatus int
		TestFunc func(app int64, w http.ResponseWriter)error
	}

	type testResult struct {
		FinalState bool
		AppError error // Only if applications make the error, not batch
		Requests []string
	}

	tests := map[*testCase]testResult {
		&testCase{0, false, 200, startApplication} : testResult{true,  nil, []string{"start"}},
		&testCase{1, false, -1,  startApplication} : testResult{false, UnknownApplicationError, []string{}},
		&testCase{0, true,  -1,  startApplication} : testResult{true,  StateNotChangedError, []string{}},
		&testCase{0, false, 500, startApplication} : testResult{false, nil, []string{"start", "stop"}},

		&testCase{0, true,  200, stopApplication}  : testResult{false, nil, []string{"stop"}},
		&testCase{1, true,  -1,  stopApplication}  : testResult{true,  UnknownApplicationError, []string{}},
		&testCase{0, false, -1,  stopApplication}  : testResult{false, StateNotChangedError, []string{}},
		&testCase{0, true,  500, stopApplication}  : testResult{true,  nil, []string{"stop", "start"}},
	}

	for c, res := range tests {
		i := 0
		testInst := func(w http.ResponseWriter, r *http.Request) {
			assert.True(t, strings.HasSuffix(r.URL.Path, res.Requests[i]))
			i += 1
		}

		controlInst := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(c.ControlStatus)
		}

		SetupTestingTable()
		insts, servers := batch.SetupServers(testInst, controlInst)
		app := applicationData{Id : 0, Active : c.Active, Instances : insts}
		applications.Insert(makeDatabaseEntryFor(app))

		err := c.TestFunc(c.Id, httptest.NewRecorder())

		assert.Equal(t, len(res.Requests), i)

		if c.ControlStatus == 200 {
			assert.Nil(t, err)
		} else if res.AppError != nil {
			assert.Equal(t, res.AppError, err)
		} else {
			assert.NotNil(t, err)
		}

		var resApp applicationData
		applications.SelectRow(nil, nil, nil, &resApp)
		assert.Equal(t, res.FinalState, resApp.Active)

		TeardownTestingTable()
		batch.ShutdownServers(servers)
	}
}

func Test_SetApplicationStateTo_Unknown(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	_, servers := batch.SetupServers(nil)
	defer batch.ShutdownServers(servers)

	err := setApplicationStateTo(0, true, httptest.NewRecorder())
	assert.Equal(t, UnknownApplicationError, err)
}

func Test_SetApplicationStateTo_NoChange(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	insts, servers := batch.SetupServers(nil)
	defer batch.ShutdownServers(servers)

	app := applicationData{Id : 0, Active : true, Instances : insts}
	applications.Insert(makeDatabaseEntryFor(app))

	err := setApplicationStateTo(0, true, httptest.NewRecorder())
	assert.Equal(t, StateNotChangedError, err)
}

func Test_GetApplicationList(t *testing.T) {
	SetupTestingTable()
	auth.SetupTestingTable()
	defer TeardownTestingTable()
	defer auth.SetupTestingTable()

	auth.CreateUser("email", "", "")
	user, _ := auth.GetUser("email")

	acc, _ := addApplication("ACC", []string{"Insts1"})
			  addApplication("BAD", []string{"Insts2"})
	own, _ := addApplication("OWN", []string{"Insts3"})
	mod, _ := addApplication("MOD", []string{"Insts4"})

	user.SetAuthLevel("Applications", "ACC", auth.AccessAuthLevel)
	user.SetAuthLevel("Applications", "OWN", auth.OwnerAuthLevel)
	user.SetAuthLevel("Applications", "MOD", auth.ModifyAuthLevel)

	apps, _ := getApplicationList(user)

	assert.Equal(t, 3, len(apps))
	assert.Equal(t, acc, apps[0])
	assert.Equal(t, own, apps[1])
	assert.Equal(t, mod, apps[2])
}

func Test_GetApplicationHistory_OK(t *testing.T) {
	SetupTestingTable()
	auth.SetupTestingTable()
	defer TeardownTestingTable()
	defer auth.SetupTestingTable()

	auth.CreateUser("email", "", "")
	user, _ := auth.GetUser("email")

	app, _ := addApplication("APP", nil)

	cmd := map[string]interface{}{"Image" : "test"}
	ds := make([]deploymentData, 3)

	ds[0], _ = addDeployment(0, cmd, "otheruser")
	ds[1], _ = addDeployment(0, cmd, "email")
	ds[2], _ = addDeployment(0, cmd, "otheruser")
	addDeployment(1, cmd, "email")

	tests := map[int][]deploymentData {
		-1 : []deploymentData{},
		auth.AccessAuthLevel : ds,
		auth.ModifyAuthLevel : ds,
		auth.OwnerAuthLevel : ds,
	}

	for level, key := range tests {
		user.SetAuthLevel("Applications", "APP", level)
		list, err := getApplicationHistory(user, app)

		assert.Nil(t, err)
		assert.Equal(t, len(key), len(list))

		for i, deploy := range key {
			assert.Equal(t, deploy.Id, list[i]["Id"])
			assert.Equal(t, deploy.User, list[i]["Creator"])
			assert.Equal(t, deploy.Date, list[i]["Date"])
		}
	}
}

func Test_GetRevertDeployment(t *testing.T) {
	SetupTestingTable()
	defer TeardownTestingTable()

	type testCase struct {
		App int64
		Target int64
	}

	type testResult struct {
		Deploy deploymentData
		Error error
	}

	cmd := map[string]interface{}{"Image" : "test"}
	d0, _ := addDeployment(0, cmd, "")
	d1, _ := addDeployment(0, cmd, "")
		     addDeployment(1, cmd, "")
	d3, _ := addDeployment(0, cmd, "")
	dFail := deploymentData{}

	tests := map[testCase]testResult{
		testCase{0, 0} : testResult{d0, nil},
		testCase{0, 1} : testResult{d1, nil},
		testCase{0, 2} : testResult{dFail, DeploymentMismatchError},
		testCase{0, 3} : testResult{d3, nil},
		testCase{0, 4} : testResult{dFail, UnknownDeploymentError},
		testCase{1, 0} : testResult{dFail, DeploymentMismatchError},

		testCase{0, -1} : testResult{d1, nil},
		testCase{0, -2} : testResult{d0, nil},
		testCase{0, -3} : testResult{dFail, NotEnoughDeploymentsError},
		testCase{2, -1} : testResult{dFail, UnknownApplicationError},
	}

	for c, r := range tests {
		dep, err := getRevertDeployment(c.App, c.Target)
		assert.Equal(t, r.Deploy, dep, fmt.Sprint("Test case ", c))
		assert.Equal(t, r.Error, err, fmt.Sprint("Test case ", c))
	}
}

func Test_DoDeployment(t *testing.T) {
	type testCase struct {
		Deploy *deploymentData
		Start bool
		Pull bool
	}

	type testResult struct {
		Requests []int 
		FailureStep int 
	}

	type Request struct {
		Method string
		Dest string
	}

	requestList := []Request {
		Request{"POST",   "/images/create?fromImage?test"},               // 0
		Request{"POST",   "/containers/create?name=TestApp_tmp"},         // 1
		Request{"DELETE", "/containers/TestApp_tmp?force=true"},          // 2
		Request{"DELETE", "/containers/TestApp?force=true"},              // 3
		Request{"POST",   "/containers/TestApp_tmp/rename?name=TestApp"}, // 4
		Request{"POST",   "/containers/TestApp/start"},                   // 5
	}

	cmd := map[string]interface{}{"Image" : "test"}
	dNormal := &deploymentData{42, 0, cmd, "email", "12345"}
	dNoImage := &deploymentData{42, 0, map[string]interface{}{}, "email", "12345"}

	tests := map[testCase]testResult{
		// Success cases
		testCase{dNormal, true, true}   : testResult{[]int{0, 1, 3, 4, 5}, -1}, 
		testCase{dNormal, false, true}  : testResult{[]int{0, 1, 3, 4}, -1},
		testCase{dNormal, true, false}  : testResult{[]int{1, 3, 4, 5}, -1}, 
		testCase{dNormal, false, false} : testResult{[]int{1, 3, 4}, -1}, 

		// Failure cases
		testCase{dNoImage, false, true} : testResult{[]int{}, 0}, // Bad command
		testCase{dNormal, false, true}  : testResult{[]int{0}, 0}, // Bad pull
		testCase{dNormal, false, false} : testResult{[]int{1, 2}, 0}, // Bad create
		testCase{dNormal, false, false} : testResult{[]int{1, 3}, 1}, // Bad delete
		testCase{dNormal, false, false} : testResult{[]int{1, 3, 4, 2}, 2}, // Bad rename
		testCase{dNormal, true, false}  : testResult{[]int{1, 3, 4, 5}, 3}, // Bad start
	}

	for c, res := range tests {
		SetupTestingTable()
		
		errorMsg := fmt.Sprintf("Test case %v with result %v", c, res)

		i := 0
		h := func(w http.ResponseWriter, r *http.Request) {
			if i >= len(res.Requests) {
				t.Errorf("Too many requests for case %v", c)
				w.WriteHeader(500)
				return
			}

			keyRequest := requestList[res.Requests[i]]
			assert.Equal(t, keyRequest.Method, r.Method, errorMsg)
			assert.Equal(t, keyRequest.Dest, r.RequestURI, errorMsg)

			if i == res.FailureStep {
				w.WriteHeader(500)
			} else {
				w.WriteHeader(200)
			}			

			i += 1
		}

		insts, servers := batch.SetupServers(h)
		app, _ := addApplication("TestApp", insts)

		err := doDeployment(app, *c.Deploy, c.Start, c.Pull, httptest.NewRecorder())

		assert.Equal(t, len(res.Requests), i, errorMsg)

		if res.FailureStep == -1 { // Passes
			assert.Nil(t, err, errorMsg)
			app, _ = GetApplicationById(app.Id)
			assert.Equal(t, c.Deploy.Id, app.CurrentDeployment, errorMsg)
		} else {
			assert.NotNil(t, err, errorMsg)
		}

		TeardownTestingTable()
		batch.ShutdownServers(servers)
	}
}