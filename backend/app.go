package main

import (
    "fmt"
    "net/http"
    "time"

    "./auth"
    "./plugins"

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


func main() {
    fmt.Println("Starting...")
    r := mux.NewRouter()

    auth.Handle(r)

    plugins.Handle(r.PathPrefix("/plugins").Subrouter())

    ignoreURLs := []string{"/", "/login", "/logout", "/plugins/gce/vms/find"}
    app := auth.AuthMiddleware(r, ignoreURLs)

    http.Handle("/", LoggingMiddleware(app))

    fmt.Println("Ready...")
    http.ListenAndServe(":5000", nil)
}
