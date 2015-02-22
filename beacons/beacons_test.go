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

    "fmt"

    "github.com/stretchr/testify/assert"

    "github.com/lighthouse/lighthouse/beacons/aliases"
    "github.com/lighthouse/lighthouse/databases"
)

func setup() (databases.TableInterface, func()) {
    SetupTestingTable()
    aliases.SetupTestingTable()

    return beacons, func() {
        TeardownTestingTable()
        aliases.TeardownTestingTable()
    }
}

func Test_AddBeaconData(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    users := userMap{"USER":true}

    testBeaconData := beaconData{
        "INST_ADDR", "BEACON_ADDR", "TOKEN", users,
    }

    addInstance(testBeaconData)

    var values beaconData
    table.SelectRowSchema(nil, nil, &values)

    assert.Equal(t, testBeaconData, values)
}

func Test_UpdateBeaconData(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    testBeaconData := map[string]interface{}{
        "InstanceAddress" : "INST_ADDR", 
        "BeaconAddress" : "BEACON_ADDR_FAIL", 
        "Token" : "TOKEN_FAIL", 
        "Users" : userMap{"USER_FAIL":true},
    }

    table.InsertSchema(testBeaconData)

    userPass := userMap{"USER_PASS":true}
    keyData := beaconData {
        "INST_ADDR_PASS", "BEACON_ADDR_PASS", "TOKEN_PASS", userPass,
    }

    var values beaconData

    updateBeaconField("BeaconAddress", "BEACON_ADDR_PASS", "INST_ADDR")
    updateBeaconField("Token", "TOKEN_PASS", "INST_ADDR")
    updateBeaconField("Users", userPass, "INST_ADDR")
    updateBeaconField("InstanceAddress", "INST_ADDR_PASS", "INST_ADDR")

    table.SelectRowSchema(nil, nil, &values)
    assert.Equal(t, keyData, values)
}

func Test_GetBeaconAddress_Found(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    users := userMap{"USER":true}

    testBeaconData := map[string]interface{}{
        "InstanceAddress" : "INST_ADDR", 
        "BeaconAddress" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : users,
    }

    table.InsertSchema(testBeaconData)

    res, err := GetBeaconAddress("INST_ADDR")

    assert.Nil(t, err, 
        "GetBeaconAddress should not return error beacon was found")

    assert.Equal(t, "BEACON_ADDR", res, 
        "GetBeaconAddress should give correct address")
}

func Test_GetBeaconAddress_NotFound(t *testing.T) {
    _, teardown := setup()
    defer teardown()

    res, err := GetBeaconAddress("BAD_ADDR")

    assert.NotNil(t, err, 
        "GetBeaconAddress should forward errors")

    assert.Equal(t, "", res, 
        "GetBeaconAddress should give empty string on error")
}

func Test_GetBeaconData_Found(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    users := userMap{"USER":true}

    testBeaconData := map[string]interface{}{
        "InstanceAddress" : "INST_ADDR", 
        "BeaconAddress" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : users,
    }

    table.InsertSchema(testBeaconData)

    res, err := getBeaconData("INST_ADDR")

    assert.Nil(t, err, "getBeaconData should not return error beacon was found")

    key := beaconData{"INST_ADDR", "BEACON_ADDR", "TOKEN", users}
    assert.Equal(t, key, res, 
        "getBeaconData should give correct beaconData")
}

func Test_GetBeaconData_NotFound(t *testing.T) {
    _, teardown := setup()
    defer teardown()

    res, err := getBeaconData("BAD_INST")

    assert.NotNil(t, err, "getBeaconData should forward errors")

    assert.Equal(t, beaconData{}, res, 
        "getBeaconData should give empty beaconData on error")
}

func Test_GetBeaconToken_NotFound(t *testing.T) {
    _, teardown := setup()
    defer teardown()

    res, err := GetBeaconToken("BAD_INST", "junk user")

    assert.NotNil(t, err, "GetBeaconToken should forward errors")

    assert.Equal(t, "", res, 
        "GetBeaconToken should give empty token on error")
}

func Test_GetBeaconToken_NotPermitted(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    testBeaconData := map[string]interface{}{
        "InstanceAddress" : "INST_ADDR", 
        "BeaconAddress" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{},
    }

    table.InsertSchema(testBeaconData)

    res, err := GetBeaconToken("INST_ADDR", "BAD_USER")

    assert.NotNil(t, err, 
        "GetBeaconToken should return error on bad permissions")

    assert.Equal(t, "", res, 
        "GetBeaconToken should give empty token on bad permissions")
}

func Test_GetBeaconToken_Valid(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    testBeaconData := map[string]interface{}{
        "InstanceAddress" : "INST_ADDR", 
        "BeaconAddress" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{"USER":true},
    }

    table.InsertSchema(testBeaconData)

    res, err := GetBeaconToken("INST_ADDR", "USER")

    assert.Nil(t, err, 
        "GetBeaconToken should return nil error on success")

    assert.Equal(t, "TOKEN", res, 
        "GetBeaconToken should give corrent token")
}

func Test_ListBeacons_ValidUser(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    keyList := make([]aliases.Alias, 0)

    for i := 1; i <= 2; i++ {
        beaconList, err := getBeaconsList("USER")

        assert.Nil(t, err, "getBeaconList returned an error")

        assert.Equal(t, keyList, beaconList, 
            "getBeaconList output differed from key")

        newBeacon := map[string]interface{} {
            "InstanceAddress" : "INST_ADDR", 
            "BeaconAddress" : fmt.Sprintf("BEACON_ADDR %d", i), 
            "Token" : "TOKEN", 
            "Users" : userMap{"USER":true},
        }

        keyPair := aliases.Alias{"", newBeacon["BeaconAddress"].(string)}

        keyList = append(keyList, keyPair)
        table.InsertSchema(newBeacon)
    }

    beaconList, err := getBeaconsList("USER")

    assert.Nil(t, err, "getBeaconList returned an error")
    assert.Equal(t, keyList, beaconList)
}

func Test_ListBeacons_BadUser(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    goodBeacon := map[string]interface{} {
        "InstanceAddress" : "INST_ADDR 1", 
        "BeaconAddress" : "BEACON_ADDR 1", 
        "Token" : "TOKEN", 
        "Users" : userMap{"GOOD_USER":true},
    }

    badBeacon := map[string]interface{} {
        "InstanceAddress" : "INST_ADDR 2", 
        "BeaconAddress" : "BEACON_ADDR 2", 
        "Token" : "TOKEN", 
        "Users" : userMap{"BAD_USER":true},
    }

    keyList := []aliases.Alias{aliases.Alias{"", "BEACON_ADDR 1"}}

    table.InsertSchema(goodBeacon)
    table.InsertSchema(badBeacon)

    beaconList, err := getBeaconsList("GOOD_USER")

    assert.Nil(t, err, "getBeaconList returned an error")
    assert.Equal(t, keyList, beaconList)
}

func Test_ListInstances_ValidUser(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    keyList := make([]string, 0)

    for i := 1; i <= 2; i++ {
        instanceList, err := getInstancesList("BEACON_ADDR", "USER")

        assert.Nil(t, err, "getBeaconList returned an error")

        assert.Equal(t, keyList, instanceList, 
            "getBeaconList output differed from key")

        newInstance := map[string]interface{} {
            "InstanceAddress" : fmt.Sprintf("INST_ADDR %d", i), 
            "BeaconAddress" : "BEACON_ADDR", 
            "Token" : "TOKEN", 
            "Users" : userMap{"USER":true},
        }

        keyList = append(keyList, newInstance["InstanceAddress"].(string))
        table.InsertSchema(newInstance)
    }

    instanceList, err := getInstancesList("BEACON_ADDR", "USER")

    assert.Nil(t, err, "getBeaconList returned an error")
    assert.Equal(t, keyList, instanceList)
}

func Test_ListInstances_BadUser(t *testing.T) {
    table, teardown := setup()
    defer teardown()

    goodInstance := map[string]interface{} {
        "InstanceAddress" : "INST_ADDR 1", 
        "BeaconAddress" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{"GOOD_USER":true},
    }

    badInstance := map[string]interface{} {
        "InstanceAddress" : "INST_ADDR 2", 
        "BeaconAddress" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{"BAD_USER":true},
    }

    keyList := []string{"INST_ADDR 1",}

    table.InsertSchema(goodInstance)
    table.InsertSchema(badInstance)

    instanceList, err := getInstancesList("BEACON_ADDR", "GOOD_USER")

    assert.Nil(t, err, "getBeaconList returned an error")
    assert.Equal(t, keyList, instanceList)
}