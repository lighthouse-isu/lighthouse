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

    "github.com/lighthouse/lighthouse/logging"

    "github.com/gorilla/mux"
)


/*
    Handles all requests through the Docker endpoint.  Calls all
    relevant custom handlers and then passes request on to Docker.

    If an error occurs in a custom handler or with the Docker request
    itself, the custom handlers will be instructed to rollback.
*/
func DockerHandler(w http.ResponseWriter, r *http.Request) {
    // Ready all HTTP form data for the handlers
    r.ParseForm()

    info := handlers.GetHandlerInfo(r)

    var customHandlers = handlers.CustomHandlerMap{
        //regexp.MustCompile("example"): ExampleHandler,
    }

    runCustomHandlers, err := handlers.RunCustomHandlers(info, customHandlers)

    // On success, send request to Docker
    if err == nil {
        err = handlers.DockerRequestHandler(w, info)
    }

    // On error, rollback
    if err != nil {
        handlers.Rollback(w, *err, info, runCustomHandlers)
    }
}

const (
    API_VERSION_0_1 = "/api/v0.1"
)

func main() {

    logging.Info("Starting...")
    baseRouter := mux.NewRouter()

    baseRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "static/index.html")
    }).Methods("GET")

    staticServer := http.FileServer(http.Dir("static"))
    baseRouter.PathPrefix("/static/").Handler(
        http.StripPrefix("/static/", staticServer))


    versionRouter := baseRouter.PathPrefix(API_VERSION_0_1).Subrouter()
    hostRouter    := versionRouter.PathPrefix("/{Host}")
    dockerRouter  := hostRouter.PathPrefix("/d").Methods("GET", "POST", "PUT", "DELETE").Subrouter()
    dockerRouter.HandleFunc("/{DockerURL:.*}", DockerHandler)

    provider.Handle(versionRouter.PathPrefix("/provider").Subrouter())

    auth.Handle(versionRouter)

    ignoreURLs := []string{
        "/",
        fmt.Sprintf("%s/login", API_VERSION_0_1),
        fmt.Sprintf("%s/logout", API_VERSION_0_1),
    }

    app := auth.AuthMiddleware(baseRouter, ignoreURLs)
    app = logging.Middleware(app)

    http.Handle("/", app)

    logging.Info("Ready...")

    http.ListenAndServe(":5000", nil)
}
