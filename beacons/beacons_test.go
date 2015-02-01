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

    "strings"
    "encoding/json"
    "database/sql"

    "github.com/stretchr/testify/assert"

    "github.com/lighthouse/lighthouse/databases"
)

var mockTable map[string]Beacon

func makeTestingDatabase(t *testing.T) *databases.MockDatabase {
    mockTable = make(map[string]Beacon)
    db := &databases.MockDatabase{}

    db.MockExec = func(s string, i ...interface{}) (r sql.Result, e error) {

        args := i[0].([]interface{})

        if strings.Contains(s, "INSERT") || strings.Contains(s, "UPDATE") {
            var beacon Beacon
            json.Unmarshal([]byte(args[1].(string)), &beacon)
            mockTable[args[0].(string)] = beacon
        }

        return
    }

    db.MockQueryRow = func(s string, i ...interface{}) (r *sql.Row) {      
        return
    }

    return db
}

func Test_AddBeacon(t *testing.T) {
    SetupTestingTable(makeTestingDatabase(t))
    defer TeardownTestingTable()

    testBeacon := Beacon{
        "ADDRESS_PASS", "TOKEN_PASS", map[string]bool{"USER_PASS":true},
    }

    AddBeacon("TEST_INST", testBeacon)

    _, found := mockTable["TEST_INST"]
    assert.True(t, found, 
        "AddBeacon should create entry for given instance")

    assert.Equal(t, testBeacon, mockTable["TEST_INST"], 
        "AddBeacon should insert the given Beacon")
}

func Test_UpdateBeacon(t *testing.T) {
    SetupTestingTable(makeTestingDatabase(t))
    defer TeardownTestingTable()

    testBeacon := Beacon{
        "ADDRESS_FAIL", "TOKEN_FAIL", map[string]bool{"USER_FAIL":true},
    }

    mockTable["TEST_INST"] = testBeacon

    updateBeacon := Beacon{
        "ADDRESS_PASS", "TOKEN_PASS", map[string]bool{"USER_PASS":true},
    }

    UpdateBeacon("TEST_INST", updateBeacon)

    _, found := mockTable["TEST_INST"]
    assert.True(t, found, 
        "UpdateBeacon should create entry for given instance")

    assert.Equal(t, updateBeacon, mockTable["TEST_INST"], 
        "UpdateBeacon should insert the given Beacon")
}

func Test_GetBeacon_NotFound(t *testing.T) {
    SetupTestingTable(makeTestingDatabase(t))
    defer TeardownTestingTable()

    res, err := GetBeacon("TEST_INST")

    assert.NotNil(t, err, 
        "GetBeacon should forward SQL errors")

    assert.Equal(t, "", res.Address, 
        "GetBeacon should return an empty Address on unknown instance")

    assert.Equal(t, "", res.Token, 
        "GetBeacon should return an empty Token on unknown instance")

    assert.NotNil(t, res.Users, 
        "GetBeacon should return an empty Users map on unknown instance")
}