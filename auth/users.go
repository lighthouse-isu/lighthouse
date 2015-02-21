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
    "fmt"
    "net/http"
    "io/ioutil"
    "encoding/json"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/session"
    "github.com/lighthouse/lighthouse/databases"
    "github.com/lighthouse/lighthouse/databases/postgres"
)

const (
    DefaultAuthLevel = 0
    CreateUserAuthLevel = 1
)

type User struct {
    Email string
    Salt string
    Password string
    AuthLevel int
    Permissions Permission
}

var users databases.TableInterface

var schema = databases.Schema {
    "Email" : "text UNIQUE PRIMARY KEY",
    "Salt" : "text",
    "Password" : "text",
    "AuthLevel" : "int",
    "Permissions" : "json",
}

func getDBSingleton() databases.TableInterface {
    if users == nil {
        panic("Users database not initialized")
    }
    return users
}

func Init() {
    if users == nil {
        users = databases.NewSchemaTable(postgres.Connection(), "users", schema)
    }
}

func CreateUser(email, salt, password string) error {
    return createUserWithAuthLevel(email, salt, password, DefaultAuthLevel)
}   

func createUserWithAuthLevel(email, salt, password string, level int) error {
    user := map[string]interface{}{
        "Email" : email,
        "Salt" : salt,
        "Password" : password,
        "AuthLevel" : level,
        "Permissions" : NewPermission(),
    }

    return getDBSingleton().InsertSchema(user)
}

func GetUser(email string) (*User, error) {
    user := &User{}
    where := databases.Filter{"Email" : email}
    err := getDBSingleton().SelectRowSchema(nil, where, user)

    if err != nil {
        return nil, err
    }

    user.convertPermissionsFromDB()

    return user, nil
}

func GetCurrentUser(r *http.Request) *User {
    email := session.GetValueOrDefault(r, "auth", "email", "").(string)
    user, _ := GetUser(email)
    return user
}

func SetUserBeaconAuthLevel(user *User, beacon string, level int) error {
    user.SetAuthLevel("Beacons", beacon, level)
    
    to := map[string]interface{}{"Permissions" : user.Permissions}
    where := map[string]interface{}{"Email" : user.Email}

    return getDBSingleton().UpdateSchema(to, where)
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
    currentUser := GetCurrentUser(r)
    userList, err := getAllUsers(currentUser)

    var userJson []byte
    if err == nil {
        userJson, err = json.Marshal(userList)
    }

    if err == nil {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, string(userJson))
    } else {
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprint(w, err)
    }
}

func handleGetUser(w http.ResponseWriter, r *http.Request) {
    reqEmail := mux.Vars(r)["Email"]
    reqUser, err := GetUser(reqEmail)

    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    currentUser := GetCurrentUser(r)

    if !currentUser.CanViewUser(reqUser) {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    userInfo := struct {
        AuthLevel int
        Permissions Permission
    }{
        reqUser.AuthLevel, reqUser.Permissions,
    }

    userJson, err := json.Marshal(userInfo)

    if err == nil {
        w.WriteHeader(http.StatusOK)
        fmt.Fprint(w, string(userJson))
    } else {
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprint(w, err)
    }
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request) {
    var err error = nil
    var code int = http.StatusOK

    defer func() { 
        w.WriteHeader(code)
        if err != nil {
            fmt.Fprint(w, err) 
        }
    }()

    reqEmail := mux.Vars(r)["Email"]
    reqUser, err := GetUser(reqEmail)

    if err != nil {
        code = http.StatusNotFound
        return
    }

    currentUser := GetCurrentUser(r)

    if !currentUser.CanViewUser(reqUser) {
        code = http.StatusNotFound
        return
    }

    if !currentUser.CanModifyUser(reqUser) {
        code = http.StatusUnauthorized
        return
    }

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        code = http.StatusInternalServerError
        return
    }

    values, code := parseUserUpdateRequest(currentUser, reqUser, reqBody)
    if code != http.StatusOK {
        return
    }

    where := map[string]interface{}{"Email" : reqEmail}
    err = getDBSingleton().UpdateSchema(values, where)

    if err != nil {
        code = http.StatusInternalServerError
        return
    }
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
    var err error = nil
    var code int = http.StatusOK

    defer func() { 
        w.WriteHeader(code)
        if err != nil {
            fmt.Fprint(w, err) 
        }
    }()

    currentUser := GetCurrentUser(r)

    if currentUser.AuthLevel < CreateUserAuthLevel {
        code = http.StatusUnauthorized
        return
    }

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        code = http.StatusInternalServerError
        return
    }

    var userInfo struct {
        Email string
        Password string
    }

    err = json.Unmarshal(reqBody, &userInfo)
    if err != nil {
        code = http.StatusInternalServerError
        return
    }

    salt := GenerateSalt()
    saltedPassword := SaltPassword(userInfo.Password, salt)

    err = CreateUser(userInfo.Email, salt, saltedPassword)
    if err != nil {
        code = http.StatusBadRequest
        return
    }
}

func getAllUsers(currentUser *User) ([]string, error) {
    opts := databases.SelectOptions{}
    cols := []string{"Email", "AuthLevel"}
    userRows, err := getDBSingleton().SelectSchema(cols, nil, opts)

    if err != nil {
        return nil, err
    }

    list := make([]string, 0)
    var user User

    for userRows.Next() {
        err = userRows.Scan(&user)

        if err != nil {
            return nil, err
        }

        if currentUser.CanViewUser(&user) {
            list = append(list, user.Email)
        }
    }

    return list, nil
}

func parseUserUpdateRequest(curUser, modUser *User, updateJSON []byte) (map[string]interface{}, int) {
    
    updates := struct {
        AuthLevel int           `json:omitempty`
        Password string         `json:omitempty`
        Beacons map[string]int  `json:omitempty`
    }{
        AuthLevel: modUser.AuthLevel, 
        Password: modUser.Password,
    }

    err := json.Unmarshal(updateJSON, &updates)
    if err != nil {
        fmt.Println(err)
        return nil, http.StatusInternalServerError
    }

    updateValues := make(map[string]interface{})

    if updates.AuthLevel != modUser.AuthLevel {
        if updates.AuthLevel < DefaultAuthLevel {
            return nil, http.StatusBadRequest
        }

        if updates.AuthLevel > curUser.AuthLevel {
            return nil, http.StatusUnauthorized
        }

        updateValues["AuthLevel"] = updates.AuthLevel
    }
    
    if updates.Password != modUser.Password {
        updateValues["Password"] = SaltPassword(updates.Password, modUser.Salt)
    }

    updateValues["Permissions"] = modUser.Permissions

    if  updates.Beacons != nil {
        for beacon, level := range updates.Beacons {

            permitted := curUser.CanModifyBeacon(beacon) && 
                level <= curUser.GetAuthLevel("Beacons", beacon)

            if permitted {
                modUser.SetAuthLevel("Beacons", beacon, level)
            } else {
                return nil, http.StatusUnauthorized
            }
        }
    }    

    return updateValues, http.StatusOK
}