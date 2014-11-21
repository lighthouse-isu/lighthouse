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

package handlers

import (
    "fmt"
    "net/http"
    "io/ioutil"
    "bytes"

    "github.com/lighthouse/lighthouse/session"
    "github.com/lighthouse/lighthouse/users/permissions"
)

/*
    Forwards the given request on to the Docker client.  Data stored
    in the request's 'Payload' field is also forwarded.

    Will only write to the given ResponseWriter on success.

    RETURN: nil on succes.  A non-nil *HandlerError on failure
*/
func DockerRequestHandler(w http.ResponseWriter, info HandlerInfo) *HandlerError {
    url := "http://" + info.Host + "/" + info.DockerEndpoint

    payload := []byte(nil)
    if info.Body != nil {
        payload = []byte(info.Body.Payload)
    }

    method := info.Request.Method

    req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
    if err != nil {
        return &HandlerError{500, "control", "Failed to create " + method + " request"}
    }

    // TODO - better client
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return &HandlerError{500, "control", method + " request failed"}
    }

    // Close body after return
    defer resp.Body.Close()

    if resp.StatusCode > 299 {
        return &HandlerError{resp.StatusCode, "docker", resp.Status}
    }

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return &HandlerError{500, "control", "Failed reading response body"}
    }

    w.WriteHeader(resp.StatusCode)
    w.Write(body)
    return nil
}

/*
    Handles all requests through the Docker endpoint.  Calls all
    relevant custom handlers and then passes request on to Docker.

    If an error occurs in a custom handler or with the Docker request
    itself, the custom handlers will be instructed to rollback.
*/
func DockerHandler(w http.ResponseWriter, r *http.Request) {
    // Ready all HTTP form data for the handlers
    r.ParseForm()

    info := GetHandlerInfo(r)

    reqAllowed := false

    email := session.GetValueOrDefault(r, "auth", "email", "").(string)
    perms, dbErr := permissions.GetPermissions(email)

    fmt.Print(perms)

    if dbErr != nil {
        WriteError(w, HandlerError{401, "control", "unknown user"})
        return
    }

    for host, _ := range perms.Providers {
        if host == info.Host {
            reqAllowed = true
            break
        }
    }

    if !reqAllowed {
        WriteError(w, HandlerError{401, "control",
            fmt.Sprintf("user not permitted to access host: %s", info.Host),
        })
        return
    }

    var customHandlers = CustomHandlerMap{
        //regexp.MustCompile("example"): ExampleHandler,
    }

    runCustomHandlers, err := RunCustomHandlers(info, customHandlers)

    // On success, send request to Docker
    if err == nil {
        err = DockerRequestHandler(w, info)
    }

    // On error, rollback
    if err != nil {
        Rollback(w, *err, info, runCustomHandlers)
    }
}
