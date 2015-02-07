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

    "github.com/zenazn/goji/web"

    "github.com/lighthouse/lighthouse/session"
    "github.com/lighthouse/lighthouse/beacons"
)

/*
    Forwards the given request on to the Docker client.  Data stored
    in the request's 'Payload' field is also forwarded.

    Will only write to the given ResponseWriter on success.

    RETURN: nil on succes.  A non-nil *HandlerError on failure
*/
func DockerRequestHandler(w http.ResponseWriter, info HandlerInfo) *HandlerError {
    email := session.GetValueOrDefault(info.Request, "auth", "email", "").(string)
    beaconInstance, err := beacons.GetBeacon(info.Host)

    requestIsToBeacon := err == nil

    var targetAddress, targetEndpoint string

    if requestIsToBeacon {
        targetAddress = beaconInstance.Address
        targetEndpoint = fmt.Sprintf("d/%s/%s", info.Host, info.DockerEndpoint)
    } else {
        targetAddress = info.Host
        targetEndpoint = info.DockerEndpoint
    }

    url := fmt.Sprintf("http://%s/%s", targetAddress, targetEndpoint)

    if info.Request.URL.RawQuery != "" {
        url += "?" + info.Request.URL.RawQuery
    }

    payload := []byte(nil)
    if info.Body != nil {
        payload = []byte(info.Body.Payload)
    }

    method := info.Request.Method

    req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
    if err != nil {
        return &HandlerError{500, "control", "Failed to create " + method + " request"}
    }

    if requestIsToBeacon && beaconInstance.Users[email] {
        req.Header.Set(beacons.HEADER_TOKEN_KEY, beaconInstance.Token)
    }

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
func DockerHandler(c web.C, w http.ResponseWriter, r *http.Request) {
    // Ready all HTTP form data for the handlers
    r.ParseForm()

    info := GetHandlerInfo(c, r)

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
