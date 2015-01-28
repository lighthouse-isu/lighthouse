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
    "os"
    "fmt"
    "io/ioutil"
    "net/http"
    "encoding/json"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

const (
    HEADER_TOKEN_KEY = "Token"
)

var beacons *databases.Table

type Beacon struct {
    Address string
    Token string
    Users map[string]bool
}

func getDBSingleton() *databases.Table {
    if beacons == nil {
        beacons = databases.NewTable(postgres.Connection(), "beacons")
    }
    return beacons
}

func AddBeacon(instance string, inst Beacon) error {
    return getDBSingleton().Insert(instance, inst)
}

func UpdateBeacon(instance string, inst Beacon) error {
    return getDBSingleton().Update(instance, inst)
}

func AddUserToBeacon(instance, user string) error {
    beacon, _ := GetBeacon(instance)
    beacon.Users[user] = true
    return UpdateBeacon(instance, beacon)
}

func RemoveUserFromBeacon(instance, user string) error {
    beacon, _ := GetBeacon(instance)
    delete(beacon.Users, user)
    return UpdateBeacon(instance, beacon)
}

func UpdateBeaconToken(instance, token string) error {
    beacon, _ := GetBeacon(instance)
    beacon.Token = token
    return UpdateBeacon(instance, beacon)
}

func UpdateBeaconAddress(instance, address string) error {
    beacon, _ := GetBeacon(instance)
    beacon.Address = address
    return UpdateBeacon(instance, beacon)
}

func GetBeacon(instance string) (Beacon, error) {
    var inst Beacon
    err := getDBSingleton().SelectRow(instance, &inst)

    if err != nil {
        return Beacon{}, err
    }

    return inst, nil
}

func LoadBeacons() map[string]Beacon {
    var fileName string
    if _, err := os.Stat("/config/beacon_permissions.json"); os.IsNotExist(err) {
        fileName = "./config/beacon_permissions.json"
    } else {
        fileName = "/config/beacon_permissions.json"
    }

    configFile, _ := ioutil.ReadFile(fileName)

    var beacons []struct {
        Address string
        Beacon Beacon
    }

    json.Unmarshal(configFile, &beacons)

    perms := make(map[string]Beacon)
    for _, beacon := range beacons {
        perms[beacon.Address] = beacon.Beacon
    }

    return perms
}

func Handle(r *mux.Router) {
    beacons := LoadBeacons()

    for instance, beacon := range beacons {
        AddBeacon(instance, beacon)
    }

    updateRouter := r.PathPrefix("/update").Subrouter()

    updateRouter.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
        handleUpdateRoutes(w, r, AddUserToBeacon)
    }).Methods("PUT")

    updateRouter.HandleFunc("/user/rm", func(w http.ResponseWriter, r *http.Request) {
        handleUpdateRoutes(w, r, RemoveUserFromBeacon)
    }).Methods("PUT")

    updateRouter.HandleFunc("/address", func(w http.ResponseWriter, r *http.Request) {
        handleUpdateRoutes(w, r, UpdateBeaconAddress)
    }).Methods("PUT")

    updateRouter.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
        handleUpdateRoutes(w, r, UpdateBeaconToken)
    }).Methods("PUT")

    r.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
        reqBody, err := ioutil.ReadAll(r.Body)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError) 
            fmt.Fprint(w, err)
        }

        var body struct {
            InstanceAddress string
            BeaconAddress string
            Token string
            Users []string
        }

        err = json.Unmarshal(reqBody, &body)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError) 
            fmt.Fprint(w, err)
        }

        if body.InstanceAddress == "" {
            w.WriteHeader(http.StatusInternalServerError) 
            fmt.Fprint(w, "missing instance's address")
        }

        if body.BeaconAddress == "" || body.Token == "" {
            w.WriteHeader(http.StatusInternalServerError) 
            fmt.Fprint(w, "missing beacon's address or token")
        }

        beacon := Beacon{body.BeaconAddress, body.Token, make(map[string]bool)}
        for _, user := range body.Users {
            beacon.Users[user] = true
        }

        err = AddBeacon(body.InstanceAddress, beacon)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError) 
            fmt.Fprint(w, err)
        }

        w.WriteHeader(http.StatusOK)
    }).Methods("POST")
}

func handleUpdateRoutes(w http.ResponseWriter, r *http.Request, updateFunc func(string, string)(error)) {

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, err)
    }

    var body struct {
        Instance string
        Data string
    }

    err = json.Unmarshal(reqBody, &body)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, err)
    }

    if body.Instance == "" {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, "missing instance to update")
    }

    if body.Data == "" {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, "missing value to update")
    }

    err = updateFunc(body.Instance, body.Data)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, err)
    }

    w.WriteHeader(http.StatusOK)
}