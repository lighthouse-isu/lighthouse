package handlers

import (
    "net/http"
    "io/ioutil"
    "bytes"
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
