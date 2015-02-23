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
    "io/ioutil"
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
    NotEnoughParametersError = errors.New("beacons: not enough parameters given")
)

var beacons databases.TableInterface
var instances databases.TableInterface

var beaconSchema = databases.Schema {
    "Address" : "text UNIQUE PRIMARY KEY",
    "Token" : "text",
    "Users" : "json",
}

var instanceSchema = databases.Schema {
    "InstanceAddress" : "text UNIQUE PRIMARY KEY",
    "Name" : "text",
    "CanAccessDocker" : "boolean",
    "BeaconAddress" : "text",
}

type beaconData struct {
    Address string
    Token string
    Users userMap
}

type instanceData struct {
    InstanceAddress string
    Name string
    CanAccessDocker bool
    BeaconAddress string
}

type userMap map[string]interface{}

func Init() {
    if beacons == nil {
        beacons = databases.NewSchemaTable(postgres.Connection(), "beacons", beaconSchema)
    }

    if instances == nil {
        instances = databases.NewSchemaTable(postgres.Connection(), "instances", instanceSchema)
    }
}

func GetBeaconAddress(instance string) (string, error) {
    var beacon instanceData
    where := databases.Filter{"InstanceAddress" : instance}
    columns := []string{"BeaconAddress"}

    err := instances.SelectRowSchema(columns, where, &beacon)

    if err != nil {
        return "", err
    }
   
    return beacon.BeaconAddress, nil
}

func GetBeaconToken(beacon, user string) (string, error) {
    var data beaconData
    where := databases.Filter{"Address" : beacon}
    columns := []string{"Token", "Users"}

    err := beacons.SelectRowSchema(columns, where, &data)

    if err != nil {
        return "", err
    }

    // Database gives nil on empty map
    if data.Users == nil {
        return "", TokenPermissionError
    }

    _, ok := data.Users[user]

    if !ok {
        return "", TokenPermissionError
    }
   
    return data.Token, nil
}

func LoadBeacons() {
    var fileName string
    var err error
    if _, err = os.Stat("./config/beacon_permissions.json.dev"); err == nil {
        fileName = "./config/beacon_permissions.json.dev"
    } else if _, err = os.Stat("./config/beacon_permissions.json"); err == nil {
        fileName = "./config/beacon_permissions.json"
    } else {
        fileName = "/config/beacon_permissions.json"
    }

    configFile, _ := ioutil.ReadFile(fileName)

    var entries struct {
        Beacons []beaconData
        Instances []instanceData
    }

    json.Unmarshal(configFile, &entries)

    for _, beacon := range entries.Beacons {
        addBeacon(beacon)
    }

    for _, inst := range entries.Instances {
        addInstance(inst)
    }
}

func Handle(r *mux.Router) {
    LoadBeacons()

    r.HandleFunc("/user/{Endpoint:.*}", handleAddUserToBeacon).Methods("PUT")

    r.HandleFunc("/user/{Endpoint:.*}", handleRemoveUserFromBeacon).Methods("DELETE")

    r.HandleFunc("/token/{Endpoint:.*}", handleUpdateBeaconToken).Methods("PUT")

    r.HandleFunc("/create", handleBeaconCreate).Methods("POST")

    r.HandleFunc("/list", handleListBeacons).Methods("GET")

    r.HandleFunc("/list/{Beacon:.*}", handleListInstances).Methods("GET")

    r.HandleFunc("/refresh/{Beacon:.*}", handleRefreshBeacon).Methods("PUT")
}