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

package docker

import (
    "fmt"
    "net/http"
    "bytes"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/session"
    "github.com/lighthouse/lighthouse/beacons"
    "github.com/lighthouse/lighthouse/handlers"
)

/*
    Forwards the given request on to the Docker client.  Data stored
    in the request's 'Payload' field is also forwarded.

    Will only write to the given ResponseWriter on success.

    RETURN: nil on succes.  A non-nil *handlers.HandlerError on failure
*/
func DockerRequestHandler(w http.ResponseWriter, info handlers.HandlerInfo) *handlers.HandlerError {
    email := session.GetValueOrDefault(info.Request, "auth", "email", "").(string)
    beaconAddress, err := beacons.GetBeaconAddress(info.Host)

    requestIsToBeacon := err == nil
    
    var targetAddress, targetEndpoint string

    if requestIsToBeacon {
        targetAddress = beaconAddress
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
        return &handlers.HandlerError{500, "control", "Failed to create " + method + " request"}
    }

    if requestIsToBeacon {
        token, _ := beacons.GetBeaconToken(beaconAddress, email)
        req.Header.Set(beacons.HEADER_TOKEN_KEY, token)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return &handlers.HandlerError{500, "control", method + " request failed"}
    }

    // Close body after return
    defer resp.Body.Close()

    if resp.StatusCode > 299 {
        return &handlers.HandlerError{resp.StatusCode, "docker", resp.Status}
    }

    w.WriteHeader(resp.StatusCode)
    var bodyBuffer = make([]byte, 16)

    for {
        n, err := resp.Body.Read(bodyBuffer)
        w.Write(bodyBuffer[:n])

        if err != nil {
            break
        }
    }

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

    info, ok := handlers.GetHandlerInfo(r)

    if !ok {
        handlers.WriteError(w, handlers.HandlerError {
            http.StatusBadRequest, 
            "control", "could not get required data for handler",
        })
        return
    }

    var customHandlers = handlers.CustomHandlerMap{
        //regexp.MustCompile("example"): ExampleHandler,
    }

    runCustomHandlers, err := handlers.RunCustomHandlers(info, customHandlers)

    // On success, send request to Docker
    if err == nil {
        err = DockerRequestHandler(w, info)
    }

    // On error, rollback
    if err != nil {
        handlers.Rollback(w, *err, info, runCustomHandlers)
    }
}

func Handle(r *mux.Router) {
    r.HandleFunc("/{Endpoint:.*}", DockerHandler)
}
