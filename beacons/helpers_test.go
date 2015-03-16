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
    "strings"
    "encoding/json"
    "net/http"

    "github.com/stretchr/testify/assert"

    "github.com/lighthouse/beacon/structs"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/beacons/aliases"
)

func Test_BeaconExists_True(t *testing.T) {
    setup()
    defer teardown()

    testData := map[string]interface{}{ 
        "Address" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
    }

    beacons.InsertSchema(testData, "")

    assert.True(t, beaconExists("BEACON_ADDR"))
}

func Test_BeaconExists_False(t *testing.T) {
    setup()
    defer teardown()

    assert.False(t, beaconExists("BEACON_ADDR"))
}

func Test_InstanceExists_True(t *testing.T) {
    setup()
    defer teardown()

    testData := map[string]interface{}{
        "InstanceAddress" : "INST_ADDR", 
        "BeaconAddress" : "BEACON_ADDR",
        "Name" : "VM",
        "CanAccessDocker" : true,
    }

    instances.InsertSchema(testData, "")

    assert.True(t, instanceExists("INST_ADDR"))
}

func Test_InstanceExists_False(t *testing.T) {
    setup()
    defer teardown()

    assert.False(t, instanceExists("INST_ADDR"))
}

func Test_AddBeaconData_New(t *testing.T) {
    setup()
    defer teardown()

    testBeaconData := beaconData{
        "BEACON_ADDR", "TOKEN",
    }

    addBeacon(testBeaconData)

    var values beaconData
    beacons.SelectRowSchema(nil, nil, &values)

    assert.Equal(t, testBeaconData, values)
}

func Test_AddBeaconData_Dup(t *testing.T) {
    setup()
    defer teardown()

    testBeaconData := beaconData{
        "BEACON_ADDR", "TOKEN",
    }

    addBeacon(testBeaconData)

    assert.NotNil(t, addBeacon(testBeaconData))
}

func Test_AddInstanceData_New(t *testing.T) {
    setup()
    defer teardown()

    testData := instanceData{
        InstanceAddress : "INST_ADDR", 
        BeaconAddress : "BEACON_ADDR",
        Name : "VM",
        CanAccessDocker : true,
    }

    addInstance(testData)

    var values instanceData
    instances.SelectRowSchema(nil, nil, &values)

    assert.Equal(t, testData, values)
}

func Test_AddInstanceData_Dup(t *testing.T) {
    setup()
    defer teardown()

    testData := instanceData{
        InstanceAddress : "INST_ADDR", 
        BeaconAddress : "BEACON_ADDR",
        Name : "VM",
        CanAccessDocker : true,
    }

    addInstance(testData)

    assert.NotNil(t, addInstance(testData))
}

func Test_UpdateBeaconData(t *testing.T) {
    setup()
    defer teardown()

    testBeaconData := map[string]interface{}{
        "Address" : "ADDR_FAIL", 
        "Token" : "TOKEN_FAIL", 
    }

    beacons.InsertSchema(testBeaconData, "")

    var result beaconData
    
    updateBeaconField("Token", "TOKEN_PASS", "ADDR_FAIL")
    beacons.SelectRowSchema(nil, nil, &result)
    assert.Equal(t, "TOKEN_PASS", result.Token)

    updateBeaconField("Address", "ADDR_PASS", "ADDR_FAIL")
    beacons.SelectRowSchema(nil, nil, &result)
    assert.Equal(t, "ADDR_PASS", result.Address)
}

func Test_GetBeaconData_Found(t *testing.T) {
    setup()
    defer teardown()

    testBeaconData := map[string]interface{}{
        "Address" : "BEACON_ADDR", 
        "Token" : "TOKEN", 
    }

    beacons.InsertSchema(testBeaconData, "")

    res, err := getBeaconData("BEACON_ADDR")

    assert.Nil(t, err, "getBeaconData should not return error beacon was found")

    key := beaconData{"BEACON_ADDR", "TOKEN"}
    assert.Equal(t, key, res, 
        "getBeaconData should give correct beaconData")
}

func Test_GetBeaconData_NotFound(t *testing.T) {
    setup()
    defer teardown()

    res, err := getBeaconData("BAD_INST")

    assert.NotNil(t, err, "getBeaconData should forward errors")

    assert.Equal(t, beaconData{}, res, 
        "getBeaconData should give empty beaconData on error")
}

func Test_ListBeacons_ValidUser(t *testing.T) {
    setup()
    defer teardown()

    auth.CreateUser("EMAIL", "", "")
    user, _ := auth.GetUser("EMAIL")

    keyList := make([]aliases.Alias, 0)

    for i := 1; i <= 2; i++ {
        beaconList, err := getBeaconsList(user)

        assert.Nil(t, err, "getBeaconList returned an error")
        assert.Equal(t, keyList, beaconList, 
            "getBeaconList output differed from key")

        addr := fmt.Sprintf("BEACON_ADDR %d", i)

        newBeacon := map[string]interface{} {
            "Address" : addr, 
            "Token" : "TOKEN", 
        }

        auth.SetUserBeaconAuthLevel(user, addr, auth.OwnerAuthLevel)

        keyList = append(keyList, aliases.Alias{"", newBeacon["Address"].(string)})
        beacons.InsertSchema(newBeacon, "")
    }

    beaconList, err := getBeaconsList(user)

    assert.Nil(t, err, "getBeaconList returned an error")
    assert.Equal(t, keyList, beaconList)
}

func Test_ListBeacons_BadUser(t *testing.T) {
    setup()
    defer teardown()

    goodBeacon := map[string]interface{} {
        "Address" : "BEACON_ADDR 1", 
        "Token" : "TOKEN", 
    }

    badBeacon := map[string]interface{} {
        "Address" : "BEACON_ADDR 2", 
        "Token" : "TOKEN", 
    }

    auth.CreateUser("EMAIL", "", "")
    user, _ := auth.GetUser("EMAIL")
    auth.SetUserBeaconAuthLevel(user, "BEACON_ADDR 1", auth.OwnerAuthLevel)

    beacons.InsertSchema(goodBeacon, "")
    beacons.InsertSchema(badBeacon, "")

    beaconList, err := getBeaconsList(user)

    assert.Equal(t, 1, len(beaconList))
    assert.Nil(t, err)
    assert.Equal(t, "BEACON_ADDR 1", beaconList[0].Address)
}

func Test_ListInstances_ValidUser(t *testing.T) {
    setup()
    defer teardown()

    auth.CreateUser("EMAIL", "", "")
    user, _ := auth.GetUser("EMAIL")
    auth.SetUserBeaconAuthLevel(user, "BEACON_ADDR", auth.OwnerAuthLevel)

    beacons.InsertSchema(map[string]interface{}{
        "Address" : "BEACON_ADDR", "Token" : "TOKEN",
    }, "")

    keyList := make([]map[string]interface{}, 0)

    for i := 1; i <= 2; i++ {
        instanceList, err := getInstancesList("BEACON_ADDR", user, false)

        assert.Nil(t, err, "getInstancesList returned an error")
        assert.Equal(t, keyList, instanceList, 
            "getInstancesList output differed from key")

        newInstance := map[string]interface{} {
            "InstanceAddress" : fmt.Sprintf("INST_ADDR %d", i), 
            "Name" : "VM",
            "CanAccessDocker" : true,
            "BeaconAddress" : "BEACON_ADDR", 
        }

        instances.InsertSchema(newInstance, "")

        newInstance["Alias"] = ""
        keyList = append(keyList, newInstance)
    }

    instanceList, err := getInstancesList("BEACON_ADDR", user, false)

    assert.Nil(t, err, "getInstancesList returned an error")
    assert.Equal(t, keyList, instanceList)
}

func Test_ListInstances_BadUser(t *testing.T) {
    setup()
    defer teardown()

    beacons.InsertSchema(map[string]interface{}{
        "Address" : "BEACON_ADDR 1", "Token" : "TOKEN",
    }, "")

    beacons.InsertSchema(map[string]interface{}{
        "Address" : "BEACON_ADDR 2", "Token" : "TOKEN",
    }, "")

    auth.CreateUser("EMAIL", "", "")
    user, _ := auth.GetUser("EMAIL")
    auth.SetUserBeaconAuthLevel(user, "BEACON_ADDR 1", auth.OwnerAuthLevel)

    goodInstance := map[string]interface{} {
        "InstanceAddress" : "INST_ADDR 1",
        "Name" : "NAME 1",
        "CanAccessDocker" : true,
        "BeaconAddress" : "BEACON_ADDR 1", 
    }

    badInstance := map[string]interface{} {
        "InstanceAddress" : "INST_ADDR 2",
        "Name" : "NAME 2",
        "CanAccessDocker" : false,
        "BeaconAddress" : "BEACON_ADDR 2", 
    }

    instances.InsertSchema(goodInstance, "")
    instances.InsertSchema(badInstance, "")

    key := goodInstance
    key["Alias"] = ""

    instanceList, err := getInstancesList("BEACON_ADDR 1", user, false)

    assert.Nil(t, err, "getInstancesList returned an error")
    assert.Equal(t, 1, len(instanceList))
    assert.Equal(t, key, instanceList[0])
}

func Test_RefreshVMListOf(t *testing.T) {
    setup()
    defer teardown()

    vm := structs.VM {
        Name : "NAME",
        Address : "ADDR",
        Port : "1234",
        Version : "v1.12",
        CanAccessDocker : true,
    }

    f := func(w http.ResponseWriter, r *http.Request) {
        val, _ := json.Marshal([]structs.VM{vm})
        fmt.Fprint(w, string(val))
    }

    s := setupServer(&f)
    defer s.Close()

    url := strings.Replace(s.URL, "http://", "", 1)

    beacons.InsertSchema(map[string]interface{} {
        "Address" : url, 
        "Token" : "", 
    }, "")

    data := beaconData{Address: url}

    refreshVMListOf(data)

    key := instanceData {
        fmt.Sprintf("%s:%s/%s", vm.Address, vm.Port, vm.Version),
        vm.Name,
        vm.CanAccessDocker,
        data.Address,
    }

    var inst instanceData
    instances.SelectRowSchema(nil, nil, &inst)

    assert.Equal(t, key, inst)
}
