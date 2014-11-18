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
    "time"
    "net/http"
)

type VM struct {
    Name string `json:"name,omitempty"`
    Address string `json:"address,omitempty"`
    Port string `json:"port,omitempty"`
    Version string `json:"version,omitempty"`
    CanAccessDocker bool `json:"canAccessDocker,omitempty"`
}

type Provider struct {
    Name string
    IsApplicable func() bool
    GetVMs func(email string) []*VM
}

func PingDocker(vm *VM) bool {
    pingAddress := fmt.Sprintf("http://%s:%s/%s/_ping",
        vm.Address, vm.Port, vm.Version)

    client := &http.Client{
        Timeout: time.Duration(2)*time.Second,
    }
    _, err := client.Get(pingAddress)
    return err == nil
}
