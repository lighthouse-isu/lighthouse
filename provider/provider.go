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

package provider

import (
    "fmt"
    "net/http"
    "encoding/json"

    "github.com/ngmiller/lighthouse/provider/models"

    "github.com/ngmiller/lighthouse/provider/providers/gce"
    "github.com/ngmiller/lighthouse/provider/providers/local"
    "github.com/ngmiller/lighthouse/provider/providers/unknown"

    "github.com/gorilla/mux"
)


func DecideProvider(providers []*models.Provider) *models.Provider {
    for _, provider := range providers {
        if provider.IsApplicable() {
            return provider
        }
    }
    return unknown.Provider
}

func Handle(r *mux.Router) {
    selectedProvider := DecideProvider([]*models.Provider{
        gce.Provider,
        local.Provider,
    })

    fmt.Printf("Detected provider is %s....\n", selectedProvider.Name)

    r.HandleFunc("/vms", func(w http.ResponseWriter, r *http.Request) {
        vms := selectedProvider.GetVMs()
        response, _ := json.Marshal(vms)

        fmt.Fprintf(w, "%s", response)
    }).Methods("GET")

    r.HandleFunc("/which", func(w http.ResponseWriter, r *http.Request) {
        response, _ := json.Marshal(selectedProvider.Name)
        fmt.Fprintf(w, "%s", response)
    }).Methods("GET")
}
