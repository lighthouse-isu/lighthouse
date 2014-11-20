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

package local

import (
    "os"
    "fmt"
    "testing"
    "net"
    "net/http"
    "net/http/httptest"

    "github.com/stretchr/testify/assert"
)


func Test_IsApplicable_With_ENV(t *testing.T) {
    os.Setenv("DOCKER_HOST", "tcp://192.168.59.103:2375")

    assert.True(t, Provider.IsApplicable(),
        "Local provider should always be applicable, when $DOCKER_HOST is exists.")
}

func Test_IsApplicable_Without_ENV(t *testing.T) {
    os.Setenv("DOCKER_HOST", "")

    assert.False(t, Provider.IsApplicable(),
        "Local provider should always be applicable, when $DOCKER_HOST is exists.")
}

func Test_GetVMs(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "OK")
    }))
    defer ts.Close()

    // Address info changes nearly each test run for test server
    mockAddress, mockPort, _ := net.SplitHostPort(ts.Listener.Addr().String())
    mockURL := fmt.Sprintf("tcp://%s:%s", mockAddress, mockPort)

    os.Setenv("DOCKER_HOST", mockURL)
    vms := Provider.GetVMs()

    assert.NotEmpty(t, vms,
        "Local provider should have at least one element in response")

    assert.Equal(t, vms[0].Address, mockAddress,
        "Local provider did not read the Docker address correctly.")

    assert.Equal(t, vms[0].Port, mockPort,
        "Local provider did not read the Docker address port correctly.")

    assert.Equal(t, vms[0].Version, "v1",
        "Local provider did not label the vm version correctly.")

    assert.True(t, vms[0].CanAccessDocker,
        "Local provider should have properly detected the mock boot2docker server")
}
