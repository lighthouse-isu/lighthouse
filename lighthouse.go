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
    "time"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/provider"
    "github.com/lighthouse/lighthouse/handlers"

    "github.com/gorilla/mux"
)


func LoggingMiddleware(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        h.ServeHTTP(w, r)

        end := time.Now()
        latency := end.Sub(start)
        fmt.Printf("%s %12s %s %s\n",
            end.Format("2006/01/02 - 15:04:05"), latency, r.Method, r.URL)
    })
}

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
    runCustomHandlers, err := handlers.RunCustomHandlers(info)

    // On success, send request to Docker
    if err == nil {
        err = handlers.DockerRequestHandler(w, info)
    }

    // On error, rollback
    if err != nil {
        handlers.Rollback(w, err, info, runCustomHandlers)
    }
}


func main() {
    fmt.Println("Starting...")
    baseRouter := mux.NewRouter()

    baseRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "static/index.html")
    }).Methods("GET")

    staticServer := http.FileServer(http.Dir("static"))
    baseRouter.PathPrefix("/static/").Handler(
        http.StripPrefix("/static/", staticServer))

    auth.Handle(baseRouter)

    versionRouter := baseRouter.PathPrefix("/api/v0.1").Subrouter()
    hostRouter    := versionRouter.PathPrefix("/{host}")
    dockerRouter  := hostRouter.PathPrefix("/d").Methods("GET", "POST", "PUT", "DELETE").Subrouter()
    // The regex ignores /'s that would normally break the url
    dockerRouter.HandleFunc("/{dockerURL:.*}", DockerHandler)

    provider.Handle(baseRouter.PathPrefix("/provider").Subrouter())

    ignoreURLs := []string{"/", "/login", "/logout"}
    app := auth.AuthMiddleware(baseRouter, ignoreURLs)
    app = LoggingMiddleware(app)

    http.Handle("/", app)
    fmt.Println("Ready...")
    http.ListenAndServe(":5000", nil)
}
