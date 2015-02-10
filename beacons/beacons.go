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

    "github.com/zenazn/goji/web"

    beaconStructs "github.com/lighthouse/beacon/structs"

    "github.com/lighthouse/lighthouse/beacons/aliases"

    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

const (
    HEADER_TOKEN_KEY = "Token"
)

var beacons *databases.Table

var (
    NotEnoughParametersError = errors.New("beacons: not enough parameters in endpoint")
)

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

func init() {
    if beacons == nil {
        beacons = databases.NewTable(postgres.Connection(), "beacons")
    }
}

func AddBeacon(instance string, beacon Beacon) error {
    return getDBSingleton().Insert(instance, beacon)
}

func UpdateBeacon(instance string, beacon Beacon) error {
    return getDBSingleton().Update(instance, beacon)
}

func GetBeacon(instance string) (Beacon, error) {
    var beacon Beacon
    err := getDBSingleton().SelectRow(instance, &beacon)

    if err != nil {
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

func Handle(m *web.Mux) {
    beacons := LoadBeacons()

    for instance, beacon := range beacons {
        AddBeacon(instance, beacon)
    }

    m.Put("/user/*", handleAddUserToBeacon)

    m.Delete("/user/*", handleRemoveUserFromBeacon)

    m.Put("/address/*", handleUpdateBeaconAddress)

    m.Put("/token/*", handleUpdateBeaconToken)

    m.Post("/create", func(c web.C, w http.ResponseWriter, r *http.Request) {
        code, err := handleCreate(r)
        w.WriteHeader(code) 
        if err != nil {
            fmt.Fprint(w, err)
        }
    })
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
        case databases.KeyNotFoundError, databases.NoUpdateError, 
            databases.EmptyKeyError, NotEnoughParametersError:
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

func handleAddUserToBeacon(c web.C, w http.ResponseWriter, r *http.Request) {
    if _, ok := c.Env["3"]; !ok {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    instance := getInstanceAlias(c.Env["2"].(string))
    userId := c.Env["3"].(string)

    beacon, err := GetBeacon(instance)
    if err == nil {
        beacon.Users[userId] = true
        err = UpdateBeacon(instance, beacon)
    }

    writeResponse(err, w)
}

func handleRemoveUserFromBeacon(c web.C, w http.ResponseWriter, r *http.Request) {
    if _, ok := c.Env["3"]; !ok {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    instance := getInstanceAlias(c.Env["2"].(string))
    userId := c.Env["3"].(string)

    beacon, err := GetBeacon(instance)
    if err == nil {
        delete(beacon.Users, userId)
        err = UpdateBeacon(instance, beacon)
    }

    writeResponse(err, w)
}

func handleUpdateBeaconAddress(c web.C, w http.ResponseWriter, r *http.Request) {
    if _, ok := c.Env["3"]; !ok {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    instance := getInstanceAlias(c.Env["2"].(string))
    address := c.Env["3"].(string)

    beacon, err := GetBeacon(instance)
    if err == nil {
        beacon.Address = address
        err = UpdateBeacon(instance, beacon)
    }

    writeResponse(err, w)
}

func handleUpdateBeaconToken(c web.C, w http.ResponseWriter, r *http.Request) {
    if _, ok := c.Env["2"]; !ok {
        writeResponse(NotEnoughParametersError, w)
        return
    }

    instance := getInstanceAlias(c.Env["2"].(string))

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

    if token == "" {
        writeResponse(NotEnoughParametersError, w)
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
        AddBeacon(address, beacon)
    }

    return http.StatusOK, nil
}