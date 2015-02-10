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

    beaconStructs "github.com/lighthouse/beacon/structs"

    "github.com/lighthouse/lighthouse/beacons/aliases"

    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

const (
    HEADER_TOKEN_KEY = "Token"
)

var beacons *databases.Table

var schema = databases.Schema {
    "InstanceAddress" : "text UNIQUE PRIMARY KEY",
    "Address" : "text",
    "Token" : "text",
    "Users" : "json",
}

type Beacon struct {
    Address string
    Token string
    Users map[string]bool
}

func getDBSingleton() *databases.Table {
    if beacons == nil {
        panic("Beacons database not initialized")
    }
    return beacons
}

func Init() {
    if beacons == nil {
        beacons = databases.NewSchemaTable(postgres.Connection(), "beacons", schema)
    }
}

func createDatabaseEntryFor(instance string, beacon Beacon) databases.Filter {
    usersJson, _ := json.Marshal(beacon.Users)
    
    return databases.Filter{
        "InstanceAddress" : instance,
        "Address" : beacon.Address,
        "Token" : beacon.Token,
        "Users" : string(usersJson),
    }
}

func AddBeacon(instance string, beacon Beacon) error {
    entry := createDatabaseEntryFor(instance, beacon)
    return getDBSingleton().InsertSchema(entry)
}

func UpdateBeacon(instance string, beacon Beacon) error {
    to := createDatabaseEntryFor(instance, beacon)
    where := databases.Filter{"InstanceAddress": instance}

    return getDBSingleton().UpdateSchema(to, where)
}

func GetBeacon(instance string) (Beacon, error) {
    var beacon Beacon
    where := databases.Filter{"InstanceAddress" : instance}
    columns := []string{"Address" , "Token", "Users"}

    err := getDBSingleton().SelectRowSchema(columns, where, &beacon)

    if err != nil {
        fmt.Println(err)
        return Beacon{"", "", make(map[string]bool)}, err
    }
   
    return beacon, nil
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
}

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
    vars := mux.Vars(r)

    instance := getInstanceAlias(vars["Instance"])
    userId := vars["Id"]

    beacon, err := GetBeacon(instance)

    if err == nil {
        beacon.Users[userId] = true
        err = UpdateBeacon(instance, beacon)
    }

    writeResponse(err, w)
}

func handleRemoveUserFromBeacon(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)

    instance := getInstanceAlias(vars["Instance"])
    userId := vars["Id"]

    beacon, err := GetBeacon(instance)
    if err == nil {
        delete(beacon.Users, userId)
        err = UpdateBeacon(instance, beacon)
    }

    writeResponse(err, w)
}

func handleUpdateBeaconAddress(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)

    instance := getInstanceAlias(vars["Instance"])
    address := vars["Address"]

    beacon, err := GetBeacon(instance)
    if err == nil {
        beacon.Address = address
        err = UpdateBeacon(instance, beacon)
    }

    writeResponse(err, w)
}

func handleUpdateBeaconToken(w http.ResponseWriter, r *http.Request) {
    instance := getInstanceAlias(mux.Vars(r)["Instance"])

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

    beacon, _ := GetBeacon(instance)
    if err == nil {
        beacon.Token = token
        err = UpdateBeacon(instance, beacon)
    }

    writeResponse(err, w)
}

func handleCreate(r *http.Request) (int, error) {
    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return http.StatusInternalServerError, err
    }

    var beaconData struct {
        Address string
        Token string
        Users []string
    }

    err = json.Unmarshal(reqBody, &beaconData)
    if err != nil {
        return http.StatusInternalServerError, err
    }

    beacon := Beacon{beaconData.Address, beaconData.Token, make(map[string]bool)}
    for _, user := range beaconData.Users {
        beacon.Users[user] = true
    }

    vmsTarget := fmt.Sprintf("http://%s/vms", beacon.Address)

    req, err := http.NewRequest("GET", vmsTarget, nil)
    if err != nil {
        return http.StatusInternalServerError, err
    }

    // Assuming user has permission to access token since they provided it
    req.Header.Set(HEADER_TOKEN_KEY, beaconData.Token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return http.StatusInternalServerError, err
    }

    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return resp.StatusCode, errors.New("beacon error")
    }

    vmsBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return http.StatusInternalServerError, err
    }

    var vms []beaconStructs.VM

    err = json.Unmarshal(vmsBody, &vms)
    if err != nil {
        return http.StatusInternalServerError, err
    }

    for _, vm := range vms {
        address := fmt.Sprintf("%s:%s/%s", vm.Address, vm.Port, vm.Version)
        err = AddBeacon(address, beacon)
    }

    return http.StatusOK, nil
}