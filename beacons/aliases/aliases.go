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
    "os"
    "fmt"
    "io/ioutil"
    "net/http"

    "encoding/json"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

var aliases databases.TableInterface

var schema = databases.Schema{
    "Alias" : "text",
    "Address" : "text UNIQUE PRIMARY KEY",
}

type Alias struct {
    Alias string
    Address string
}

func getDBSingleton() databases.TableInterface {
    if aliases == nil {
        panic("Aliases database not initialized")
    }
    return aliases
}

func Init() {
    if aliases == nil {
        aliases = databases.NewSchemaTable(postgres.Connection(), "aliases", schema)
    }
}

func AddAlias(alias, address string) error {
    entry := map[string]interface{}{
        "Alias" : alias,
        "Address" : address,
    }

    return getDBSingleton().InsertSchema(entry)
}

func UpdateAlias(alias, address string) error {
    to := databases.Filter{"Alias": alias}
    where := databases.Filter{"Address" : address}

    return getDBSingleton().UpdateSchema(to, where)
}

func SetAlias(alias, address string) error {
    err := UpdateAlias(alias, address)

    if err == databases.NoUpdateError {
        err = AddAlias(alias, address)
    }

    return err
}

func GetAddressOf(alias string) (string, error) {
    cols := []string{"Address"}
    where := databases.Filter{"Alias": alias}
    
    var val Alias

    err := getDBSingleton().SelectRowSchema(cols, where, &val)
    
    if err != nil {
        return "", err
    }

    return val.Address, nil
}

func GetAliasOf(address string) (string, error) {
    cols := []string{"Alias"}
    where := databases.Filter{"Address": address}
    
    var val Alias

    err := getDBSingleton().SelectRowSchema(cols, where, &val)
    
    if err != nil {
        return "", err
    }

    return val.Alias, nil
}

func LoadAliases() map[string]string {
    var fileName string
    if _, err := os.Stat("/config/aliases.json"); os.IsNotExist(err) {
        fileName = "./config/aliases.json"
    } else {
        fileName = "/config/aliases.json"
    }
    configFile, _ := ioutil.ReadFile(fileName)

    var data []Alias

    json.Unmarshal(configFile, &data)

    configAliases := make(map[string]string)
    for _, item := range data {
        configAliases[item.Alias] = item.Address
    }

    return configAliases
}

func Handle(r *mux.Router) {
    aliasConfig := LoadAliases()

    for alias, address := range aliasConfig {
        AddAlias(alias, address)
    }

    r.HandleFunc("/{Address:.*}", handleUpdateAlias).Methods("PUT")
}

func handleUpdateAlias(w http.ResponseWriter, r *http.Request) {
    var code int = http.StatusOK
    var err error = nil
    defer func(){
        w.WriteHeader(code)
        if err != nil {
            fmt.Fprint(w, err)
        }
    }()

    address := mux.Vars(r)["Address"]

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        code = http.StatusInternalServerError
        return
    }

    var alias string

    err = json.Unmarshal(reqBody, &alias)
    if err != nil {
        code = http.StatusInternalServerError
        return
    }

    if alias == "" {
        code = http.StatusBadRequest
        return
    }

    _, res := GetAliasOf(address)
    if res == databases.NoRowsError {
        err = AddAlias(alias, address)
    } else {
        err = UpdateAlias(alias, address)
    }

    if err != nil {
        code = http.StatusInternalServerError
        return
    }

    return
}