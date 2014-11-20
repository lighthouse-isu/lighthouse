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
    "testing"
    "net/http"
    "net/http/httptest"
    "encoding/json"

    "github.com/lighthouse/lighthouse/provider/providers/gce"
    "github.com/lighthouse/lighthouse/provider/providers/local"
    "github.com/lighthouse/lighthouse/provider/providers/unknown"

    "github.com/lighthouse/lighthouse/provider/models"

    "github.com/stretchr/testify/assert"
    "github.com/gorilla/mux"

)

func NotApplicable() bool {
    return false
}

func Applicable() bool {
    return true
}

var FAKE_VM_A = &models.VM {
    Name: "A",
    Address: "1.2.3.4",
    Port: "42",
    Version: "v9000",
    CanAccessDocker: false,
}

var FAKE_VM_B = &models.VM {
    Name: "B",
    Address: "123.123.123.123",
    Port: "2222",
    Version: "v1",
    CanAccessDocker: true,
}

func GetSomeVMs() []*models.VM {
    return []*models.VM{FAKE_VM_A, FAKE_VM_B}
}


var FakeProvider = &models.Provider {
    Name: "fake provider",
    IsApplicable: Applicable,
    GetVMs: GetSomeVMs,
}


func Test_DecideProvider(t *testing.T) {
    a := &models.Provider {
        IsApplicable: NotApplicable,
    }

    b := &models.Provider {
        IsApplicable: NotApplicable,
    }

    c := &models.Provider {
        IsApplicable: Applicable,
    }

    d := &models.Provider {
        IsApplicable: NotApplicable,
    }


    selectedProvider := DecideProvider([]*models.Provider{
        a, b, c, d,
    })

    assert.Equal(t, c, selectedProvider,
        "DecideProvider choose the wrong provider.")
}


func Test_DecideProvider_unknown(t *testing.T) {
    a := &models.Provider {
        IsApplicable: NotApplicable,
    }

    b := &models.Provider {
        IsApplicable: NotApplicable,
    }

    c := &models.Provider {
        IsApplicable: NotApplicable,
    }

    d := &models.Provider {
        IsApplicable: NotApplicable,
    }

    selectedProvider := DecideProvider([]*models.Provider{
        a, b, c, d,
    })

    assert.Equal(t, unknown.Provider, selectedProvider,
        "DecideProvider should have choosen the unknown provider.")
}

func HandleWithSpecificProvider(r *mux.Router, provider *models.Provider) *mux.Router {
    // this helper method forces the provider handler
    // to choose a specified provider

    // patch all the providers with the specified provider
    // to ensure our specified provider is chosen
    gceProvider, localProvider := gce.Provider, local.Provider
    gce.Provider, local.Provider = provider, provider

    // setup routing
    Handle(r)

    // resote patched providers
    gce.Provider, local.Provider = gceProvider, localProvider
    return r
}

func Test_WhichRequest(t *testing.T) {
    mockRouter := HandleWithSpecificProvider(mux.NewRouter(), FakeProvider)

    w := httptest.NewRecorder()
    r, _ := http.NewRequest("GET", "http://localhost/which", nil)
    mockRouter.ServeHTTP(w, r)

    assert.Equal(t, w.Code, 200,
        "Provider 'which' request did not reutrn expected status code")

    expectedResult, _ := json.Marshal("fake provider")
    assert.Equal(t, w.Body.String(), string(expectedResult),
        "Provider 'which' request did not reutrn expected predicted provider")
}


func Test_VMRequest(t *testing.T) {
    mockRouter := HandleWithSpecificProvider(mux.NewRouter(), FakeProvider)

    w := httptest.NewRecorder()
    r, _ := http.NewRequest("GET", "http://localhost/vms", nil)
    mockRouter.ServeHTTP(w, r)

    assert.Equal(t, w.Code, 200,
        "Provider 'vms' request did not reutrn expected status code")

    expectedResult, _ := json.Marshal([]*models.VM{FAKE_VM_A, FAKE_VM_B})
    assert.Equal(t, w.Body.String(), string(expectedResult),
        "Provider 'vms' request did not reutrn expected predicted provider")
}
