package main

import (
    "fmt"
    "net/http"
    "net/url"
    "strconv"
)

func main() {
    // Note, the usual calls to stop and remove will use IDs instead
    // of names, but this way I know exactly what it was called.

    var status int

    fmt.Printf("\nStarting CreateTest...\n")
    status = CreateTest("test_container")
    if status != 200 {
        panic(status)
    }
    fmt.Printf("   ...success!\n")

    fmt.Printf("\nStarting StopTest...\n")
    status = StopTest("test_container", 10)
    if status != 200 {
        panic(status)
    }
    fmt.Printf("   ...success!\n")

    fmt.Printf("\nStarting RemoveTest...\n")
    status = RemoveTest("test_container", false, false)
    if status != 200 {
        panic(status)
    }
    fmt.Printf("   ...success!\n")

    fmt.Printf("\nAll tests succeeded\n\n")
}

func CreateTest(name string) int {

    var jsonStr =
     `{
         "Hostname":"",
         "Domainname": "",
         "User":"",
         "Memory":0,
         "MemorySwap":0,
         "CpuShares": 512,
         "Cpuset": "0,1",
         "AttachStdin":false,
         "AttachStdout":true,
         "AttachStderr":true,
         "PortSpecs":null,
         "Tty":false,
         "OpenStdin":false,
         "StdinOnce":false,
         "Env":null,
         "Cmd":[
                 "date"
         ],
         "Image":"ubuntu:latest",
         "Volumes":{
                 "/tmp": {}
         },
         "WorkingDir":"",
         "NetworkDisabled": false,
         "ExposedPorts":{
                 "22/tcp": {}
         }
     }`

    values := make(url.Values)
    values.Set("Payload", jsonStr)

    dest := "http://localhost:5000/api/v0.1/containers/create"

    if name != "" {
        dest = dest + "/" + name
    }

    resp, err := http.PostForm(dest, values)

    if err != nil {
        return 0
    }
    defer resp.Body.Close()

    return resp.StatusCode
}

func StopTest(id string, t uint64) int {
    dest := "http://localhost:5000/api/v0.1/containers/stop/" + id
    dest = string(strconv.AppendUint([]byte(dest + "?t="), t, 10))

    resp, err := http.PostForm(dest, make(url.Values))

    if err != nil {
        return 0
    }
    defer resp.Body.Close()

    return resp.StatusCode
}

func RemoveTest(id string, v, f bool) int {
    dest := "http://localhost:5000/api/v0.1/containers/remove/" + id
    dest = string(strconv.AppendBool([]byte(dest + "?v="), v))
    dest = string(strconv.AppendBool([]byte(dest + "&force="), f))

    req, err := http.NewRequest("DELETE", dest, nil)
    if err != nil {
        return 0
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return 0
    }
    defer resp.Body.Close()

    return resp.StatusCode
}
