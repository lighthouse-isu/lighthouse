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
    "github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/databases"
)

func getDBSingleton() databases.TableInterface {
    if beacons == nil {
        panic("Beacons database not initialized")
    }
    return beacons
}

func instanceExists(instance string) bool {
    var testInstance struct { InstanceAddress string }
    columns := []string{"InstanceAddress"}
    where := databases.Filter{"InstanceAddress" : instance}

    err := getDBSingleton().SelectRowSchema(columns, where, &testInstance)
    return err != databases.NoRowsError
}

func addInstance(beacon beaconData) error {
    entry := map[string]interface{}{
        "InstanceAddress" : beacon.InstanceAddress,
        "BeaconAddress" : beacon.BeaconAddress,
        "Token" : beacon.Token,
    }
    return getDBSingleton().InsertSchema(entry)
}

func updateBeaconField(field string, val interface{}, instance string) error {
    to := databases.Filter{field : val}
    where := databases.Filter{"InstanceAddress": instance}

    return getDBSingleton().UpdateSchema(to, where)
}

func getBeaconData(instance string) (beaconData, error) {
    var beacon beaconData
    where := databases.Filter{"InstanceAddress" : instance}

    err := getDBSingleton().SelectRowSchema(nil, where, &beacon)

    if err != nil {
        return beaconData{}, err
    }
   
    return beacon, nil
}

func getBeaconsList(user *auth.User) ([]string, error) {
    opts := databases.SelectOptions{Distinct : true}
    cols := []string{"BeaconAddress"}

    scanner, err := getDBSingleton().SelectSchema(cols, nil, opts)

    if err != nil {
        return nil, err
    }

    beacons := make([]string, 0)
    seenBeacons := make(map[string]bool)
    var beacon struct {
        BeaconAddress string
    }

    for scanner.Next() {
        scanner.Scan(&beacon)

        if user.CanAccessBeacon(beacon.BeaconAddress) {
            address := beacon.BeaconAddress

            if _, found := seenBeacons[address]; !found {
                beacons = append(beacons, address)
                seenBeacons[address] = true
            }
        }
    }
   
    return beacons, nil
}

func getInstancesList(beacon string, user *auth.User) ([]string, error) {
    if !user.CanAccessBeacon(beacon) {
        return []string{}, nil
    }

    opts := databases.SelectOptions{Distinct : true}
    cols := []string{"InstanceAddress"}
    where := databases.Filter{"BeaconAddress": beacon}

    scanner, err := getDBSingleton().SelectSchema(cols, where, opts)

    if err != nil {
        return nil, err
    }

    instances := make([]string, 0)
    seenAddresses := make(map[string]bool)
    var instance struct {
        InstanceAddress string
    }

    for scanner.Next() {
        scanner.Scan(&instance)

        address := instance.InstanceAddress

        if _, found := seenAddresses[address]; !found {
            instances = append(instances, address)
            seenAddresses[address] = true
        }        
    }
   
    return instances, nil
}