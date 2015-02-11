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

func getDBSingleton() databases.TableInterface {
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

func createDatabaseEntryFor(beacon beaconData) map[string]interface{} {
    usersJson, _ := json.Marshal(beacon.Users)
    
    return map[string]interface{}{
        "InstanceAddress" : beacon.InstanceAddress,
        "BeaconAddress" : beacon.BeaconAddress,
        "Token" : beacon.Token,
        "Users" : string(usersJson),
    }
}

func addBeacon(beacon beaconData) error {
    entry := createDatabaseEntryFor(beacon)
    return getDBSingleton().InsertSchema(entry)
}

func updateBeaconField(field string, val interface{}, instance string) error {
    to := databases.Filter{field : val}
    where := databases.Filter{"InstanceAddress": instance}

    return getDBSingleton().UpdateSchema(to, where)
}

func getBeaconData(instance string) (beaconData, error) {
    var beacon beaconData
    where := databases.Filter{"InstanceAddress" : instance}

    err := getDBSingleton().SelectRowSchema(nil, where, &beacon)

    if err != nil {
        return beaconData{}, err
    }
   
    return beacon, nil
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

    beacon, err := getBeaconData(instance)

    if err == nil {
        if beacon.Users == nil { // JSON gives nil on empty map
            beacon.Users = make(userMap)
        }

        beacon.Users[userId] = true
        newUsers, _ := json.Marshal(beacon.Users)
        err = updateBeaconField("Users", newUsers, instance)
    }

    writeResponse(err, w)
}

func handleRemoveUserFromBeacon(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)

    instance := getInstanceAlias(vars["Instance"])
    userId := vars["Id"]

    beacon, err := getBeaconData(instance)

    if err == nil {
        delete(beacon.Users, userId)
        var newUsers []byte

        if len(beacon.Users) > 0 {
            newUsers, _ = json.Marshal(beacon.Users)
        } else {
            newUsers = []byte("{}")
        }

        err = updateBeaconField("Users", newUsers, instance)
    }

    writeResponse(err, w)
}

func handleUpdateBeaconAddress(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)

    instance := getInstanceAlias(vars["Instance"])
    address := vars["Address"]

    err := updateBeaconField("BeaconAddress", address, instance)

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

    err = updateBeaconField("Token", token, instance)

    writeResponse(err, w)
}

func handleCreate(r *http.Request) (int, error) {
    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return http.StatusInternalServerError, err
    }

    var beaconInfo struct {
        Address string
        Token string
        Users []string
    }

    err = json.Unmarshal(reqBody, &beaconInfo)
    if err != nil {
        return http.StatusInternalServerError, err
    }

    beacon := beaconData{"", beaconInfo.Address, beaconInfo.Token, make(userMap)}
    for _, user := range beaconInfo.Users {
        beacon.Users[user] = true
    }

    vmsTarget := fmt.Sprintf("http://%s/vms", beacon.BeaconAddress)

    req, err := http.NewRequest("GET", vmsTarget, nil)
    if err != nil {
        return http.StatusInternalServerError, err
    }

    // Assuming user has permission to access token since they provided it
    req.Header.Set(HEADER_TOKEN_KEY, beaconInfo.Token)

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
        beacon.InstanceAddress = fmt.Sprintf("%s:%s/%s", vm.Address, vm.Port, vm.Version)
        err = addBeacon(beacon)
    }

    return http.StatusOK, nil
}