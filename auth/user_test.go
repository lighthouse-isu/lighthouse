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

	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/lighthouse/lighthouse/databases"
	"github.com/lighthouse/lighthouse/session"
)

func setup() {
	SetupTestingTable()
}

func teardown() {
	TeardownTestingTable()
}

func addUsers(list ...User) {
	for _, user := range list {

		users.Insert(map[string]interface{}{
			"Email":       user.Email,
			"Salt":        user.Salt,
			"Password":    user.Password,
			"AuthLevel":   user.AuthLevel,
			"Permissions": user.Permissions,
		})
	}
}

func handleAndServe(endpoint string, f http.HandlerFunc, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	m := mux.NewRouter()
	m.HandleFunc(endpoint, f)
	m.ServeHTTP(w, r)
	return w
}

func Test_WriteResponse_OK(t *testing.T) {
	w := httptest.NewRecorder()
	writeResponse(w, 200, nil)
	assert.Equal(t, 200, w.Code)
}

func Test_CreateUser(t *testing.T) {
	setup()
	defer teardown()

	email := "EMAIL"
	salt := "SALT"
	password := "PASSWORD"

	CreateUser(email, salt, password)

	keyUser := User{
		email, salt, password, DefaultAuthLevel, NewPermission(),
	}

	var actual User
	users.SelectRow(nil, nil, nil, &actual)

	assert.Equal(t, keyUser, actual)
}

func Test_CreateUserWithAuthLevel(t *testing.T) {
	setup()
	defer teardown()

	email := "EMAIL"
	salt := "SALT"
	password := "PASSWORD"
	authLevel := 3

	createUserWithAuthLevel(email, salt, password, authLevel)

	keyUser := User{
		email, salt, password, authLevel, NewPermission(),
	}

	var actual User
	users.SelectRow(nil, nil, nil, &actual)

	assert.Equal(t, keyUser, actual)
}

func Test_GetUser_Valid(t *testing.T) {
	setup()
	defer teardown()

	perms := NewPermission()
	perms["TestField"] = map[string]interface{}{"TestKey": 1}

	keyUser := User{
		"EMAIL", "SALT", "PASSWORD", 3, perms,
	}

	addUsers(keyUser)

	user, err := GetUser(keyUser.Email)

	assert.Nil(t, err)
	assert.Equal(t, keyUser.Email, user.Email)
	assert.Equal(t, keyUser.Salt, user.Salt)
	assert.Equal(t, keyUser.Password, user.Password)
	assert.Equal(t, keyUser.AuthLevel, user.AuthLevel)
	assert.Equal(t, keyUser.Permissions, user.Permissions)
}

func Test_GetUser_Invalid(t *testing.T) {
	setup()
	defer teardown()

	user, err := GetUser("BAD EMAIL")

	assert.NotNil(t, err)
	assert.Nil(t, user)
}

func Test_GetCurrentUser(t *testing.T) {
	setup()
	defer teardown()

	r, _ := http.NewRequest("GET", "/", nil)
	session.SetValue(r, "auth", "email", "EMAIL")

	perms := NewPermission()
	perms["TestField"] = map[string]interface{}{"TestKey": 1}

	keyUser := User{
		"EMAIL", "SALT", "PASSWORD", 3, perms,
	}

	addUsers(keyUser)

	user := GetCurrentUser(r)

	assert.Equal(t, keyUser.Email, user.Email)
	assert.Equal(t, keyUser.Salt, user.Salt)
	assert.Equal(t, keyUser.Password, user.Password)
	assert.Equal(t, keyUser.AuthLevel, user.AuthLevel)
	assert.Equal(t, keyUser.Permissions, user.Permissions)
}

func Test_SetUserBeaconAuthLevel(t *testing.T) {
	setup()
	defer teardown()

	perms := NewPermission()
	perms["Beacons"] = map[string]interface{}{"OVERWRITE": 0}

	keyPerms := map[string]interface{}{
		"OVERWRITE": 1, "NEW": 2,
	}

	user := &User{Email: "EMAIL", Permissions: perms}

	addUsers(*user)

	SetUserBeaconAuthLevel(user, "OVERWRITE", 1)
	SetUserBeaconAuthLevel(user, "NEW", 2)

	cols := []string{"Permissions"}
	users.SelectRow(cols, nil, nil, user)

	assert.Equal(t, keyPerms, user.Permissions["Beacons"])
}

func Test_GetAllUsers(t *testing.T) {
	setup()
	defer teardown()

	current := &User{Email: "current", AuthLevel: 1}

	addUsers(
		*current,
		User{Email: "lower", AuthLevel: 0},
		User{Email: "equal", AuthLevel: 1},
		User{Email: "higher", AuthLevel: 2},
	)

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

	var update []byte
	var vals map[string]interface{}
	var code int

	update = []byte(`{"AuthLevel" : 1}`)
	vals, code = parseUserUpdateRequest(curUser, modUser, update)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 1, vals["AuthLevel"])
}

func Test_ParseUserUpdateRequest_AuthLevel_Invalid(t *testing.T) {
	curUser := &User{AuthLevel: 1}
	modUser := &User{AuthLevel: 0}

	var update []byte
	var vals map[string]interface{}
	var code int

	update = []byte(`{"AuthLevel" : -1}`)
	vals, code = parseUserUpdateRequest(curUser, modUser, update)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Nil(t, vals)

	update = []byte(`{"AuthLevel" : 2}`)
	vals, code = parseUserUpdateRequest(curUser, modUser, update)
	assert.Equal(t, http.StatusForbidden, code)
	assert.Nil(t, vals)
}

func Test_ParseUserUpdateRequest_Password(t *testing.T) {
	modUser := &User{AuthLevel: 0, Password: "OLD"}

	keyPassword := SaltPassword("PASSWORD", modUser.Salt)

	update := []byte(`{"Password" : "PASSWORD"}`)
	vals, code := parseUserUpdateRequest(nil, modUser, update)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, keyPassword, vals["Password"])
}

func Test_ParseUserUpdateRequest_Beacons_Valid(t *testing.T) {
	curPerms := NewPermission()
	curPerms["Beacons"] = map[string]interface{}{
		"Beacon 1": ModifyAuthLevel,
		"Beacon 2": ModifyAuthLevel,
		"Beacon 3": ModifyAuthLevel,
	}

	modPerms := NewPermission()
	modPerms["Beacons"] = map[string]interface{}{
		"Beacon 1": AccessAuthLevel,
		"Beacon 3": ModifyAuthLevel,
	}

	curUser := &User{Permissions: curPerms}
	modUser := &User{Permissions: modPerms}

	updateStr := fmt.Sprintf(
		`{"Beacons" : {"Beacon 1": %d, "Beacon 2" : %d, "Beacon 3" : %d}}`,
		ModifyAuthLevel, AccessAuthLevel, -1)

	vals, code := parseUserUpdateRequest(curUser, modUser, []byte(updateStr))
	assert.Equal(t, http.StatusOK, code)
	if code != http.StatusOK {
		return
	}

	perms := vals["Permissions"].(Permission)
	beacons := perms["Beacons"].(map[string]interface{})
	_, found := beacons["Beacon 3"]

	assert.Equal(t, ModifyAuthLevel, beacons["Beacon 1"])
	assert.Equal(t, AccessAuthLevel, beacons["Beacon 2"])
	assert.False(t, found) // Beacon 3 removed
}

func Test_ParseUserUpdateRequest_Beacons_CantModify(t *testing.T) {
	curPerms := NewPermission()
	curPerms["Beacons"] = map[string]interface{}{
		"Beacon": AccessAuthLevel,
	}

	modPerms := NewPermission()

	curUser := &User{Permissions: curPerms}
	modUser := &User{Permissions: modPerms}

	updateStr := fmt.Sprintf(`{"Beacons" : {"Beacon" : %d}}`, AccessAuthLevel)

	vals, code := parseUserUpdateRequest(curUser, modUser, []byte(updateStr))
	assert.Equal(t, http.StatusForbidden, code)
	assert.Nil(t, vals)
}

func Test_ParseUserUpdateRequest_Beacons_TooHigh(t *testing.T) {
	curPerms := NewPermission()
	curPerms["Beacons"] = map[string]interface{}{
		"Beacon": ModifyAuthLevel,
	}

	modPerms := NewPermission()

	curUser := &User{Permissions: curPerms}
	modUser := &User{Permissions: modPerms}

	updateStr := fmt.Sprintf(`{"Beacons" : {"Beacon" : %d}}`, OwnerAuthLevel)

	vals, code := parseUserUpdateRequest(curUser, modUser, []byte(updateStr))
	assert.Equal(t, http.StatusForbidden, code)
	assert.Nil(t, vals)
}

func Test_ParseUserUpdateRequest_BadJSON(t *testing.T) {
	curPerms := NewPermission()
	curPerms["Beacons"] = map[string]interface{}{
		"Beacon": ModifyAuthLevel,
	}

	modPerms := NewPermission()

	curUser := &User{Permissions: curPerms}
	modUser := &User{Permissions: modPerms}

	updateStr := "{"

	vals, code := parseUserUpdateRequest(curUser, modUser, []byte(updateStr))
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Nil(t, vals)
}

func Test_HandleListUsers(t *testing.T) {
	setup()
	defer teardown()

	keyEmail := "EMAIL"

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	session.SetValue(r, "auth", "email", keyEmail)

	addUsers(User{Email: keyEmail})

	http.Handler(http.HandlerFunc(handleListUsers)).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), keyEmail))
}

func Test_HandleGetUsers_Valid(t *testing.T) {
	setup()
	defer teardown()

	keyEmail := "EMAIL"

	r, _ := http.NewRequest("GET", "/EMAIL", nil)

	session.SetValue(r, "auth", "email", keyEmail)
	addUsers(User{Email: keyEmail, AuthLevel: 2})

	w := handleAndServe("/{Email}", handleGetUser, r)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := make(map[string]interface{})
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, "EMAIL", resp["Email"])
	assert.Equal(t, float64(2), resp["AuthLevel"])
}

func Test_HandleGetUsers_NotFound(t *testing.T) {
	setup()
	defer teardown()

	r, _ := http.NewRequest("GET", "/BAD", nil)

	session.SetValue(r, "auth", "email", "USER")
	addUsers(User{Email: "USER", AuthLevel: 0})

	w := handleAndServe("/{Email}", handleGetUser, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_HandleGetUsers_NotAuthorized(t *testing.T) {
	setup()
	defer teardown()

	r, _ := http.NewRequest("GET", "/ADMIN", nil)

	session.SetValue(r, "auth", "email", "USER")
	addUsers(
		User{Email: "ADMIN", AuthLevel: 2},
		User{Email: "USER", AuthLevel: 0},
	)

	w := handleAndServe("/{Email}", handleGetUser, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_HandleUpdateUser_Valid(t *testing.T) {
	setup()
	defer teardown()

	keyUser := User{
		Email:     "USER",
		AuthLevel: 1,
		Password:  SaltPassword("PASSWORD", ""),
		Permissions: map[string]interface{}{
			"Beacons": map[string]interface{}{
				"BEACON": AccessAuthLevel,
			},
		},
	}

	baseUser := User{
		Email:     "USER",
		AuthLevel: 2,
		Password:  "OLD",
		Permissions: map[string]interface{}{
			"Beacons": map[string]interface{}{
				"BEACON": OwnerAuthLevel,
			},
		},
	}

	addUsers(baseUser)

	updateJSON, _ := json.Marshal(
		map[string]interface{}{
			"AuthLevel": 1,
			"Password":  "PASSWORD",
			"Beacons": map[string]interface{}{
				"BEACON": AccessAuthLevel,
			},
		})

	r, _ := http.NewRequest("PUT", "/USER", bytes.NewBuffer(updateJSON))

	session.SetValue(r, "auth", "email", "USER")

	w := handleAndServe("/{Email}", handleUpdateUser, r)

	var user User
	users.SelectRow(nil, nil, nil, &user)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, keyUser, user)
}

func Test_HandleUpdateUser_NotAuthorized(t *testing.T) {
	setup()
	defer teardown()

	keyUser := User{
		Email:     "USER",
		AuthLevel: 0,
		Password:  "OLD",
		Permissions: map[string]interface{}{
			"Beacons": map[string]interface{}{
				"BEACON": OwnerAuthLevel,
			},
		},
	}

	addUsers(keyUser)

	updateJSON := []byte(`{"AuthLevel" : 1}`)

	r, _ := http.NewRequest("PUT", "/USER", bytes.NewBuffer(updateJSON))

	session.SetValue(r, "auth", "email", "USER")

	w := handleAndServe("/{Email}", handleUpdateUser, r)

	var user User
	users.SelectRow(nil, nil, nil, &user)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, keyUser, user)
}

func Test_HandleUpdateUser_CantView(t *testing.T) {
	setup()
	defer teardown()

	actingUser := User{
		Email: "USER", AuthLevel: 0,
	}

	modUser := User{
		Email: "USER2", AuthLevel: 1,
	}

	addUsers(actingUser, modUser)

	r, _ := http.NewRequest("PUT", "/USER2", bytes.NewBuffer([]byte(`{}`)))

	session.SetValue(r, "auth", "email", "USER")

	w := handleAndServe("/{Email}", handleUpdateUser, r)

	var user User
	users.SelectRow(nil, nil, nil, &user)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_HandleUpdateUser_UnknownUser(t *testing.T) {
	setup()
	defer teardown()

	actingUser := User{
		Email: "USER", AuthLevel: 0,
	}

	addUsers(actingUser)

	r, _ := http.NewRequest("PUT", "/BAD_USER", bytes.NewBuffer([]byte(`{}`)))

	session.SetValue(r, "auth", "email", "USER")

	w := handleAndServe("/{Email}", handleUpdateUser, r)

	var user User
	users.SelectRow(nil, nil, nil, &user)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_HandleCreateUser_Valid(t *testing.T) {
	setup()
	defer teardown()

	admin := User{Email: "ADMIN", AuthLevel: 1}
	addUsers(admin)

	add := map[string]string{"Email": "USER", "Password": "PASSWORD"}
	addJSON, _ := json.Marshal(add)

	r, _ := http.NewRequest("POST", "/", bytes.NewBuffer(addJSON))
	session.SetValue(r, "auth", "email", "ADMIN")

	w := handleAndServe("/", handleCreateUser, r)

	var user User
	where := databases.Filter{"Email": "USER"}
	users.SelectRow(nil, where, nil, &user)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "USER", user.Email)
	assert.Equal(t, SaltPassword("PASSWORD", user.Salt), user.Password)
	assert.Equal(t, 0, user.AuthLevel)
}

func Test_HandleCreateUser_Unauthorized(t *testing.T) {
	setup()
	defer teardown()

	admin := User{Email: "ADMIN", AuthLevel: 0}
	addUsers(admin)

	add := map[string]string{"Email": "USER", "Password": "PASSWORD"}
	addJSON, _ := json.Marshal(add)

	r, _ := http.NewRequest("POST", "/", bytes.NewBuffer(addJSON))
	session.SetValue(r, "auth", "email", "ADMIN")

	w := handleAndServe("/", handleCreateUser, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func Test_HandleCreateUser_BadJSON(t *testing.T) {
	setup()
	defer teardown()

	admin := User{Email: "ADMIN", AuthLevel: 1}
	addUsers(admin)

	add := []int{1}
	addJSON, _ := json.Marshal(add)

	r, _ := http.NewRequest("POST", "/", bytes.NewBuffer(addJSON))
	session.SetValue(r, "auth", "email", "ADMIN")

	w := handleAndServe("/", handleCreateUser, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_HandleCreateUser_Exists(t *testing.T) {
	setup()
	defer teardown()

	admin := User{Email: "ADMIN", AuthLevel: 1}
	addUsers(admin)

	add := map[string]string{"Email": "ADMIN", "Password": "PASSWORD"}
	addJSON, _ := json.Marshal(add)

	r, _ := http.NewRequest("POST", "/", bytes.NewBuffer(addJSON))
	session.SetValue(r, "auth", "email", "ADMIN")

	w := handleAndServe("/", handleCreateUser, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
