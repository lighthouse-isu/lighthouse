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
    "database/sql"

    "encoding/json"
)

type Aliases struct {
    db *sql.DB
}

var aliases *Aliases = nil

type Alias struct {
    Alias string
    Value string
}

func connect() *Aliases {
    host := os.Getenv("POSTGRES_PORT_5432_TCP_ADDR")
    var postgresOptions string

    if host == "" { // if running locally
        postgresOptions = "sslmode=disable"
    } else { // if running in docker
        postgresOptions = fmt.Sprintf(
            "host=%s sslmode=disable user=postgres", host)
    }

    postgres, err := sql.Open("postgres", postgresOptions)

    if err != nil {
        fmt.Println(err.Error())
    }

    return &Aliases{postgres}
}

func (this *Aliases) init() *Aliases {
    this.db.Exec("CREATE TABLE aliases (alias text, value text)")
    return this
}

func (this *Aliases) drop() *Aliases {
    this.db.Exec("DROP TABLE aliases")
    return this
}

func Setup() {
    if aliases == nil {
        aliases = connect().drop().init()

        for _, alias := range LoadAliases() {
            AddAlias(alias.Alias, alias.Value)
        }
    }
}

func AddAlias(alias, value string) {
    Setup()

    aliases.db.Exec("INSERT INTO aliases (alias, value) VALUES (($1), ($2))",
        alias, value)
}

func UpdateAlias(alias, value string) {
    Setup()

    aliases.db.Exec("UPDATE aliases SET value = ($1) WHERE alias = ($2)",
        value, alias)
}

func GetAlias(alias string) *Alias {
    Setup()

    row := aliases.db.QueryRow(
        "SELECT alias, value FROM aliases WHERE alias = ($1)", alias)

    found := &Alias{}
    err := row.Scan(&found.Alias, &found.Value)

    if err != nil {
        fmt.Println(err.Error())
        return nil
    }

    return found
}

func LoadAliases() []Alias {
    var fileName string
    if _, err := os.Stat("/config/aliases.json"); os.IsNotExist(err) {
        fileName = "./config/aliases.json"
    } else {
        fileName = "/config/aliases.json"
    }
    configFile, _ := ioutil.ReadFile(fileName)

    var configAliases []Alias
    json.Unmarshal(configFile, &configAliases)

    return configAliases
}
