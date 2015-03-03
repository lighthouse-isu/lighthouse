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

package main

import (
    "fmt"
    "net/http"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/beacons"
    "github.com/lighthouse/lighthouse/beacons/aliases"
    "github.com/lighthouse/lighthouse/users"
    "github.com/lighthouse/lighthouse/handlers/docker"

    "github.com/lighthouse/lighthouse/logging"

    "github.com/gorilla/mux"
)

const (
    API_VERSION_0_2 = "/api/v0.2"
)

func ServeIndex(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "static/index.html")
}

func main() {

    logging.Info("Starting...")

    users.Init()
    beacons.Init()
    aliases.Init()

    baseRouter := mux.NewRouter()

    baseRouter.HandleFunc("/", ServeIndex).Methods("GET")
    baseRouter.NotFoundHandler =  http.HandlerFunc(ServeIndex)

    staticServer := http.FileServer(http.Dir("static"))
    baseRouter.PathPrefix("/static/").Handler(
        http.StripPrefix("/static/", staticServer))

    versionRouter := baseRouter.PathPrefix(API_VERSION_0_2).Subrouter()

    docker.Handle(versionRouter.PathPrefix("/d").Subrouter())
    beacons.Handle(versionRouter.PathPrefix("/beacons").Subrouter())
    aliases.Handle(versionRouter.PathPrefix("/aliases").Subrouter())
    auth.Handle(versionRouter)

    ignoreURLs := []string{
        "/",
        "/login",
        fmt.Sprintf("%s/login", API_VERSION_0_2),
        fmt.Sprintf("%s/logout", API_VERSION_0_2),
        fmt.Sprintf("%s/whoami", API_VERSION_0_2),
    }

    app := auth.AuthMiddleware(baseRouter, ignoreURLs)
    app = logging.Middleware(app)

    http.Handle("/", app)

    logging.Info("Ready...")

    http.ListenAndServe(":5000", nil)
}
