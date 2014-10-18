package main

import (
    "net/http"
    "encoding/json"
    "github.com/gorilla/mux"
    "./handlers"
)

/*
    Sets endpoints for the server and begins listening for requests
*/
func main() {
    baseRouter    := mux.NewRouter()
    versionRouter := baseRouter.PathPrefix("/api/v0.1").Subrouter()

    // TODO - Host may contain /'s, need a cleaner way to support this
    hostRouter    := versionRouter.PathPrefix("/__{Host:.*}__")
    dockerRouter  := hostRouter.PathPrefix("/d").Methods("GET", "POST", "PUT", "DELETE").Subrouter()

    // The regex allows the handler to take Docker endpoint.
    // Otherwise it would get cut at a /
    dockerRouter.HandleFunc("/{DockerURL:.*}", DockerHandler)

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

    vars := mux.Vars(r)
    url := vars["DockerURL"]

    // Ready all HTTP form data for the handlers
    r.ParseForm()

    runCustomHandlers, err := handlers.RunCustomHandlers(r, url)

    // On success, send request to Docker
    if err == nil {
        err = handlers.DockerRequestHandler(w, r)
    }

    // On error, rollback
    if err != nil {
        WriteError(w, err)
        for _, handler := range runCustomHandlers {
            handler(r, true)
        }
    }
}

/*
    Writes error data and code to the HTTP response.
*/
func WriteError(w http.ResponseWriter, err *handlers.HandlerError) {
    json, _ := json.Marshal(struct {
        Error string
        Message string
    }{err.Cause, err.Message})

    w.WriteHeader(err.StatusCode)
    w.Write(json)
}
