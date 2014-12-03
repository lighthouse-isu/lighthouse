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
    "github.com/lighthouse/lighthouse/provider"
    "github.com/lighthouse/lighthouse/handlers"
    "github.com/lighthouse/lighthouse/users"

    "github.com/lighthouse/lighthouse/logging"

    "github.com/gorilla/mux"
)

const (
    API_VERSION_0_1 = "/api/v0.1"
)

func ServeIndex(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "static/index.html")
}


func main() {

    logging.Info("Starting...")
    baseRouter := mux.NewRouter()

    baseRouter.HandleFunc("/", ServeIndex).Methods("GET")
    baseRouter.NotFoundHandler =  http.HandlerFunc(ServeIndex)

    staticServer := http.FileServer(http.Dir("static"))
    baseRouter.PathPrefix("/static/").Handler(
        http.StripPrefix("/static/", staticServer))

    versionRouter := baseRouter.PathPrefix(API_VERSION_0_1).Subrouter()

    hostRouter    := versionRouter.PathPrefix("/{Host}")
    dockerRouter := hostRouter.PathPrefix("/d").Methods("GET", "POST", "PUT", "DELETE").Subrouter()
    dockerRouter.HandleFunc("/{DockerURL:.*}", handlers.DockerHandler)

    provider.Handle(versionRouter.PathPrefix("/provider").Subrouter())

    auth.Handle(versionRouter)

    users.Handle(versionRouter.PathPrefix("/users").Subrouter())

    ignoreURLs := []string{
        "/",
        "/login",
        fmt.Sprintf("%s/login", API_VERSION_0_1),
        fmt.Sprintf("%s/logout", API_VERSION_0_1),
    }

    app := auth.AuthMiddleware(baseRouter, ignoreURLs)
    app = logging.Middleware(app)

    http.Handle("/", app)

    logging.Info("Ready...")

    http.ListenAndServe(":5000", nil)
}
