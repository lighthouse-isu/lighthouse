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

    instanceRouter := r.PathPrefix("/{Instance}").Subrouter()

    instanceRouter.HandleFunc("/user/{Id}", handleAddUserToBeacon).Methods("PUT")

    instanceRouter.HandleFunc("/user/{Id}", handleRemoveUserFromBeacon).Methods("DELETE")

    instanceRouter.HandleFunc("/address/{Address}", handleUpdateBeaconAddress).Methods("PUT")

    instanceRouter.HandleFunc("/token", handleUpdateBeaconToken).Methods("PUT")

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

func handleAddUserToBeacon(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)

    instance := vars["Instance"]
    userId := vars["Id"]

    beacon, _ := GetBeacon(instance)
    beacon.Users[userId] = true
    err := UpdateBeacon(instance, beacon)

    if err != nil {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, err)
    } else {
        w.WriteHeader(http.StatusOK)
    }
}

func handleRemoveUserFromBeacon(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)

    instance := vars["Instance"]
    userId := vars["Id"]

    beacon, _ := GetBeacon(instance)
    delete(beacon.Users, userId)
    err := UpdateBeacon(instance, beacon)

    if err != nil {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, err)
    } else {
        w.WriteHeader(http.StatusOK)
    }
}

func handleUpdateBeaconAddress(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)

    instance := vars["Instance"]
    address := vars["Address"]

    beacon, _ := GetBeacon(instance)
    beacon.Address = address
    err := UpdateBeacon(instance, beacon)

    if err != nil {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, err)
    } else {
        w.WriteHeader(http.StatusOK)
    }
}

func handleUpdateBeaconToken(w http.ResponseWriter, r *http.Request) {
    instance := mux.Vars(r)["Instance"]

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, err)
    }

    var token string

    err = json.Unmarshal(reqBody, &token)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, err)
        return
    }

    beacon, _ := GetBeacon(instance)
    beacon.Token = token
    err = UpdateBeacon(instance, beacon)

    if err != nil {
        w.WriteHeader(http.StatusInternalServerError) 
        fmt.Fprint(w, err)
    } else {
        w.WriteHeader(http.StatusOK)
    }
}

