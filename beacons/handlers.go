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
    "strconv"
	"net/http"
	"encoding/json"
	"io/ioutil"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/beacons/aliases"
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/handlers"
)

func getAddressOf(alias string) string {
    address, err := aliases.GetAddressOf(alias)
    if err != nil {
        return alias
    }
    return address
}

func writeResponse(err error, w http.ResponseWriter) {
    switch err {
        case nil:
            w.WriteHeader(http.StatusOK)

        // Errors outside of package
        case databases.KeyNotFoundError, databases.DuplicateKeyError,
                databases.NoUpdateError, databases.EmptyKeyError:
            handlers.WriteError(w, http.StatusBadRequest, "beacons", err.Error())

        case TokenPermissionError:
            handlers.WriteError(w, http.StatusForbidden, "beacons", err.Error())

        case NotEnoughParametersError, DuplicateBeaconError:
            handlers.WriteError(w, http.StatusBadRequest, "beacons", err.Error())

        default:
            handlers.WriteError(w, http.StatusInternalServerError, "beacons", err.Error())
    }    
}

func handleUpdateBeaconToken(w http.ResponseWriter, r *http.Request) {
    var err error = nil
    defer func() { writeResponse(err, w) }()

    params, ok := handlers.GetEndpointParams(r, []string{"Beacon"})

    if ok == false || len(params) < 1 || params["Beacon"] == "" {
        err = NotEnoughParametersError
        return
    }

    beacon := getAddressOf(params["Beacon"])

    user := auth.GetCurrentUser(r)

    if !user.CanModifyBeacon(beacon) {
        err = TokenPermissionError
        return
    }

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return
    }

    var token string

    err = json.Unmarshal(reqBody, &token)
    if err != nil {
        err = NotEnoughParametersError
        return
    }

    to := map[string]interface{}{"Token" : token}
    where := map[string]interface{}{"Address" : beacon}
    err = beacons.UpdateSchema(to, where)

    return
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
    }

    err = json.Unmarshal(reqBody, &beaconInfo)
    if err != nil {
        err = NotEnoughParametersError
        return
    }

    if beaconInfo.Address == "" || beaconInfo.Alias == "" {
        err = NotEnoughParametersError
        return
    }

    beacon := beaconData{beaconInfo.Address, beaconInfo.Token}

    currentUser := auth.GetCurrentUser(r)
    auth.SetUserBeaconAuthLevel(currentUser, beacon.Address, auth.OwnerAuthLevel)

    err = aliases.AddAlias(beaconInfo.Alias, beaconInfo.Address)
    if err != nil {
        return
    }

    err = addBeacon(beacon)
    if err == databases.DuplicateKeyError {
        err = DuplicateBeaconError
    }

    if err != nil {
        aliases.RemoveAlias(beaconInfo.Address)
        return
    }

    err = refreshVMListOf(beacon)
    if err != nil {
        aliases.RemoveAlias(beaconInfo.Address)
        removeBeacon(beacon.Address)
        return
    }

    return
}

func handleListBeacons(w http.ResponseWriter, r *http.Request) {
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