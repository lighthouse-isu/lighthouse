package main

import (
    "./plugins"

    "net/http"

    "github.com/gorilla/mux"
)


func main() {
    r := mux.NewRouter()

    plugins.Handle(r.PathPrefix("/plugins").Subrouter())

    http.Handle("/", r)
    http.ListenAndServe(":5000", nil)
}