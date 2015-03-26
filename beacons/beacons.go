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

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/databases"
)

const (
    HEADER_TOKEN_KEY = "Token"
    INSTANCE_ALIAS_DELIM = "."
)

var (
    TokenPermissionError = errors.New("beacons: user not permitted to access token")
    NotEnoughParametersError = errors.New("beacons: not enough or invalid parameters given")
    DuplicateBeaconError = errors.New("beacons: tried to add an beacon which already exists")
)

var beacons databases.TableInterface
var instances databases.TableInterface

var beaconSchema = databases.Schema {
    "Address" : "text UNIQUE PRIMARY KEY",
    "Token" : "text",
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
}

type instanceData struct {
    InstanceAddress string
    Name string
    CanAccessDocker bool
    BeaconAddress string
}

func Init(reload bool) {
    if beacons == nil {
        beacons = databases.NewTable(nil, "beacons", beaconSchema)
    }

    if instances == nil {
        instances = databases.NewTable(nil, "instances", instanceSchema)
    }

    if reload {
        beacons.Reload()
        instances.Reload()
        LoadBeacons()
    }
}

func GetBeaconAddress(instance string) (string, error) {
    var beacon instanceData
    where := databases.Filter{"InstanceAddress" : instance}
    columns := []string{"BeaconAddress"}

    err := instances.SelectRow(columns, where, &beacon)

    if err != nil {
        return "", err
    }
   
    return beacon.BeaconAddress, nil
}

func TryGetBeaconToken(beacon string, user *auth.User) (string, error) {
    if !user.CanAccessBeacon(beacon) {
        return "", TokenPermissionError
    }

    var data beaconData
    where := databases.Filter{"Address" : beacon}
    columns := []string{"Token"}

    err := beacons.SelectRow(columns, where, &data)

    if err != nil {
        return "", err
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
    r.HandleFunc("/token/{Endpoint:.*}", handleUpdateBeaconToken).Methods("PUT")

    r.HandleFunc("/create", handleBeaconCreate).Methods("POST")

    r.HandleFunc("/list", handleListBeacons).Methods("GET")

    r.HandleFunc("/list/{Beacon:.*}", handleListInstances).Methods("GET")

    r.HandleFunc("/refresh/{Beacon:.*}", handleRefreshBeacon).Methods("PUT")
}