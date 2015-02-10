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
    "strings"
    "net/http"
    "net/url"

    "github.com/lighthouse/lighthouse/auth"
    "github.com/lighthouse/lighthouse/provider"
    "github.com/lighthouse/lighthouse/handlers"
    "github.com/lighthouse/lighthouse/beacons"

    "github.com/lighthouse/lighthouse/logging"

    "github.com/zenazn/goji/web"
    "github.com/zenazn/goji/web/middleware"
)

const (
    API_VERSION_0_2 = "/api/v0.2"
)

func ServeIndex(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "static/index.html")
}

func AttachSubrouterTo(m *web.Mux, route string, handle func(m *web.Mux)) *web.Mux {
    sub := web.New()
    m.Handle(fmt.Sprintf("%s/*", route), sub)
    sub.Use(middleware.SubRouter)
    handle(sub)
    sub.Compile()
    return sub
}

func QueryParamExtract(c *web.C, h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        params := strings.Split(r.RequestURI, "/")
        for i, param := range params[3:] {
            param, _ = url.QueryUnescape(param)
            c.Env[fmt.Sprintf("%d", i)] = param
        }
        h.ServeHTTP(w, r)
    })
}

func main() {

    logging.Info("Starting...")

    baseRouter := web.New()
    baseRouter.Use(auth.AuthMiddleware)
    baseRouter.Use(logging.Middleware)

    baseRouter.NotFound(ServeIndex)

    staticServer := http.FileServer(http.Dir("static"))
    baseRouter.Get("/static/*", staticServer)

    versionRouter := web.New()
    baseRouter.Handle(API_VERSION_0_2 + "/*", versionRouter)
    versionRouter.Use(middleware.SubRouter)
    versionRouter.Use(middleware.EnvInit)
    versionRouter.Use(QueryParamExtract)

    AttachSubrouterTo(versionRouter, "/d", handlers.Handle)
    AttachSubrouterTo(versionRouter, "/provider", provider.Handle)
    AttachSubrouterTo(versionRouter, "/beacons", beacons.Handle)

    ignoreURLs := []string{
        "/",
        "/login",
        fmt.Sprintf("%s/login", API_VERSION_0_2),
        fmt.Sprintf("%s/logout", API_VERSION_0_2),
    }
    auth.Handle(versionRouter)
    auth.Ignore(ignoreURLs)

    versionRouter.Compile()
    baseRouter.Compile()
    http.Handle("/", baseRouter)

    logging.Info("Ready...")

    http.ListenAndServe(":5000", nil)
}
