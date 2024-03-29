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
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/beacons"
	"github.com/lighthouse/lighthouse/beacons/aliases"
	"github.com/lighthouse/lighthouse/databases"
	"github.com/lighthouse/lighthouse/databases/postgres"
	"github.com/lighthouse/lighthouse/handlers"
	"github.com/lighthouse/lighthouse/handlers/applications"
	"github.com/lighthouse/lighthouse/handlers/docker"
	"github.com/lighthouse/lighthouse/session"

	"github.com/lighthouse/lighthouse/logging"

	"github.com/gorilla/mux"
)

const (
	API_VERSION_0_2 = "/api/v0.2"
)

var databasesReload = flag.Bool("databases-reload", false, "Start all databases from empty if true")
var databasesDriver = flag.String("databases-driver", "postgres", "Type of database to connect to")

func ServeIndex(w http.ResponseWriter, r *http.Request) {
	authData := struct {
		LoggedIn bool
		Email    string
	}{
		session.GetValueOrDefault(r, "auth", "logged_in", false).(bool),
		session.GetValueOrDefault(r, "auth", "email", "").(string),
	}

	var indexPath string
	if _, err := os.Stat("/config/auth.json"); os.IsNotExist(err) {
		indexPath = "static/index.html"
	} else {
		indexPath = "/static/index.html"
	}

	indexTemplate, err := template.New("index.html").ParseFiles(indexPath)
	if err == nil {
		err = indexTemplate.Execute(w, authData)
	}

	if err != nil {
		handlers.WriteError(w, 404, "lighthouse", err.Error())
	}
}

func init() {
	flag.Parse()

	connections := map[string]func() databases.DBInterface{
		"postgres": postgres.Connection,
	}

	if connFunc, ok := connections[*databasesDriver]; ok {
		databases.SetDefaultConnection(connFunc())
	} else {
		fmt.Fprintf(os.Stderr, "Driver '%s' not supported\n", *databasesDriver)
		os.Exit(-1)
	}

	auth.Init(*databasesReload)
	beacons.Init(*databasesReload)
	aliases.Init(*databasesReload)
	applications.Init(*databasesReload)
}

func main() {
	logging.Info("Starting...")

	baseRouter := mux.NewRouter()

	baseRouter.HandleFunc("/", ServeIndex).Methods("GET")
	baseRouter.NotFoundHandler = http.HandlerFunc(ServeIndex)

	staticServer := http.FileServer(http.Dir("static"))
	baseRouter.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", staticServer))

	versionRouter := baseRouter.PathPrefix(API_VERSION_0_2).Subrouter()

	docker.Handle(versionRouter.PathPrefix("/d").Subrouter())
	beacons.Handle(versionRouter.PathPrefix("/beacons").Subrouter())
	aliases.Handle(versionRouter.PathPrefix("/aliases").Subrouter())
	applications.Handle(versionRouter.PathPrefix("/applications").Subrouter())
	auth.Handle(versionRouter)

	ignoreURLs := []string{
		"/",
		"/login",
		fmt.Sprintf("%s/login", API_VERSION_0_2),
		fmt.Sprintf("%s/logout", API_VERSION_0_2),
	}

	app := auth.AuthMiddleware(baseRouter, ignoreURLs)
	app = logging.Middleware(app)

	http.Handle("/", app)

	logging.Info("Ready...")

	http.ListenAndServe(":5000", nil)
}
