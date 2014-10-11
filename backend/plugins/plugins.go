package plugins

import (
    "./gce"

    "github.com/gorilla/mux"
)

func Handle(r *mux.Router) {
    gce.Handle(r.PathPrefix("/gce").Subrouter())
}
