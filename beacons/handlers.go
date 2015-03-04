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
    "time"
    "strconv"
    "net"
	"net/http"
	"encoding/json"
	"io/ioutil"

    "github.com/gorilla/mux"

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
                databases.EmptyKeyError, databases.DuplicateKeyError,
                NotEnoughParametersError:
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
    params, ok := handlers.GetEndpointParams(r, []string{"Beacon", "UserId"})
    if ok == false || len(params) < 2 {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    beacon := getInstanceAlias(params["Beacon"])
    userId := params["UserId"]

    data, err := getBeaconData(beacon)

    if err == nil {
        if data.Users == nil {
            data.Users = userMap{}
        }

        data.Users[userId] = true
        err = updateBeaconField("Users", data.Users, beacon)
    }

    writeResponse(err, w)
}

func handleRemoveUserFromBeacon(w http.ResponseWriter, r *http.Request) {
    params, ok := handlers.GetEndpointParams(r, []string{"Beacon", "UserId"})
    if ok == false || len(params) < 2 {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    beacon := getInstanceAlias(params["Beacon"])
    userId := params["UserId"]

    data, err := getBeaconData(beacon)
    if err == nil {
        delete(data.Users, userId)
        err = updateBeaconField("Users", data.Users, beacon)
    }

    writeResponse(err, w)
}

func handleUpdateBeaconToken(w http.ResponseWriter, r *http.Request) {
    params, ok := handlers.GetEndpointParams(r, []string{"Beacon"})
    if ok == false || len(params) < 1 {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    beacon := getInstanceAlias(params["Beacon"])

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        writeResponse(err, w)
        return
    }

    var token string

    err = json.Unmarshal(reqBody, &token)
    if err == nil {
        err = updateBeaconField("Token", token, beacon)
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
        Alias string
        Users []string
    }

    err = json.Unmarshal(reqBody, &beaconInfo)
    if err != nil {
        return
    }

    if beaconInfo.Address == "" || beaconInfo.Alias == "" {
        err = NotEnoughParametersError
        return
    }

    beacon := beaconData{beaconInfo.Address, beaconInfo.Token, userMap{}}

    currentUser := session.GetValueOrDefault(r, "auth", "email", "").(string)
    beacon.Users[currentUser] = true

    for _, user := range beaconInfo.Users {
        beacon.Users[user] = true
    }

    _, err = net.DialTimeout("ip", "http://" + beacon.Address, 
        time.Duration(3) * time.Second)
    if err != nil {
        //return
    }

    err = aliases.AddAlias(beaconInfo.Alias, beaconInfo.Address)
    if err != nil {
        return
    }

    err = addBeacon(beacon)
    if err != nil {
        return
    }

    err = refreshVMListOf(beacon)

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

    refreshParam := r.URL.Query().Get("refresh")
    refresh, ok := strconv.ParseBool(refreshParam)

    instances, err := getInstancesList(beacon, user, refresh && (ok == nil))
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

func handleRefreshBeacon(w http.ResponseWriter, r *http.Request) {
    beacon := mux.Vars(r)["Beacon"]
    data, err := getBeaconData(beacon)

    if err == nil {
        err = refreshVMListOf(data)
    }

    writeResponse(err, w) 
}