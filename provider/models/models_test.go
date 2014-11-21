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

package models

import (
    "fmt"
    "testing"
    "net"
    "net/http"
    "net/http/httptest"

    "github.com/stretchr/testify/assert"
)


func Test_PingDocker(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/v1/_ping" {
            fmt.Fprintln(w, "OK")
        } else {
            w.WriteHeader(500)
        }
    }))
    defer ts.Close()
    mockAddress, mockPort, _ := net.SplitHostPort(ts.Listener.Addr().String())

    assert.True(t, PingDocker(&VM{
        Address: mockAddress,
        Port: mockPort,
        Version: "v1",
    }), "Ping Docker did not send the correct request to Docker")

    assert.False(t, PingDocker(&VM{
        Address: "123.456.789.111",
        Port: "1234",
        Version: "v9000",
    }), "What.... how did you get that to work?")
}
