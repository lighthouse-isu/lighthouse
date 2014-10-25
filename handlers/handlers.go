package handlers

import (
    "net/http"
    "regexp"
    "encoding/json"
    "io/ioutil"
    "github.com/gorilla/mux"
    "github.com/ngmiller/lighthouse/hosts"
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

type customHandlerFunc func(info HandlerInfo, rollback bool) (*HandlerError)

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
}

/*
    The body of POST and PUT requests will need to be very well
    defined as this struct needs to match it completely.

    The fields of the body should be a subset of the fields of this
    struct i.e. all fields are optional in the body, but all
    fields that appear in the body must be declared in this struct.
*/
type RequestBody struct {
    Payload string `json:"payload,omitempty"`
    App     string `json:"app,omitempty"`
}

/*
    Searches for any applicable custom handlers for the given request.

    RETURN: A list of custom handlers which have run (for rollback)
            an non-nil *HandlerError on failure, nil otherwise
*/
func RunCustomHandlers(info HandlerInfo) ([]customHandlerFunc, *HandlerError) {

    // Could make this global - more memory overhead, less latency
    customHandlers := map[*regexp.Regexp]customHandlerFunc{
        //regexp.MustCompile("example"): ExampleHandler,
    }

    runHandlers := []customHandlerFunc{}

    for exp, handler := range customHandlers {
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
    err *HandlerError,
    info HandlerInfo,
    runHandlers []customHandlerFunc,
) {
    WriteError(w, err)
    for _, handler := range runHandlers {
        handler(info, true)
    }
}

/*
    Writes error data and code to the HTTP response.
*/
func WriteError(w http.ResponseWriter, err *HandlerError) {
    json, _ := json.Marshal(struct {
        Error   string  `json:"error"`
        Message string  `json:"message"`
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
func GetHandlerInfo(r *http.Request) HandlerInfo {
    vars := mux.Vars(r)
    var info HandlerInfo

    info.Host = hosts.AliasLookup(vars["host"])
    info.DockerEndpoint = vars["dockerURL"]
    info.Body = GetRequestBody(r)
    info.Request = r

    return info
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

    var body *RequestBody
    json.Unmarshal(reqBody, body)

    return body
}
