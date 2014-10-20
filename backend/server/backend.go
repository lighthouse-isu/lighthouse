package main

import (
    "net/http"
    "github.com/gorilla/mux"
    "./handlers"
)

/*
    Sets endpoints for the server and begins listening for requests
*/
func main() {
    baseRouter    := mux.NewRouter()
    versionRouter := baseRouter.PathPrefix("/api/v0.1").Subrouter()
    hostRouter    := versionRouter.PathPrefix("/{host}")
    dockerRouter  := hostRouter.PathPrefix("/d").Methods("GET", "POST", "PUT", "DELETE").Subrouter()

    // The regex ignores /'s that would normally break the url
    dockerRouter.HandleFunc("/{dockerURL:.*}", DockerHandler)

    http.Handle("/", baseRouter)
    // Can use http.ListenAndServeTLS to enforce HTTPS once we have certificates
    http.ListenAndServe(":5000", nil)
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
