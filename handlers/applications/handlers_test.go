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
    "strings"
    "bytes"
    "io/ioutil"
    "encoding/json"
    "reflect"
    "net/http"
    "net/http/httptest"

    "github.com/gorilla/mux"

    "github.com/stretchr/testify/assert"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/session"
    "github.com/lighthouse/lighthouse/handlers/batch"
)

func Test_HandleCreateApplication(t *testing.T) {
    auth.SetupTestingTable()
    defer auth.TeardownTestingTable()

    auth.CreateUser("email", "", "")
    user, _ := auth.GetUser("email")

    m := mux.NewRouter()
    m.HandleFunc("/create", handleCreateApplication)

    type testCase struct {
        Command map[string]interface{}
        Start bool
        Pull bool
    }

    type testResult struct {
        Code int
        Succeeds bool
    }

    type requestObj struct {
        Name string
        Command map[string]interface{}
        Instances []string
    }

    validCommand := map[string]interface{}{"Image" : "image"}
    noImageCommand := map[string]interface{}{"Something" : "else"}
    emptyCommand := map[string]interface{}{}

    tests := map[*testCase]testResult {
        // Completely valid
        &testCase{validCommand, true, true} : testResult{200, true},
        &testCase{validCommand, false, true} : testResult{200, true},
        &testCase{validCommand, true, false} : testResult{200, true},
        &testCase{validCommand, false, false} : testResult{200, true},
        
        // Missing image - fails at pull
        &testCase{noImageCommand, true, true} : testResult{200, false},
        &testCase{noImageCommand, false, true} : testResult{200, false},
        &testCase{noImageCommand, true, false} : testResult{200, true},
        &testCase{noImageCommand, false, false} : testResult{200, true},

        // No command at all - fails in handler
        &testCase{emptyCommand, true, true} : testResult{400, false},
        &testCase{emptyCommand, false, true} : testResult{400, false},
        &testCase{emptyCommand, true, false} : testResult{400, false},
        &testCase{emptyCommand, false, false} : testResult{400, false},
    }

    for c, res := range tests {
        sawStart, sawPull, sawCreate := !c.Start, !c.Pull, false

        h := func(w http.ResponseWriter, r *http.Request) {
            sawStart  = sawStart  || strings.Contains(r.URL.Path, "start")
            sawPull   = sawPull   || strings.Contains(r.URL.Path, "images/create")
            sawCreate = sawCreate || strings.Contains(r.URL.Path, "containers/create")
        }

        SetupTestingTable()
        insts, servers := batch.SetupServers(h)

        endpoint := fmt.Sprintf("%s/create?start=%v&forcePull=%v", insts[0], c.Start, c.Pull)
        dataObj := requestObj{"TestApp", c.Command, insts}
        data, _ := json.Marshal(dataObj)

        req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
        session.SetValue(req, "auth", "email", "email")

        w := httptest.NewRecorder()
        m.ServeHTTP(w, req)

        assert.Equal(t, res.Code, w.Code)
        assert.True(t, !res.Succeeds || sawStart)
        assert.True(t, !res.Succeeds || sawPull)
        assert.True(t, !res.Succeeds || sawCreate)

        if res.Succeeds {
            assert.Equal(t, auth.OwnerAuthLevel, user.GetAuthLevel("Applications", "TestApp"))
        
            var app applicationData
            applications.SelectRow(nil, nil, nil, &app)
            assert.Equal(t, "TestApp", app.Name)

            var deploy deploymentData
            deployments.SelectRow(nil, nil, nil, &deploy)
            assert.Equal(t, c.Command, deploy.Command)
        }

        TeardownTestingTable()
        batch.ShutdownServers(servers)
    }
}

func Test_HandleListApplications(t *testing.T) {
    SetupTestingTable()
    defer TeardownTestingTable()
    auth.SetupTestingTable()
    defer auth.TeardownTestingTable()

    auth.CreateUser("email", "", "")
    user, _ := auth.GetUser("email")

    appCnt := 100
    apps := make([]applicationData, appCnt)
    keyList := make([]applicationData, 0)

    for i := 0; i < appCnt; i++ {
        apps[i], _ = addApplication(
            fmt.Sprint("App", i), []string{fmt.Sprint("Instance", i)},
        )

        // Every 4th is not authorized
        authLevel := i % 4 - 1
        auth.SetUserApplicationAuthLevel(user, apps[i].Name, authLevel)

        if authLevel >= 0 {
            keyList = append(keyList, apps[i])
        }
    }

    req, _ := http.NewRequest("GET", "/list", nil)
    session.SetValue(req, "auth", "email", user.Email)
    w := httptest.NewRecorder()

    m := mux.NewRouter()
    m.HandleFunc("/list", handleListApplications)
    m.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)

    var list []applicationData
    body, _ := ioutil.ReadAll(w.Body)
    json.Unmarshal(body, &list)

    assert.Equal(t, len(keyList), len(list))

    for i, _ := range keyList {
        if !reflect.DeepEqual(keyList[i], list[i]) {
            t.Errorf(
                "At least one wrong application.\nExpected %v\nWas %v",
                keyList[i], list[i],
            )
            return
        }
    }
}

func Test_HandleGetApplicationHistory(t *testing.T) {
    SetupTestingTable()
    defer TeardownTestingTable()
    auth.SetupTestingTable()
    defer auth.TeardownTestingTable()

    m := mux.NewRouter()
    m.HandleFunc("/list/{Id}", handleGetApplicationHistory)
    
    auth.CreateUser("email", "", "")
    user, _ := auth.GetUser("email")
    app, _ := addApplication("TestApp",  []string{})
              addApplication("OtherApp", []string{})

    type keyType struct {
        Id int64
        Creator string
        Date string
    }

    deployCnt := 100
    deploys := make([]deploymentData, deployCnt)
    keyList := make([]keyType, 0)
    cmd := map[string]interface{}{}

    for i := 0; i < deployCnt; i++ {
        appId := int64(i % 2)
        deploys[i], _ = addDeployment(appId, cmd, user.Email)

        if appId == app.Id {
            keyList = append(keyList, keyType{
                deploys[i].Id, user.Email, deploys[i].Date,
            })
        }
    }

    tests := map[int][]keyType {
        -1 : []keyType{},
         0 : keyList,
         1 : keyList,
         2 : keyList,
    }

    for level, key := range tests {
        auth.SetUserApplicationAuthLevel(user, app.Name, level)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest("GET", "/list/0", nil)
        session.SetValue(req, "auth", "email", user.Email)
        m.ServeHTTP(w, req)

        var list []keyType
        body, _ := ioutil.ReadAll(w.Body)
        json.Unmarshal(body, &list)

        assert.Equal(t, key, list)
    }
}

func Test_HandleGetApplicationHistory_Errors(t *testing.T) {
    SetupTestingTable()
    defer TeardownTestingTable()

    m := mux.NewRouter()
    m.HandleFunc("/list/{Id}", handleGetApplicationHistory)

    tests := []string{
        "/list/-1",
        "/list/100",
        "/list/BadName",
    }

    for _, dest := range tests {
        w := httptest.NewRecorder()
        req, _ := http.NewRequest("GET", dest, nil)
        m.ServeHTTP(w, req)
        assert.Equal(t, 404, w.Code)
    }
}

func Test_HandleStartAndStopApplication(t *testing.T) {
    SetupTestingTable()
    defer TeardownTestingTable()
    auth.SetupTestingTable()
    defer auth.TeardownTestingTable()

    auth.CreateUser("email", "", "")
    user, _ := auth.GetUser("email")
    app, _ := addApplication("TestApp", []string{})

    m := mux.NewRouter()
    m.HandleFunc("/start/{Id}", handleStartApplication)
    m.HandleFunc("/stop/{Id}", handleStopApplication)

    type testCase struct {
        AuthLevel int
        Endpoint string
        InitialState bool
    }

    type testResult struct {
        Code int
        Succeeds bool
        FinalState bool
    }

    tests := map[testCase]testResult {
        // Normal cases
        testCase{auth.OwnerAuthLevel, "/stop/0", true}         : testResult{200, true, false},
        testCase{auth.OwnerAuthLevel, "/stop/TestApp", true}   : testResult{200, true, false},
        testCase{auth.OwnerAuthLevel, "/start/0", false}       : testResult{200, true, true},
        testCase{auth.OwnerAuthLevel, "/start/TestApp", false} : testResult{200, true, true},

        // Bad identifiers
        testCase{auth.OwnerAuthLevel, "/stop/-1", true}       : testResult{404, false, false},
        testCase{auth.OwnerAuthLevel, "/stop/BadApp", true}   : testResult{404, false, false},
        testCase{auth.OwnerAuthLevel, "/start/-1", false}     : testResult{404, false, true},
        testCase{auth.OwnerAuthLevel, "/start/BadApp", false} : testResult{404, false, true},

        // State not changed
        testCase{auth.OwnerAuthLevel, "/stop/0", false} : testResult{304, false, false},
        testCase{auth.OwnerAuthLevel, "/start/0", true} : testResult{304, false, true},

        // Not authorized
        testCase{auth.AccessAuthLevel, "/stop/0", true}   : testResult{403, false, true},
        testCase{auth.AccessAuthLevel, "/start/0", false} : testResult{403, false, false},
    }

    for c, res := range tests {
        auth.SetUserApplicationAuthLevel(user, app.Name, c.AuthLevel)
        req, _ := http.NewRequest("POST", c.Endpoint, nil)
        session.SetValue(req, "auth", "email", user.Email)
        setApplicationStateTo(app.Id, c.InitialState, httptest.NewRecorder())

        w := httptest.NewRecorder()
        m.ServeHTTP(w, req)

        assert.Equal(t, res.Code, w.Code)
        if res.Succeeds {
            testApp, _ := GetApplicationById(app.Id)
            assert.Equal(t, res.FinalState, testApp.Active)
        }
    }
}

func Test_HandleRevertApplication(t *testing.T) {
    SetupTestingTable()
    defer TeardownTestingTable()
    auth.SetupTestingTable()
    defer auth.TeardownTestingTable()

    auth.CreateUser("email", "", "")
    user, _ := auth.GetUser("email")

    m := mux.NewRouter()
    m.HandleFunc("/revert/{Id}", handleRevertApplication)

    app, _ := addApplication("TestApp", []string{})
    target, _ := addDeployment(app.Id, map[string]interface{}{}, user.Email)
    // Current deployment
    addDeployment(app.Id, map[string]interface{}{}, user.Email)

    auth.SetUserApplicationAuthLevel(user, app.Name, auth.OwnerAuthLevel)
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/revert/TestApp", nil)
    session.SetValue(req, "auth", "email", user.Email)

    m.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)

    testApp, _ := GetApplicationById(app.Id)
    assert.Equal(t, target.Id, testApp.CurrentDeployment)
}

func Test_HandleRevertApplication_Errors(t *testing.T) {
    SetupTestingTable()
    defer TeardownTestingTable()
    auth.SetupTestingTable()
    defer auth.TeardownTestingTable()

    auth.CreateUser("email", "", "")
    user, _ := auth.GetUser("email")
    app, _ := addApplication("TestApp", []string{})
    addDeployment(app.Id, map[string]interface{}{}, user.Email)

    m := mux.NewRouter()
    m.HandleFunc("/revert/{Id}", handleRevertApplication)
    
    type testCase struct {
        Endpoint string
        AuthLevel int
    }

    tests := map[testCase]int {
        // Bad ID
        testCase{"/revert/-1", auth.OwnerAuthLevel}      : 404,
        testCase{"/revert/100", auth.OwnerAuthLevel}     : 404,
        testCase{"/revert/BadName", auth.OwnerAuthLevel} : 404,

        // Unauthorized
        testCase{"/revert/TestApp", auth.AccessAuthLevel} : 403,

        // Bad target
        testCase{"/revert/TestApp?target=-100", auth.OwnerAuthLevel} : 400,
        testCase{"/revert/TestApp?target=100", auth.OwnerAuthLevel}  : 404,
        testCase{"/revert/TestApp", auth.OwnerAuthLevel}             : 400,
    }

    for c, res := range tests {
        auth.SetUserApplicationAuthLevel(user, app.Name, c.AuthLevel)
        w := httptest.NewRecorder()
        req, _ := http.NewRequest("GET", c.Endpoint, nil)
        session.SetValue(req, "auth", "email", user.Email)

        m.ServeHTTP(w, req)
        assert.Equal(t, res, w.Code)
        t.Log(w.Body)
    }
}

func Test_HandleUpdateApplication_Errors(t *testing.T) {
    SetupTestingTable()
    defer TeardownTestingTable()
    auth.SetupTestingTable()
    defer auth.TeardownTestingTable()

    auth.CreateUser("email", "", "")
    user, _ := auth.GetUser("email")
    app, _ := addApplication("TestApp", []string{})

    m := mux.NewRouter()
    m.HandleFunc("/update/{Id}", handleUpdateApplication)
    
    type testCase struct {
        Endpoint string
        AuthLevel int
    }

    tests := map[testCase]int {
        // Bad ID
        testCase{"/update/-1", auth.OwnerAuthLevel}      : 404,
        testCase{"/update/100", auth.OwnerAuthLevel}     : 404,
        testCase{"/update/BadName", auth.OwnerAuthLevel} : 404,

        // Unauthorized
        testCase{"/update/TestApp", auth.AccessAuthLevel} : 403,
    }

    for c, res := range tests {
        auth.SetUserApplicationAuthLevel(user, app.Name, c.AuthLevel)
        w := httptest.NewRecorder()
        req, _ := http.NewRequest("GET", c.Endpoint, nil)
        session.SetValue(req, "auth", "email", user.Email)

        m.ServeHTTP(w, req)
        assert.Equal(t, res, w.Code)
    }
}

func Test_HandleUpdateApplication(t *testing.T) {
    SetupTestingTable()
    defer TeardownTestingTable()
    auth.SetupTestingTable()
    defer auth.TeardownTestingTable()

    h := func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
    }

    insts, servers := batch.SetupServers(h, h, h, h, h, h, h, h)
    defer batch.ShutdownServers(servers)

    initialInsts := insts[:4]
    addedInsts := insts[4:]
    removeInsts := []string{insts[1], insts[3], insts[2]}
    finalInsts := append([]string{insts[0]}, addedInsts...)

    command := map[string]interface{}{"Image" : "test", "Pass" : true}

    auth.CreateUser("email", "", "")
    user, _ := auth.GetUser("email")

    app, _ := addApplication("TestApp", initialInsts)
    dep, _ := addDeployment(app.Id, map[string]interface{}{}, user.Email)
    doDeployment(app, dep, false, false, httptest.NewRecorder())

    auth.SetUserApplicationAuthLevel(user, app.Name, auth.OwnerAuthLevel)

    m := mux.NewRouter()
    m.HandleFunc("/update/{Id}", handleUpdateApplication)

    dataObj := struct {
        Add []string
        Remove []string
        Command map[string]interface{}
    }{
        addedInsts, 
        removeInsts,
        command,
    }

    data, _ := json.Marshal(dataObj)
    req, _ := http.NewRequest("GET", "/update/TestApp", bytes.NewBuffer(data))
    session.SetValue(req, "auth", "email", user.Email)
    w := httptest.NewRecorder()

    m.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)

    testApp, _ := GetApplicationById(app.Id)
    assert.Equal(t, finalInsts, testApp.Instances)
    assert.Equal(t, dep.Id + 1, testApp.CurrentDeployment)

    var testDep deploymentData
    opts := databases.SelectOptions{Top: 1, OrderBy: []string{"Id"}, Desc : true}
    deployments.SelectRow(nil, nil, &opts, &testDep)

    t.Log(deployments)

    assert.Equal(t, dep.Id + 1, testDep.Id)
    assert.Equal(t, command, testDep.Command)
    assert.Equal(t, app.Id, testDep.AppId)
}