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
    "testing"
    "reflect"
    "encoding/json"

    "github.com/stretchr/testify/assert"

    "github.com/lighthouse/lighthouse/databases"
)

var database map[string]interface{}

func makeTestingDatabase(t *testing.T) *databases.MockTable {
    table := &databases.MockTable{}

    database = map[string]interface{}{
        "InstanceAddress" : "",
        "BeaconAddress" : "",
        "Token" : "",
        "Users" : nil,
    }

    table.MockInsertSchema = func(values map[string]interface{})(error) {
        for k, v := range values {
            database[k] = v
        }
        return nil
    }

    table.MockUpdateSchema = func(to, where map[string]interface{})(error) {
        for k, v := range to {
            database[k] = v
        }
        return nil
    }

    table.MockSelectRowSchema = func(cols []string, where databases.Filter, dest interface{})(error) {
        rv := reflect.ValueOf(dest).Elem()
        for _, col := range cols {
            rv.FieldByName(col).Set(reflect.ValueOf(database[col]))
        }
        return nil
    }

    return table
}

func Test_AddBeaconData(t *testing.T) {
    SetupTestingTable(makeTestingDatabase(t))
    defer TeardownTestingTable()

    users := userMap{"USER_PASS":true}
    userJSON, _ := json.Marshal(users)

    testBeaconData := beaconData{
        "INST_ADDR_PASS", "BEACON_ADDR_PASS", "TOKEN_PASS", users,
    }

    addBeacon(testBeaconData)

    assert.Equal(t, "INST_ADDR_PASS", database["InstanceAddress"],
        "AddBeacon should set InstanceAddress")

    assert.Equal(t, "BEACON_ADDR_PASS", database["BeaconAddress"],
        "AddBeacon should set InstanceAddress")

    assert.Equal(t, "TOKEN_PASS", database["Token"],
        "AddBeacon should set InstanceAddress")

    assert.Equal(t, string(userJSON), database["Users"],
        "AddBeacon should set InstanceAddress")
}

func Test_UpdateBeaconData(t *testing.T) {
    SetupTestingTable(makeTestingDatabase(t))
    defer TeardownTestingTable()

    /*
    testBeaconData := beaconData{
        "INST_ADDR", "BEACON_ADDR_FAIL", "TOKEN_FAIL", userMap{"USER_FAIL":true},
    }

    updateBeaconData := beaconData{
        "INST_ADDR", "BEACON_ADDR_PASS", "TOKEN_PASS", userMap{"USER_PASS":true},
    }

    updateBeaconField("BeaconAddress", "BEACON_ADDR_PASS", "INST_ADDR")
    */
}

func Test_GetBeaconData_NotFound(t *testing.T) {
    SetupTestingTable(makeTestingDatabase(t))
    defer TeardownTestingTable()

    //res, err := getBeaconData("TEST_INST")
}