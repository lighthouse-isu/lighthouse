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
    "errors"
    "net/http"
    "io/ioutil"
    "encoding/json"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

const (
    HEADER_TOKEN_KEY = "Token"
)

var (
    TokenPermissionError = errors.New("beacons: user not permitted to access token")
    NotEnoughParametersError = errors.New("beacons: not enough parameters given")
)

var beacons databases.TableInterface

var schema = databases.Schema {
    "InstanceAddress" : "text UNIQUE PRIMARY KEY",
    "BeaconAddress" : "text",
    "Token" : "text",
}

type beaconData struct {
    InstanceAddress string
    BeaconAddress string
    Token string
}

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

func TryGetBeaconToken(instance string, r *http.Request) (string, error) {
    var beacon beaconData
    where := databases.Filter{"InstanceAddress" : instance}
    columns := []string{"BeaconAddress", "Token"}

    err := getDBSingleton().SelectRowSchema(columns, where, &beacon)

    if err != nil {
        return "", err
    }

    if !auth.GetCurrentUser(r).CanAccessBeacon(beacon.BeaconAddress) {
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
        addInstance(beacon)
    }

    r.HandleFunc("/token/{Endpoint:.*}", handleUpdateBeaconToken).Methods("PUT")

    r.HandleFunc("/create", handleBeaconCreate).Methods("POST")

    r.HandleFunc("/list", handleListBeacon).Methods("GET")

    r.HandleFunc("/list/{Beacon:.*}", handleListInstances).Methods("GET")
}