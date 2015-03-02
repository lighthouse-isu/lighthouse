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
    "fmt"
    "errors"
    "encoding/json"
    "io/ioutil"
    "net/http"

    "github.com/lighthouse/beacon/structs"

    "github.com/lighthouse/lighthouse/beacons/aliases"
	"github.com/lighthouse/lighthouse/databases"
)

func beaconExists(beacon string) bool {
    var test beaconData
    columns := []string{"Address"}
    where := databases.Filter{"Address" : beacon}

    err := beacons.SelectRowSchema(columns, where, &test)
    return err != databases.NoRowsError
}

func instanceExists(instance string) bool {
    var test instanceData
    columns := []string{"InstanceAddress"}
    where := databases.Filter{"InstanceAddress" : instance}

    err := instances.SelectRowSchema(columns, where, &test)
    return err != databases.NoRowsError
}

func addBeacon(beacon beaconData) error {
    entry := map[string]interface{}{
        "Address" : beacon.Address,
        "Token" : beacon.Token,
        "Users" : beacon.Users,
    }

    return beacons.InsertSchema(entry)
}

func addInstance(instance instanceData) error {
    entry := map[string]interface{}{
        "InstanceAddress" : instance.InstanceAddress,
        "Name" : instance.Name,
        "CanAccessDocker" : instance.CanAccessDocker,
        "BeaconAddress" : instance.BeaconAddress,
    }

    return instances.InsertSchema(entry)
}

func updateInstance(instance instanceData) error {
    to := map[string]interface{}{
        "Name" : instance.Name,
        "CanAccessDocker" : instance.CanAccessDocker,
        "BeaconAddress" : instance.BeaconAddress,
    }

    where := map[string]interface{} {"InstanceAddress": instance.InstanceAddress}

    return instances.UpdateSchema(to, where)
}

func updateBeaconField(field string, val interface{}, beacon string) error {
    to := databases.Filter{field : val}
    where := databases.Filter{"Address": beacon}

    return beacons.UpdateSchema(to, where)
}

func getBeaconData(beacon string) (beaconData, error) {
    var data beaconData
    where := databases.Filter{"Address" : beacon}

    err := beacons.SelectRowSchema(nil, where, &data)

    if err != nil {
        return beaconData{}, err
    }
   
    return data, nil
}

func getBeaconsList(user string) ([]aliases.Alias, error) {
    opts := databases.SelectOptions{}
    cols := []string{"Address", "Users"}
    scanner, err := beacons.SelectSchema(cols, nil, opts)

    if err != nil {
        return nil, err
    }

    beacons := make([]aliases.Alias, 0)
    seenBeacons := make(map[string]bool)

    for scanner.Next() {
        var beacon struct {
            Address string
            Users userMap
        }

        scanner.Scan(&beacon)

        if _, ok := beacon.Users[user]; ok {

            address := beacon.Address
            alias, _ := aliases.GetAliasOf(address)

            pair := aliases.Alias{Alias: alias, Address: address}

            if _, found := seenBeacons[address]; !found {
                beacons = append(beacons, pair)
                seenBeacons[address] = true
            }
        }
    }
   
    return beacons, nil
}

func getInstancesList(beacon, user string, refresh bool) ([]map[string]interface{}, error) {
    data, err := getBeaconData(beacon)
    if err != nil {
        return nil, err
    }

    if _, ok := data.Users[user]; !ok {
        return nil, TokenPermissionError
    }

    if refresh {
        refreshVMListOf(data)
    }

    opts := databases.SelectOptions{Distinct : true}
    where := databases.Filter{"BeaconAddress": beacon}

    scanner, err := instances.SelectSchema(nil, where, opts)
    if err != nil {
        return nil, err
    }

    defer scanner.Close()

    instances := make([]map[string]interface{}, 0)
    InstanceAddress := make(map[string]bool)

    for scanner.Next() {
        var instance instanceData
        scanner.Scan(&instance)

        if _, ok := data.Users[user]; ok {

            address := instance.InstanceAddress
            alias, _ := aliases.GetAliasOf(address)

            if _, found := InstanceAddress[address]; !found {

                instances = append(instances, map[string]interface{}{
                    "Alias" : alias,
                    "InstanceAddress" : instance.InstanceAddress,
                    "Name" : instance.Name,
                    "CanAccessDocker" : instance.CanAccessDocker,
                    "BeaconAddress" : instance.BeaconAddress,
                })

                InstanceAddress[address] = true
            }
        }
    }
   
    return instances, nil
}

func refreshVMListOf(beacon beaconData) (error) {
    vmsTarget := fmt.Sprintf("http://%s/vms", beacon.Address)

    req, err := http.NewRequest("GET", vmsTarget, nil)
    if err != nil {
        return err
    }

    // Assuming user has permission to access token since they provided it
    req.Header.Set(HEADER_TOKEN_KEY, beacon.Token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    vmsBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    if resp.StatusCode != http.StatusOK {
        err = errors.New(string(vmsBody))
        return err
    }

    var vms []structs.VM

    err = json.Unmarshal(vmsBody, &vms)
    if err != nil {
        return err
    }

    beaconName, _ := aliases.GetAliasOf(beacon.Address)

    for _, vm := range vms {
        instanceAddr := fmt.Sprintf("%s:%s/%s", vm.Address, vm.Port, vm.Version)
        instance := instanceData{instanceAddr, vm.Name, vm.CanAccessDocker, beacon.Address}
        
        if !instanceExists(instance.InstanceAddress) {
            addInstance(instance)
        } else {
            updateInstance(instance)
        }

        aliases.SetAlias(
            fmt.Sprintf("%s%s%s", beaconName, INSTANCE_ALIAS_DELIM, vm.Name), 
            instance.InstanceAddress,
        )
    }

    return nil
}