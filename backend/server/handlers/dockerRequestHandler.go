package handlers

import (
    "net/http"
    "io/ioutil"
    "bytes"
    "github.com/gorilla/mux"
)

/*
    Forwards the given request on to the Docker client.  Data stored
    in the request's 'Payload' field is also forwarded.

    RETURN: true on success, false otherwise
*/
func DockerRequestHandler(w http.ResponseWriter, r *http.Request) *HandlerError {
    vars := mux.Vars(r)
    host := vars["Host"]
    url := "http://" + host + "/" + vars["DockerURL"]

    payload := r.Form["Payload"]
    var payloadBody []byte

    if len(payload) > 0 {
        payloadBody = []byte(payload[0])
    } else {
        payloadBody = nil
    }

    req, err := http.NewRequest(r.Method, url, bytes.NewBuffer(payloadBody))
    if err != nil {
        return &HandlerError{500, "control", "Failed to create " + r.Method + " request"}
    }

    // TODO - better client
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return &HandlerError{500, "control", r.Method + " request failed" + host}
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
