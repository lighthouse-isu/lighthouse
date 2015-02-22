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

    "github.com/lighthouse/lighthouse/beacons/aliases"
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/handlers"
    "github.com/lighthouse/lighthouse/session"
)

func getInstanceAlias(instance string) string {
    alias, err := aliases.GetAddressOf(instance)
    if err != nil {
        return instance
    }
    return alias
}

func writeResponse(err error, w http.ResponseWriter) {
    var code int

    switch err {
        case databases.KeyNotFoundError, databases.NoUpdateError, 
                databases.EmptyKeyError, NotEnoughParametersError:
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

func handleAddUserToBeacon(w http.ResponseWriter, r *http.Request) {
    params, ok := handlers.GetEndpointParams(r, []string{"Instance", "UserId"})
    if ok == false || len(params) < 2 {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    instance := getInstanceAlias(params["Instance"])
    userId := params["UserId"]

    beacon, err := getBeaconData(instance)

    if err == nil {
        if beacon.Users == nil {
            beacon.Users = userMap{}
        }

        beacon.Users[userId] = true
        err = updateBeaconField("Users", beacon.Users, instance)
    }

    writeResponse(err, w)
}

func handleRemoveUserFromBeacon(w http.ResponseWriter, r *http.Request) {
    params, ok := handlers.GetEndpointParams(r, []string{"Instance", "UserId"})
    if ok == false || len(params) < 2 {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    instance := getInstanceAlias(params["Instance"])
    userId := params["UserId"]

    beacon, err := getBeaconData(instance)
    if err == nil {
        delete(beacon.Users, userId)
        err = updateBeaconField("Users", beacon.Users, instance)
    }

    writeResponse(err, w)
}

func handleUpdateBeaconToken(w http.ResponseWriter, r *http.Request) {
    params, ok := handlers.GetEndpointParams(r, []string{"Instance"})
    if ok == false || len(params) < 1 {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    instance := getInstanceAlias(params["Instance"])

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        writeResponse(err, w)
        return
    }

    var token string

    err = json.Unmarshal(reqBody, &token)
    if err == nil {
        err = updateBeaconField("Token", token, instance)
    }

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
        Users []string
    }

    err = json.Unmarshal(reqBody, &beaconInfo)
    if err != nil {
        return
    }

    beacon := beaconData{"", beaconInfo.Address, beaconInfo.Token, userMap{}}

    currentUser := session.GetValueOrDefault(r, "auth", "email", "").(string)
    beacon.Users[currentUser] = true

    for _, user := range beaconInfo.Users {
        beacon.Users[user] = true
    }

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

    for _, vm := range vms {
        beacon.InstanceAddress = fmt.Sprintf("%s:%s/%s", vm.Address, vm.Port, vm.Version)
        
        if !instanceExists(beacon.InstanceAddress) {
            addInstance(beacon)
        }
    }

    return
}

func handleListBeacons(w http.ResponseWriter, r *http.Request) {
    user := session.GetValueOrDefault(r, "auth", "email", "").(string)

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
    user := session.GetValueOrDefault(r, "auth", "email", "").(string)

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