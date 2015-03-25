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
    "io/ioutil"
    "net/http"

    "encoding/json"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/handlers"
    "github.com/lighthouse/lighthouse/databases"
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

func Init(reload bool) {
    if aliases == nil {
        aliases = databases.NewSchemaTable(nil, "aliases", schema)
    }

    if reload {
        aliases.Reload()
        LoadAliases()
    }
}

func AddAlias(alias, address string) error {
    entry := map[string]interface{}{
        "Alias" : alias,
        "Address" : address,
    }

    _, err := aliases.InsertSchema(entry, "")

    return err
}

func UpdateAlias(alias, address string) error {
    to := databases.Filter{"Alias": alias}
    where := databases.Filter{"Address" : address}

    return aliases.UpdateSchema(to, where)
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

    err := aliases.SelectRowSchema(cols, where, &val)
    
    if err != nil {
        return "", err
    }

    return val.Address, nil
}

func GetAliasOf(address string) (string, error) {
    cols := []string{"Alias"}
    where := databases.Filter{"Address": address}
    
    var val Alias

    err := aliases.SelectRowSchema(cols, where, &val)
    
    if err != nil {
        return "", err
    }

    return val.Alias, nil
}

func LoadAliases() {
    var fileName string
    if _, err := os.Stat("./config/aliases.json.dev"); !os.IsNotExist(err)  {
        fileName = "./config/aliases.json.dev"
    } else if _, err := os.Stat("./config/aliases.json"); !os.IsNotExist(err)  {
        fileName = "./config/aliases.json"
    } else {
        fileName = "/config/aliases.json"
    }

    configFile, _ := ioutil.ReadFile(fileName)

    var data []Alias

    json.Unmarshal(configFile, &data)

    for _, item := range data {
        AddAlias(item.Alias, item.Address)
    }
}

func Handle(r *mux.Router) {
    r.HandleFunc("/{Address:.*}", handleUpdateAlias).Methods("PUT")
}

func handleUpdateAlias(w http.ResponseWriter, r *http.Request) {
    var code int = http.StatusOK
    var err error = nil
    defer func(){
        if err == nil {
            w.WriteHeader(code)
        } else {
            handlers.WriteError(w, code, "aliases", err.Error())
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
        code = http.StatusBadRequest
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
