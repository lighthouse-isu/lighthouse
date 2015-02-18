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

package beacons

import (
	"fmt"
	"errors"
	"net/http"
	"encoding/json"
	"io/ioutil"

    "github.com/gorilla/mux"

	beaconStructs "github.com/lighthouse/beacon/structs"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/beacons/aliases"
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/handlers"
)

func getInstanceAlias(instance string) string {
    alias, err := aliases.GetAlias(instance)
    if err != nil {
        return instance
    }
    return alias
}

func writeResponse(err error, w http.ResponseWriter) {
    var code int

    switch err {
        case databases.KeyNotFoundError, databases.NoUpdateError, databases.EmptyKeyError:
            code = http.StatusBadRequest

        case NotEnoughParametersError, DuplicateInstanceError:
            code = http.StatusBadRequest

        case nil:
            code = http.StatusOK

        default:
            code = http.StatusInternalServerError
    }

    w.WriteHeader(code)

    if err != nil {
        fmt.Fprint(w, err)
    }
}

func handleUpdateBeaconToken(w http.ResponseWriter, r *http.Request) {
    params, ok := handlers.GetEndpointParams(r, []string{"Beacon"})
    if ok == false || len(params) < 1 {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    beaconAddr := getInstanceAlias(params["Beacon"])

    if !auth.GetCurrentUser(r).CanModifyBeacon(beaconAddr) {
        writeResponse(TokenPermissionError, w)
        return
    }

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        writeResponse(err, w)
        return
    }

    var token string

    err = json.Unmarshal(reqBody, &token)
    if err != nil {
        writeResponse(err, w)
        return
    }

    to := map[string]interface{}{"Token" : token}
    where := map[string]interface{}{"BeaconAddress" : beaconAddr}

    err = getDBSingleton().UpdateSchema(to, where)
    writeResponse(err, w)
}

func handleBeaconCreate(w http.ResponseWriter, r *http.Request) {
    var err error = nil
    defer func() { writeResponse(err, w) }()

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return
    }

    var beaconInfo struct {
        Address string
        Token string
    }

    err = json.Unmarshal(reqBody, &beaconInfo)
    if err != nil {
        return
    }

    beacon := beaconData{"", beaconInfo.Address, beaconInfo.Token}
    vmsTarget := fmt.Sprintf("http://%s/vms", beacon.BeaconAddress)

    req, err := http.NewRequest("GET", vmsTarget, nil)
    if err != nil {
        return
    }

    // Assuming user has permission to access token since they provided it
    req.Header.Set(HEADER_TOKEN_KEY, beaconInfo.Token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return
    }

    defer resp.Body.Close()

    vmsBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return
    }

    if resp.StatusCode != http.StatusOK {
        err = errors.New(string(vmsBody))
        return
    }

    var vms []beaconStructs.VM

    err = json.Unmarshal(vmsBody, &vms)
    if err != nil {
        return
    }

    currentUser := auth.GetCurrentUser(r)

    instances := make([]string, len(vms))

    for i, vm := range vms {
        instanceAddr := fmt.Sprintf("%s:%s/%s", vm.Address, vm.Port, vm.Version)
        
        if instanceExists(instanceAddr) {
            err = DuplicateInstanceError
            return
        }

        instances[i] = instanceAddr
    }

    for _, instance := range instances {
        beacon.InstanceAddress = instance
        addInstance(beacon)
    }

    auth.SetUserBeaconAuthLevel(currentUser, beacon.BeaconAddress, auth.OwnerAuthLevel)

    return
}

func handleListBeacon(w http.ResponseWriter, r *http.Request) {
    user := auth.GetCurrentUser(r)

    beacons, err := getBeaconsList(user)
    var output []byte

    if err == nil {
        output, err = json.Marshal(beacons)
    } 

    if err != nil {
        writeResponse(err, w) 
    } else {
        fmt.Fprint(w, string(output))
    }
}

func handleListInstances(w http.ResponseWriter, r *http.Request) {
    beacon := mux.Vars(r)["Beacon"]
    user := auth.GetCurrentUser(r)

    instances, err := getInstancesList(beacon, user)
    var output []byte

    if err == nil {
        output, err = json.Marshal(instances)
    } 

    if err != nil {
        writeResponse(err, w) 
    } else {
        fmt.Fprint(w, string(output))
    }
}