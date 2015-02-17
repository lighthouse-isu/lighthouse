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
    "os"
    "fmt"
    "strings"
    "io/ioutil"
    "net/http"

    "crypto/sha512"
    "crypto/rand"

    "encoding/hex"
    "encoding/json"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/session"
)


var SECRET_HASH_KEY string

func SaltPassword(password, salt string) string {
    key := fmt.Sprintf("%s:%s:%s", password, salt, SECRET_HASH_KEY)

    sha := sha512.New()
    sha.Write([]byte(key))

    return hex.EncodeToString(sha.Sum(nil))
}

func GenerateSalt() string {
    salt := make([]byte, 16)
    rand.Read(salt)
    return hex.EncodeToString(salt)
}

type LoginForm struct {
    Email string
    Password string
}

type AuthConfig struct {
    Admins []User
    SecretKey string
}

func LoadAuthConfig() *AuthConfig{
    var fileName string
    if _, err := os.Stat("/config/auth.json"); os.IsNotExist(err) {
        fileName = "./config/auth.json"
    } else {
        fileName = "/config/auth.json"
    }
    configFile, _ := ioutil.ReadFile(fileName)

    var config AuthConfig
    json.Unmarshal(configFile, &config)
    return &config
}

func AuthMiddleware(h http.Handler, ignorePaths []string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.HasPrefix(r.URL.Path, "/static") {
            h.ServeHTTP(w, r)
            return
        }

        for _, path := range ignorePaths {
            if path == r.URL.Path {
                h.ServeHTTP(w, r)
                return
            }
        }

        if loggedIn := session.GetValueOrDefault(r, "auth", "logged_in", false).(bool); loggedIn {
            h.ServeHTTP(w, r)
            return
        }

        http.Redirect(w, r, "/login",  http.StatusFound)
    })
}

func Handle(r *mux.Router) {
    config := LoadAuthConfig()

    SECRET_HASH_KEY = config.SecretKey

    for _, admin := range config.Admins {
        salt := GenerateSalt()
        saltedPassword := SaltPassword(admin.Password, salt)

        createUserWithAuthLevel(admin.Email, salt, saltedPassword, admin.AuthLevel)
    }

    r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
        loginForm := &LoginForm{}
        body, _ := ioutil.ReadAll(r.Body)
        json.Unmarshal(body, &loginForm)

        var userOK, passwordOK bool

        user, err := GetUser(loginForm.Email)
        userOK = err == nil

        if userOK {
            password := SaltPassword(loginForm.Password, user.Salt)
            passwordOK = password == user.Password

            session.SetValue(r, "auth", "logged_in", passwordOK)
        }

        if passwordOK {
            session.SetValue(r, "auth", "email", user.Email)
        }

        session.Save("auth", r, w)

        if userOK && passwordOK {
            w.WriteHeader(200)
        } else {
            w.WriteHeader(401)
        }

        fmt.Fprintf(w, "%t", userOK && passwordOK)
    }).Methods("POST")

    r.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
        session.SetValue(r, "auth", "logged_in", false)
        session.Save("auth", r, w)
    }).Methods("GET")

    userRoute := r.PathPrefix("/users").Subrouter()

    userRoute.HandleFunc("/list", handleListUsers).Methods("PUT")

    userRoute.HandleFunc("/{Email}", handleGetUser).Methods("GET")

    userRoute.HandleFunc("/{Email}", handleUpdateUser).Methods("PUT")

    userRoute.HandleFunc("/create", handleCreateUser).Methods("POST")
}
