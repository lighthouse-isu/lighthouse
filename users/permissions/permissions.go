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

package permissions

import (
    "os"
    "fmt"
    "net/http"
    "io/ioutil"

    "encoding/json"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
    "github.com/lighthouse/lighthouse/session"
)

var permissions *databases.Table

type Permission struct {
    Providers map[string]bool
}

func getDBSingleton() *databases.Table {
    if permissions == nil {
        panic("Permissions database not initialized")
    }
    return permissions
}

func Init() {
    if permissions == nil {
        permissions = databases.NewTable(postgres.Connection(), "permissions")
    }
}

func AddPermission(email string, permission Permission) error {
    return getDBSingleton().Insert(email, permission)
}

func UpdatePermission(email string, permission Permission) error {
    return getDBSingleton().Update(email, permission)
}

func GetPermissions(email string) (perm Permission, err error) {
    err = getDBSingleton().SelectRow(email, &perm)
    return
}

func LoadPermissions() map[string]Permission {
    var fileName string
    if _, err := os.Stat("/config/permissions.json"); os.IsNotExist(err) {
        fileName = "./config/permissions.json"
    } else {
        fileName = "/config/permissions.json"
    }

    configFile, _ := ioutil.ReadFile(fileName)

    var data []struct {
        Email   string
        Providers []string
    }

    json.Unmarshal(configFile, &data)

    perms := make(map[string]Permission)
    for _, item := range data {
        perm := make(map[string]bool)
        for _, provider := range item.Providers {
            perm[provider] = true
        }

        perms[item.Email] = Permission{perm}
    }

    return perms
}

func Handle(router *mux.Router) {
    perms := LoadPermissions()

    for email, perm := range perms {
        AddPermission(email, perm)
    }

    router.HandleFunc("/vms", func(w http.ResponseWriter, r *http.Request) {
        email := session.GetValueOrDefault(r, "auth", "email", "").(string)
        perm, err := GetPermissions(email)

        var response []byte = nil
        if err == nil {
            response, _ = json.Marshal(perm.Providers)
        } else {
            response = []byte("") // User unknown
        }

        fmt.Fprintf(w, "%s", response)
    }).Methods("GET")
}
