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

	"net/http"

	"github.com/stretchr/testify/assert"

	"github.com/lighthouse/lighthouse/session"
	"github.com/lighthouse/lighthouse/databases"
)

func setupTests() (table *databases.MockTable, teardown func()) {
    table = databases.CommonTestingTable(schema)
    SetupCustomTestingTable(table)

    teardown = func() {
        TeardownTestingTable()
    }

    return
}

func Test_CreateUser(t *testing.T) {
    table, teardown := setupTests()
    defer teardown()

    email := "EMAIL"
    salt := "SALT"
    password := "PASSWORD"

    CreateUser(email, salt, password)

    keyUser := User {
    	email, salt, password, DefaultAuthLevel, *NewPermission(),
    }

    var actual User
    table.SelectRowSchema(nil, nil, &actual)

	assert.Equal(t, keyUser, actual)
}   

func Test_CreateUserWithAuthLevel(t *testing.T) {
	table, teardown := setupTests()
    defer teardown()

    email := "EMAIL"
    salt := "SALT"
    password := "PASSWORD"
    authLevel := 3

    createUserWithAuthLevel(email, salt, password, authLevel)

    keyUser := User {
    	email, salt, password, authLevel, *NewPermission(),
    }

    var actual User
    table.SelectRowSchema(nil, nil, &actual)

	assert.Equal(t, keyUser, actual)
}

func Test_GetUser_Valid(t *testing.T) {
	table, teardown := setupTests()
    defer teardown()

    perms := *NewPermission()
    perms["TestField"] = map[string]interface{}{"TestKey" : 1}

    keyUser := User {
	    "EMAIL", "SALT", "PASSWORD", 3, perms,
	}

    table.InsertSchema(map[string]interface{}{
    	"Email" : keyUser.Email,
    	"Salt" : keyUser.Salt,
    	"Password" : keyUser.Password,
    	"AuthLevel" : keyUser.AuthLevel,
    	"Permissions" : perms,
    })

    user, err := GetUser(keyUser.Email)

    assert.Nil(t, err)
    assert.Equal(t, keyUser.Email, user.Email)
    assert.Equal(t, keyUser.Salt, user.Salt)
    assert.Equal(t, keyUser.Password, user.Password)
    assert.Equal(t, keyUser.AuthLevel, user.AuthLevel)
    assert.Equal(t, keyUser.Permissions, user.Permissions)
}

func Test_GetUser_Invalid(t *testing.T) {
	_, teardown := setupTests()
    defer teardown()

    user, err := GetUser("BAD EMAIL")

    assert.NotNil(t, err)
    assert.Nil(t, user)
}

func Test_GetCurrentUser(t *testing.T) {
	table, teardown := setupTests()
    defer teardown()

    r, _ := http.NewRequest("GET", "/", nil)
    session.SetValue(r, "auth", "email", "EMAIL")

    perms := *NewPermission()
    perms["TestField"] = map[string]interface{}{"TestKey" : 1}

    keyUser := User {
	    "EMAIL", "SALT", "PASSWORD", 3, perms,
	}

    table.InsertSchema(map[string]interface{}{
    	"Email" : keyUser.Email,
    	"Salt" : keyUser.Salt,
    	"Password" : keyUser.Password,
    	"AuthLevel" : keyUser.AuthLevel,
    	"Permissions" : perms,
    })

    user := GetCurrentUser(r)

    assert.Equal(t, keyUser.Email, user.Email)
    assert.Equal(t, keyUser.Salt, user.Salt)
    assert.Equal(t, keyUser.Password, user.Password)
    assert.Equal(t, keyUser.AuthLevel, user.AuthLevel)
    assert.Equal(t, keyUser.Permissions, user.Permissions)
}

func Test_SetUserBeaconAuthLevel(t *testing.T) {
	table, teardown := setupTests()
    defer teardown()

    perms := *NewPermission()
    perms["Beacons"] = map[string]interface{}{"OVERWRITE" : 0}

    keyPerms := map[string]interface{}{
    	"OVERWRITE" : 1, "NEW": 2,
    }

    table.InsertSchema(map[string]interface{}{
    	"Email" : "EMAIL",
    	"Permissions" : perms,
    })

    user := &User{Email: "EMAIL", Permissions: perms,}

    SetUserBeaconAuthLevel(user, "OVERWRITE", 1)
    SetUserBeaconAuthLevel(user, "NEW", 2)

    cols := []string{"Permissions"}
    table.SelectRowSchema(cols, nil, user)

    assert.Equal(t, keyPerms, user.Permissions["Beacons"])
}

func Test_GetAllUsers(t *testing.T) {
	table, teardown := setupTests()
    defer teardown()

	current := &User{Email: "current", AuthLevel: 1}

	table.InsertSchema(map[string]interface{}{
    	"Email" : "current", "AuthLevel" : 1,
    })

	table.InsertSchema(map[string]interface{}{
    	"Email" : "lower", "AuthLevel" : 0,
    })

	table.InsertSchema(map[string]interface{}{
    	"Email" : "equal", "AuthLevel" : 1,
    })

	table.InsertSchema(map[string]interface{}{
    	"Email" : "higher", "AuthLevel" : 2,
    })

    users, _ := getAllUsers(current)

    assert.Equal(t, 2, len(users))
    if len(users) != 2 {
    	return
    }

    assert.True(t, users[0] == "current" || users[1] == "current",
    	"getAllUsers should list current user")

    assert.True(t, users[0] == "lower" || users[1] == "lower",
    	"getAllUsers should list less privileged users")
}

func Test_ParseUserUpdateRequest_AuthLevel_Valid(t *testing.T) {
	curUser := &User{AuthLevel: 1}
	modUser := &User{AuthLevel: 0}

	updates := make(map[string]interface{})
	var vals map[string]interface{}
	var code int

	updates["AuthLevel"] = 0
	vals, code = parseUserUpdateRequest(curUser, modUser, updates)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, updates["AuthLevel"], vals["AuthLevel"])

	updates["AuthLevel"] = 1
	vals, code = parseUserUpdateRequest(curUser, modUser, updates)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, updates["AuthLevel"], vals["AuthLevel"])
}

func Test_ParseUserUpdateRequest_AuthLevel_Invalid(t *testing.T) {
	curUser := &User{AuthLevel: 1}
	modUser := &User{AuthLevel: 0}

	updates := make(map[string]interface{})
	var vals map[string]interface{}
	var code int

	updates["AuthLevel"] = -1
	vals, code = parseUserUpdateRequest(curUser, modUser, updates)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Nil(t, vals)

	updates["AuthLevel"] = 2
	vals, code = parseUserUpdateRequest(curUser, modUser, updates)
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Nil(t, vals)
}

func Test_ParseUserUpdateRequest_Password(t *testing.T) {
	modUser := &User{AuthLevel: 0}

	keyPassword := SaltPassword("PASSWORD", modUser.Password)

	updates := make(map[string]interface{})
	updates["Password"] = "PASSWORD"
	
	vals, code := parseUserUpdateRequest(nil, modUser, updates)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, keyPassword, vals["Password"])
}

func Test_ParseUserUpdateRequest_Beacons_Valid(t *testing.T) {
	curPerms := *NewPermission()
	curPerms["Beacons"] = map[string]interface{}{
		"Beacon 1" : ModifyAuthLevel,
		"Beacon 2" : ModifyAuthLevel,
		"Beacon 3" : ModifyAuthLevel,
	}

	modPerms := *NewPermission()
	modPerms["Beacons"] = map[string]interface{}{
		"Beacon 1" : AccessAuthLevel,
		"Beacon 3" : ModifyAuthLevel,
	}

	curUser := &User{Permissions: curPerms}
	modUser := &User{Permissions: modPerms}

	updates := make(map[string]interface{})
	updates["Beacons"] = map[string]interface{}{
		"Beacon 1" : ModifyAuthLevel,
		"Beacon 2" : AccessAuthLevel,
		"Beacon 3" : -1,
	}

	vals, code := parseUserUpdateRequest(curUser, modUser, updates)
	perms := vals["Permissions"].(Permission)
	beacons := perms["Beacons"].(map[string]interface{})
	_, found := beacons["Beacon 3"]

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, ModifyAuthLevel, beacons["Beacon 1"])
	assert.Equal(t, AccessAuthLevel, beacons["Beacon 2"])
	assert.False(t, found) // Beacon 3 removed
}

func Test_ParseUserUpdateRequest_Beacons_CantModify(t *testing.T) {
	curPerms := *NewPermission()
	curPerms["Beacons"] = map[string]interface{}{
		"Beacon" : AccessAuthLevel,
	}

	modPerms := *NewPermission()

	curUser := &User{Permissions: curPerms}
	modUser := &User{Permissions: modPerms}

	updates := make(map[string]interface{})
	updates["Beacons"] = map[string]interface{}{
		"Beacon" : AccessAuthLevel,
	}

	vals, code := parseUserUpdateRequest(curUser, modUser, updates)
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Nil(t, vals)
}

func Test_ParseUserUpdateRequest_Beacons_TooHigh(t *testing.T) {
	curPerms := *NewPermission()
	curPerms["Beacons"] = map[string]interface{}{
		"Beacon" : ModifyAuthLevel,
	}

	modPerms := *NewPermission()

	curUser := &User{Permissions: curPerms}
	modUser := &User{Permissions: modPerms}

	updates := make(map[string]interface{})
	updates["Beacons"] = map[string]interface{}{
		"Beacon" : OwnerAuthLevel,
	}

	vals, code := parseUserUpdateRequest(curUser, modUser, updates)
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Nil(t, vals)
}