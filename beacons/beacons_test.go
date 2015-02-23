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
)

func setup() (func()) {
    SetupTestingTable()
    aliases.SetupTestingTable()

    return func() {
        TeardownTestingTable()
        aliases.TeardownTestingTable()
    }
}

func Test_AddBeaconData(t *testing.T) {
    teardown := setup()
    defer teardown()

    users := userMap{"USER":true}

    testBeaconData := beaconData{
        "BEACON_ADDR", "TOKEN", users,
    }

    addBeacon(testBeaconData)

    var values beaconData
    beacons.SelectRowSchema(nil, nil, &values)

    assert.Equal(t, testBeaconData, values)
}

func Test_UpdateBeaconData(t *testing.T) {
    teardown := setup()
    defer teardown()

    testBeaconData := map[string]interface{}{ 
        "Address" : "BEACON_ADDR_FAIL", 
        "Token" : "TOKEN_FAIL", 
        "Users" : userMap{"USER_FAIL":true},
    }

    beacons.InsertSchema(testBeaconData)

    userPass := userMap{"USER_PASS":true}
    keyData := beaconData {
        "BEACON_ADDR_PASS", "TOKEN_PASS", userPass,
    }

    var values beaconData
    updateBeaconField("Token", "TOKEN_PASS", "BEACON_ADDR_FAIL")
    updateBeaconField("Users", userPass, "BEACON_ADDR_FAIL")
    updateBeaconField("Address", "BEACON_ADDR_PASS", "BEACON_ADDR_FAIL")

    beacons.SelectRowSchema(nil, nil, &values)
    assert.Equal(t, keyData, values)
}

func Test_GetBeaconAddress_Found(t *testing.T) {
    teardown := setup()
    defer teardown()

    testInstanceData := map[string]interface{}{
        "InstanceAddress" : "INST_ADDR", 
        "BeaconAddress" : "BEACON_ADDR",
    }

    instances.InsertSchema(testInstanceData)

    res, err := GetBeaconAddress("INST_ADDR")

    assert.Nil(t, err, 
        "GetBeaconAddress should not return error beacon was found")

    assert.Equal(t, "BEACON_ADDR", res, 
        "GetBeaconAddress should give correct address")
}

func Test_GetBeaconAddress_NotFound(t *testing.T) {
    teardown := setup()
    defer teardown()

    res, err := GetBeaconAddress("BAD_ADDR")

    assert.NotNil(t, err, 
        "GetBeaconAddress should forward errors")

    assert.Equal(t, "", res, 
        "GetBeaconAddress should give empty string on error")
}

func Test_GetBeaconData_Found(t *testing.T) {
    teardown := setup()
    defer teardown()

    users := userMap{"USER":true}

    testBeaconData := map[string]interface{}{
        "Address" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : users,
    }

    beacons.InsertSchema(testBeaconData)

    res, err := getBeaconData("BEACON_ADDR")

    assert.Nil(t, err, "getBeaconData should not return error beacon was found")

    key := beaconData{"BEACON_ADDR", "TOKEN", users}
    assert.Equal(t, key, res, 
        "getBeaconData should give correct beaconData")
}

func Test_GetBeaconData_NotFound(t *testing.T) {
    teardown := setup()
    defer teardown()

    res, err := getBeaconData("BAD_INST")

    assert.NotNil(t, err, "getBeaconData should forward errors")

    assert.Equal(t, beaconData{}, res, 
        "getBeaconData should give empty beaconData on error")
}

func Test_GetBeaconToken_NotFound(t *testing.T) {
    teardown := setup()
    defer teardown()

    res, err := GetBeaconToken("BAD_INST", "junk user")

    assert.NotNil(t, err, "GetBeaconToken should forward errors")

    assert.Equal(t, "", res, 
        "GetBeaconToken should give empty token on error")
}

func Test_GetBeaconToken_NotPermitted(t *testing.T) {
    teardown := setup()
    defer teardown()

    testBeaconData := map[string]interface{}{
        "Address" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{},
    }

    beacons.InsertSchema(testBeaconData)

    res, err := GetBeaconToken("BEACON_ADDR", "BAD_USER")

    assert.NotNil(t, err, 
        "GetBeaconToken should return error on bad permissions")

    assert.Equal(t, "", res, 
        "GetBeaconToken should give empty token on bad permissions")
}

func Test_GetBeaconToken_Valid(t *testing.T) {
    teardown := setup()
    defer teardown()

    testBeaconData := map[string]interface{}{
        "Address" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{"USER":true},
    }

    beacons.InsertSchema(testBeaconData)

    res, err := GetBeaconToken("BEACON_ADDR", "USER")

    assert.Nil(t, err, 
        "GetBeaconToken should return nil error on success")

    assert.Equal(t, "TOKEN", res, 
        "GetBeaconToken should give corrent token")
}

func Test_ListBeacons_ValidUser(t *testing.T) {
    teardown := setup()
    defer teardown()

    keyList := make([]aliases.Alias, 0)

    for i := 1; i <= 2; i++ {
        beaconList, err := getBeaconsList("USER")

        assert.Nil(t, err, "getBeaconList returned an error")

        assert.Equal(t, keyList, beaconList, 
            "getBeaconList output differed from key")

        newBeacon := map[string]interface{} {
            "Address" : fmt.Sprintf("BEACON_ADDR %d", i), 
            "Token" : "TOKEN", 
            "Users" : userMap{"USER":true},
        }

        keyPair := aliases.Alias{"", newBeacon["Address"].(string)}

        keyList = append(keyList, keyPair)
        beacons.InsertSchema(newBeacon)
    }

    beaconList, err := getBeaconsList("USER")

    assert.Nil(t, err, "getBeaconList returned an error")
    assert.Equal(t, keyList, beaconList)
}

func Test_ListBeacons_BadUser(t *testing.T) {
    teardown := setup()
    defer teardown()

    goodBeacon := map[string]interface{} {
        "Address" : "BEACON_ADDR 1", 
        "Token" : "TOKEN", 
        "Users" : userMap{"GOOD_USER":true},
    }

    badBeacon := map[string]interface{} {
        "Address" : "BEACON_ADDR 2", 
        "Token" : "TOKEN", 
        "Users" : userMap{"BAD_USER":true},
    }

    keyList := []aliases.Alias{aliases.Alias{"", "BEACON_ADDR 1"}}

    beacons.InsertSchema(goodBeacon)
    beacons.InsertSchema(badBeacon)

    beaconList, err := getBeaconsList("GOOD_USER")

    assert.Nil(t, err, "getBeaconList returned an error")
    assert.Equal(t, keyList, beaconList)
}

func Test_ListInstances_ValidUser(t *testing.T) {
    teardown := setup()
    defer teardown()

    beacon := map[string]interface{} {
        "Address" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{"USER":true},
    }

    beacons.InsertSchema(beacon)

    keyList := make([]instanceData, 0)

    for i := 1; i <= 2; i++ {
        instanceList, err := getInstancesList("BEACON_ADDR", "USER", false)

        assert.Nil(t, err, "getInstancesList returned an error")

        assert.Equal(t, keyList, instanceList, 
            "getInstancesList output differed from key")

        newInstance := instanceData {
            InstanceAddress : fmt.Sprintf("INST_ADDR %d", i), 
            BeaconAddress : "BEACON_ADDR",
            Name : fmt.Sprintf("VM %d", i),
            CanAccessDocker : true,
        }

        keyList = append(keyList, newInstance)

        instances.InsertSchema(map[string]interface{} {
            "InstanceAddress" : newInstance.InstanceAddress, 
            "BeaconAddress" : newInstance.BeaconAddress,
            "Name" : newInstance.Name,
            "CanAccessDocker" : newInstance.CanAccessDocker,
        })
    }

    instanceList, err := getInstancesList("BEACON_ADDR", "USER", false)

    assert.Nil(t, err, "getInstancesList returned an error")
    assert.Equal(t, keyList, instanceList)
}

func Test_ListInstances_BadUser(t *testing.T) {
    teardown := setup()
    defer teardown()

    beacon := map[string]interface{} {
        "Address" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
        "Users" : userMap{"GOOD_USER":true},
    }

    beacons.InsertSchema(beacon)

    instance := map[string]interface{} {
        "InstanceAddress" : "INST_ADDR", 
        "Name" : "NAME",
        "CanAccessDocker" : true,
        "BeaconAddress" : "BEACON_ADDR", 
    }

    instances.InsertSchema(instance)

    instanceList, err := getInstancesList("BEACON_ADDR", "BAD_USER", false)

    assert.Nil(t, err, "getInstancesList returned an error")
    assert.Equal(t, 0, len(instanceList))
}