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
    "errors"
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

var (
    TokenPermissionError = errors.New("beacons: user not permitted to access token")
)

var beacons databases.TableInterface

var schema = databases.Schema {
    "InstanceAddress" : "text UNIQUE PRIMARY KEY",
    "BeaconAddress" : "text",
    "Token" : "text",
    "Users" : "json",
}

type beaconData struct {
    InstanceAddress string
    BeaconAddress string
    Token string
    Users userMap
}

type userMap map[string]interface{}

func Init() {
    if beacons == nil {
        beacons = databases.NewSchemaTable(postgres.Connection(), "beacons", schema)
    }
}

func GetBeaconAddress(instance string) (string, error) {
    var beacon beaconData
    where := databases.Filter{"InstanceAddress" : instance}
    columns := []string{"BeaconAddress"}

    err := getDBSingleton().SelectRowSchema(columns, where, &beacon)

    if err != nil {
        return "", err
    }
   
    return beacon.BeaconAddress, nil
}

func GetBeaconToken(instance, user string) (string, error) {
    var beacon beaconData
    where := databases.Filter{"InstanceAddress" : instance}
    columns := []string{"Token", "Users"}

    err := getDBSingleton().SelectRowSchema(columns, where, &beacon)

    if err != nil {
        return "", err
    }

    // Database gives nil on empty map
    if beacon.Users == nil {
        return "", TokenPermissionError
    }

    _, ok := beacon.Users[user]

    if !ok {
        return "", TokenPermissionError
    }
   
    return beacon.Token, nil
}

func LoadBeacons() []beaconData {
    var fileName string
    if _, err := os.Stat("/config/beacon_permissions.json"); os.IsNotExist(err) {
        fileName = "./config/beacon_permissions.json"
    } else {
        fileName = "/config/beacon_permissions.json"
    }

    configFile, _ := ioutil.ReadFile(fileName)

    var beacons []beaconData
    json.Unmarshal(configFile, &beacons)

    return beacons
}

func Handle(r *mux.Router) {
    beacons := LoadBeacons()

    for _, beacon := range beacons {
        addBeacon(beacon)
    }

    r.HandleFunc("/user/{Instance}/{Id}", handleAddUserToBeacon).Methods("PUT")

    r.HandleFunc("/user/{Instance}/{Id}", handleRemoveUserFromBeacon).Methods("DELETE")

    r.HandleFunc("/address/{Instance}/{Address}", handleUpdateBeaconAddress).Methods("PUT")

    r.HandleFunc("/token/{Instance}", handleUpdateBeaconToken).Methods("PUT")

    r.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {

        code, err := handleCreate(r)
        w.WriteHeader(code) 
        if err != nil {
            fmt.Fprint(w, err)
        }

    }).Methods("POST")

    r.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {

        beacons, err := getBeaconsList()
        var output []byte

        if err == nil {
            output, err = json.Marshal(beacons)
        } 

        if err != nil {
            writeResponse(err, w) 
        } else {
            fmt.Fprint(w, string(output))
        }

    }).Methods("GET")

    r.HandleFunc("/list/{Endpoint:.*}", func(w http.ResponseWriter, r *http.Request) {
        // TODO
    }).Methods("GET")
}