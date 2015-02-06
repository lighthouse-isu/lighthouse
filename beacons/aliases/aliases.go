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

package aliases

import (
"fmt"
    "os"
    "io/ioutil"
    "net/http"

    "encoding/json"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

var aliases *databases.Table

func getDBSingleton() *databases.Table {
    if aliases == nil {
        panic("Aliases database not initialized")
    }
    return aliases
}

func Init() {
    if aliases == nil {
        aliases = databases.NewTable(postgres.Connection(), "aliases")
    }
}

func AddAlias(alias, value string) error {
    return getDBSingleton().Insert(alias, value)
}

func UpdateAlias(alias, value string) error {
    return getDBSingleton().Update(alias, value)
}

func GetAlias(alias string) (value string, err error) {
    err = getDBSingleton().SelectRow(alias, &value)
    return
}

func LoadAliases() map[string]string {
    var fileName string
    if _, err := os.Stat("/config/aliases.json"); os.IsNotExist(err) {
        fileName = "./config/aliases.json"
    } else {
        fileName = "/config/aliases.json"
    }
    configFile, _ := ioutil.ReadFile(fileName)

    var data []struct {
        Alias   string
        Value   string
    }

    json.Unmarshal(configFile, &data)

    configAliases := make(map[string]string)
    for _, item := range data {
        configAliases[item.Alias] = item.Value
    }

    return configAliases
}

func Handle(r *mux.Router) {
    aliases := LoadAliases()

    for alias, value := range aliases {
        AddAlias(alias, value)
    }

    r.HandleFunc("/{Host}/{Alias}", func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        host := vars["Host"]
        alias := vars["Alias"]

        fmt.Println(host, alias)

        err := UpdateAlias(alias, host)

        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
        } else {
            w.WriteHeader(http.StatusOK)
        }

    }).Methods("PUT")
}