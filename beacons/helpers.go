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
	"github.com/lighthouse/lighthouse/databases"
)

func getDBSingleton() databases.TableInterface {
    if beacons == nil {
        panic("Beacons database not initialized")
    }
    return beacons
}

func addBeacon(beacon beaconData) error {
    entry := map[string]interface{}{
        "InstanceAddress" : beacon.InstanceAddress,
        "BeaconAddress" : beacon.BeaconAddress,
        "Token" : beacon.Token,
        "Users" : beacon.Users,
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

func getBeaconsList() ([]string, error) {
    beacons := make([]string, 0)
    opts := databases.SelectOptions{Distinct : true}
    cols := []string{"BeaconAddress"}

    scanner, err := getDBSingleton().SelectSchema(cols, nil, opts)

    if err != nil {
        return nil, err
    }

    for scanner.Next() {
        var address struct {
            BeaconAddress string
        }

        scanner.Scan(&address)
        beacons = append(beacons, address.BeaconAddress)
    }
   
    return beacons, nil
}