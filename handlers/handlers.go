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
    "regexp"
    "strings"
    "net/http"
    "net/url"
    "encoding/json"
    "io/ioutil"

    "github.com/gorilla/mux"

    "github.com/lighthouse/lighthouse/beacons/aliases"
)

/*
    ---------------------------------------------------------------------
      THE CUSTOM HANDLER INTERFACE
    ---------------------------------------------------------------------

    *** To add a new handler, add it to the map in RunCustomHandlers ***

      A custom handler will:
    ---------------------------------------------------------------------
      - retrieve all data it requires from the request given to it

      - perform any operations neccessary to rollback changes it has
        made for a perticular request

      - perform its own error detection and recovery

      - rollback and return a non-nil *HandlerError if a failure occurs

      - return nil upon successful completion

      A custom handler may safely assume:
    ---------------------------------------------------------------------
      - r.ParseForm() will have been called prior to calling the handler

      - it will not be asked to rollback if it has not been called
        already for a particular request

      - it will not be asked to rollback if it caused an unrecoverable
        error for a particular request (i.e. initiated a rollback)

      - its error message will be returned to the client during failure
*/

type CustomHandlerFunc func(info HandlerInfo, rollback bool) (*HandlerError)

type CustomHandlerMap map[*regexp.Regexp]CustomHandlerFunc

/*
    Container of failure data created by the handlers
*/
type HandlerError struct {
    StatusCode  int
    Cause       string
    Message     string
}

/*
    Container of common data handlers will need to avoid
    re-extracting for every handler.
*/
type HandlerInfo struct {
    DockerEndpoint  string
    Host            string
    Body            *RequestBody
    Request         *http.Request
    HandlerData     map[string]interface{}
}

/*
    The body of POST and PUT requests will need to be very well
    defined as this struct needs to match it completely.

    The fields of the body should be a subset of the fields of this
    struct i.e. all fields are optional in the body, but all
    fields that appear in the body must be declared in this struct.
*/
type RequestBody struct {
    Payload string
}

/*
    Searches for any applicable custom handlers for the given request.

    RETURN: A list of custom handlers which have run (for rollback)
            an non-nil *HandlerError on failure, nil otherwise
*/
func RunCustomHandlers(info HandlerInfo, handlers CustomHandlerMap) ([]CustomHandlerFunc, *HandlerError) {

    runHandlers := []CustomHandlerFunc{}

    for exp, handler := range handlers {
        if exp.MatchString(info.DockerEndpoint) {
            if res := handler(info, false); res != nil {
                return runHandlers, res
            }
            runHandlers = append(runHandlers, handler)
        }
    }

    return runHandlers, nil
}

/*
    Performs a handler rollback by instructing each handler which
    was run to rollback its operation and writing the failure
    report to be returned to the client.
*/
func Rollback(
    w http.ResponseWriter,
    err HandlerError,
    info HandlerInfo,
    runHandlers []CustomHandlerFunc,
) {
    WriteError(w, err)
    for _, handler := range runHandlers {
        handler(info, true)
    }
}

/*
    Writes error data and code to the HTTP response.
*/
func WriteError(w http.ResponseWriter, err HandlerError) {
    json, _ := json.Marshal(struct {
        Error   string
        Message string
    }{err.Cause, err.Message})

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(err.StatusCode)
    w.Write(json)
}

/*
    Extracts data from the request to create a HandlerInfo
    which is used by the handlers.

    RETURN: A HandlerInfo extracted from the request
*/
func GetHandlerInfo(r *http.Request) (HandlerInfo, bool) {
    var info HandlerInfo

    params, ok := GetEndpointParams(r, []string{"Host", "DockerEndpoint"})

    if ok == false || len(params) < 2 {
        return HandlerInfo{}, false
    }

    hostAlias := params["Host"]

    value, err := aliases.GetAddressOf(hostAlias)
    if err == nil {
        info.Host = value
    } else {
        info.Host = hostAlias // Unknown alias, just use what was given
    }

    info.DockerEndpoint = params["DockerEndpoint"]
    info.Body = GetRequestBody(r)
    info.Request = r

    return info, true
}

/*
    Retrieves the body of the request as a *ReqestBody

    RETURN: nil if no body or on error. A *ReqestBody otherwise.
*/
func GetRequestBody(r *http.Request) *RequestBody {
    if r.Body == nil {
        return nil
    }

    reqBody, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return nil
    }

    var body RequestBody
    json.Unmarshal(reqBody, &body)

    return &body
}

/*
    Retrieves the parameters of a generic endpoint scheme.  The endpoint to 
    extract MUST be given as `mux.Vars(r)["Endpoint"]`. A map keyed by fields in the
    given array is returned.  If there are more keys than fields in the endpoint, the 
    remaining keys are ignored.  If there are more fields than keys, the last key will
    hold the entire remaining endpoint.

    RETURN: A map keyed on the given fields and true on success, nil and false otherwise
*/
func GetEndpointParams(r *http.Request, names []string) (map[string]string, bool) {
    endpoint, ok := mux.Vars(r)["Endpoint"]

    if !ok {
        return nil, false
    }

    params := make(map[string]string, len(names))

    uri := r.RequestURI[len(r.URL.Path) - len(endpoint):]
    parts := strings.SplitN(uri, "/", len(names))

    for i, part := range parts {
        param, err := url.QueryUnescape(part)
        
        if err != nil {
            return nil, false
        }

        params[names[i]] = param
    }

    return params, true
}
