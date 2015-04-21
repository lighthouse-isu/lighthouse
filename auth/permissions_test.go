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

package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewPermission(t *testing.T) {
	permissions := NewPermission()

	_, beaconOK := permissions["Beacons"]
	assert.True(t, beaconOK, "NewPermission should have 'Beacons'")
}

func Test_GetAuthLevel(t *testing.T) {
	user := &User{}
	user.Permissions = NewPermission()

	beaconPerms := map[string]interface{}{
		"GOOD" : 1,
	}

	user.Permissions["Beacons"] = beaconPerms

	assert.Equal(t, 1, user.GetAuthLevel("Beacons", "GOOD"))
	assert.Equal(t, -1, user.GetAuthLevel("Beacons", "WRONG TYPE"))
	assert.Equal(t, -1, user.GetAuthLevel("Beacons", "BAD"))
	assert.Equal(t, -1, user.GetAuthLevel("BAD TYPE", "JUNK"))
}

func Test_SetAuthLevel(t *testing.T) {
	user := &User{}
	user.Permissions = NewPermission()

	user.SetAuthLevel("Beacons", "KEY", 1)
	user.SetAuthLevel("Beacons", "OVERWRITE", 0)
	user.SetAuthLevel("Beacons", "OVERWRITE", 2)

	beaconPerms := user.Permissions["Beacons"].(map[string]interface{})

	assert.Equal(t, 1, beaconPerms["KEY"].(int))
	assert.Equal(t, 2, beaconPerms["OVERWRITE"].(int))

	// Make sure this doesn't cause a panic
	user.SetAuthLevel("BAD TYPE", "JUNK", 2)

	user.Permissions["NEW TYPE"] = nil
	user.SetAuthLevel("NEW TYPE", "KEY", 1)

	newPerms := user.Permissions["NEW TYPE"].(map[string]interface{})
	assert.Equal(t, 1, newPerms["KEY"].(int))
}

func Test_CanViewUser(t *testing.T) {
	low := &User{Email: "low", AuthLevel: 0}
	middle := &User{Email: "middle", AuthLevel: 1}
	high := &User{Email: "high", AuthLevel: 2}

	assert.True(t, middle.CanViewUser(low))
	assert.True(t, high.CanViewUser(middle))
	assert.True(t, high.CanViewUser(low))
	assert.True(t, middle.CanViewUser(middle))

	assert.False(t, low.CanViewUser(middle))
	assert.False(t, low.CanViewUser(high))
	assert.False(t, middle.CanViewUser(high))
}

func Test_CanModifyUser(t *testing.T) {
	low := &User{Email: "low", AuthLevel: 0}
	middle := &User{Email: "middle", AuthLevel: 1}
	high := &User{Email: "high", AuthLevel: 2}

	assert.True(t, middle.CanModifyUser(low))
	assert.True(t, high.CanModifyUser(middle))
	assert.True(t, high.CanModifyUser(low))
	assert.True(t, middle.CanModifyUser(middle))

	assert.False(t, low.CanModifyUser(middle))
	assert.False(t, low.CanModifyUser(high))
	assert.False(t, middle.CanModifyUser(high))
}

func Test_CanAccessBeacon(t *testing.T) {
	user := &User{}
	user.Permissions = NewPermission()

	beaconPerms := map[string]interface{}{
		"access" : AccessAuthLevel,
		"modify" : ModifyAuthLevel,
		"owner" : OwnerAuthLevel,
	}

	user.Permissions["Beacons"] = beaconPerms

	assert.False(t, user.CanAccessBeacon("none"))
	assert.True(t, user.CanAccessBeacon("access"))
	assert.True(t, user.CanAccessBeacon("modify"))
	assert.True(t, user.CanAccessBeacon("owner"))
}

func Test_CanModifyBeacon(t *testing.T) {
	user := &User{}
	user.Permissions = NewPermission()

	beaconPerms := map[string]interface{}{
		"access" : AccessAuthLevel,
		"modify" : ModifyAuthLevel,
		"owner" : OwnerAuthLevel,
	}

	user.Permissions["Beacons"] = beaconPerms

	assert.False(t, user.CanModifyBeacon("none"))
	assert.False(t, user.CanModifyBeacon("access"))
	assert.True(t, user.CanModifyBeacon("modify"))
	assert.True(t, user.CanModifyBeacon("owner"))
}