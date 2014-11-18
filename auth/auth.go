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
    "github.com/gorilla/sessions"
    "github.com/gorilla/securecookie"

    "github.com/lighthouse/lighthouse/users"

    _ "github.com/bmizerany/pq"
)


var SECRET_HASH_KEY string
var CookieStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(32))

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
    Admins []users.User
    SecretKey string
}

func GetValueOrDefault(r *http.Request, key string, def interface{}) interface{} {
    session, _ := CookieStore.Get(r, "auth")
    val, ok := session.Values[key]
    if val != nil && ok {
        return val
    }
    return def
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

        session, _ := CookieStore.Get(r, "auth")

// REMOVE ME
session.Values["logged_in"] = true
session.Values["email"] = "admin@gmail.com"

        if loggedIn, ok := session.Values["logged_in"].(bool); loggedIn && ok {
            h.ServeHTTP(w, r)
        } else {
            w.WriteHeader(http.StatusUnauthorized)
            fmt.Fprintf(w, "unauthorized")
        }
    })
}

func Handle(r *mux.Router) {
    config := LoadAuthConfig()

    SECRET_HASH_KEY = config.SecretKey

    for _, admin := range config.Admins {
        salt := GenerateSalt()
        saltedPassword := SaltPassword(admin.Password, salt)

        users.CreateUser(admin.Email, salt, saltedPassword)
    }

    r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
        session, _ := CookieStore.Get(r, "auth")

        loginForm := &LoginForm{}
        body, _ := ioutil.ReadAll(r.Body)
        json.Unmarshal(body, &loginForm)

        user := users.GetUser(loginForm.Email)
        password := SaltPassword(loginForm.Password, user.Salt)
        session.Values["logged_in"] = password == user.Password

        session.Save(r, w)

        fmt.Fprintf(w, "%t", session.Values["logged_in"].(bool))
    }).Methods("POST")


    r.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
        session, _ := CookieStore.Get(r, "auth")
        session.Values["logged_in"] = false
        session.Save(r, w)
    }).Methods("GET")
}
