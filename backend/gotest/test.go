package main

import (
    "fmt"
    "net/http"
    "github.com/fsouza/go-dockerclient"
)

func init() {
    http.HandleFunc("/", GetInfo)
    http.HandleFunc("/create", MakeContainer)
    http.HandleFunc("/stop", StopContainer)
    http.HandleFunc("/remove", RemoveContainer)
}

func GetInfo(w http.ResponseWriter, r *http.Request) {

    client, err := GetDefaultClient()
    if err != nil {
        fmt.Fprint(w, "Error retrieving client: ", err)
        return
    }

    fmt.Fprint(w, "<html><body>")

    DisplayClientInfo(client, w)
    DisplayImageInfo(client, w)
    DisplayContainerInfo(client, w)

    fmt.Fprint(w, "</body></html>")
}

func MakeContainer(w http.ResponseWriter, r *http.Request) {
	
    client, err := GetDefaultClient()
    if err != nil {
        fmt.Fprint(w, "Error retrieving client: ", err)
        return
    }

    containers, _ := GetContainerInfo(client)
    if _,ok := containers["test_container"]; ok {
        fmt.Fprint(w, "A container has already been made. For testing we can only make one.")
        return
    }

    pullConfig := docker.PullImageOptions{Repository: "themagicalkarp/docker-thing"}
    auth := docker.AuthConfiguration{Username: "", Password: "", Email: ""}
    client.PullImage(pullConfig, auth)

    port := docker.Port("5000/tcp")
    portMap := map[docker.Port]struct{}{port: struct{}{}}

    config := docker.Config{Image: "themagicalkarp/docker-thing", Cmd: []string {"python", "app.py"}, ExposedPorts: portMap}
    createConfig := docker.CreateContainerOptions{Name: "test_container", Config: &config}

    _, err = client.CreateContainer(createConfig)

    if err != nil {
        fmt.Fprint(w, "There was an error making the container: ", err)
        return
    }

    fmt.Fprint(w, "Container \"test_container\" created successfully.")
}

func StopContainer(w http.ResponseWriter, r *http.Request) {
	
    client, err := GetDefaultClient()
    if err != nil {
        fmt.Fprint(w, "Error retrieving client: ", err)
        return
    }

    err = client.StopContainer("test_container", 15)

    if err != nil {
        fmt.Fprintf(w, "The was an error stopping the container: ", err)
        return
    }

    fmt.Fprintf(w, "The container was stopped successfully.")
}

func RemoveContainer(w http.ResponseWriter, r *http.Request) {

    client, err := GetDefaultClient()
    if err != nil {
        fmt.Fprint(w, "Error retrieving client: ", err)
        return
    }

    opts := docker.RemoveContainerOptions{ID: "test_container", Force: true}
    err = client.RemoveContainer(opts)

    if err != nil {
        fmt.Fprintf(w, "The was an error removing the container: ", err)
        return
    }

    fmt.Fprintf(w, "The container was removed successfully!")
}

func GetDefaultClient() (*docker.Client, error) {
    return GetClientInstanceForIP("unix:///var/run/docker.sock", 
        CertificateList{CertPem: "", KeyPem: "", CaPem: ""})
}

func GetClientInstanceForIP(ip string, certificates CertificateList) (*docker.Client, error) {
    client, err := docker.NewClient(ip)
    if err != nil {
        return nil, err
    }

    client.HTTPClient.Transport = CreateDockerTLS(certificates)
    return client, nil
}

func DisplayClientInfo(client *docker.Client, w http.ResponseWriter) {

    fmt.Fprint(w, "<div><h1>Client</h1>")      

    info, err := GetClientInfo(client)

    if err != nil {
        fmt.Fprint(w, "<h2>Error retrieving client data: ", err, "</h2></div>")
        return
    }

    fmt.Fprint(w, "<table>")

    for k, v := range info {
        fmt.Fprint(w, "<tr><td>", k, "</td><td>", v, "</td></tr>")
    }

    fmt.Fprint(w, "</table></div>")
}

func DisplayImageInfo(client *docker.Client, w http.ResponseWriter) {
    
    fmt.Fprint(w, "<div><h1>Images</h1>")    

    allImages, err := GetImageInfo(client)

    if err != nil {
        fmt.Fprint(w, "<h2>Error retrieving images: ", err, "</h2></div>")
        return
    }
    if len(allImages) == 0 {
        fmt.Fprint(w, "<h2>No images loaded</h2></div>")
        return
    }

    for _, image := range allImages {
        fmt.Fprint(w, "<table><h2><tr><td>Name</td>")
        for key := range image {
            fmt.Fprint(w, "<td>", key, "</td>")
        }
        fmt.Fprint(w, "</tr>")
        break
    }

    for name, image := range allImages {
        fmt.Fprint(w, "<tr><td>", name, "</td>")
        
        for _, v := range image {
            fmt.Fprint(w, "<td>", v, "</td>")
        }

        fmt.Fprint(w, "</tr>")
    }

    fmt.Fprint(w, "</table></div>")
}

func DisplayContainerInfo(client *docker.Client, w http.ResponseWriter) {

    fmt.Fprint(w, "<div><h1>Containers</h1>")
    allContainers, err := GetContainerInfo(client)

    if err != nil {
        fmt.Fprint(w, "<h2>Error retrieving containers: ", err, "</h2></div>")
        return
    }
    if len(allContainers) == 0 {
        fmt.Fprint(w, "<h2>No containers running</h2></div>")
        return
    }

    for _, container := range allContainers {
        fmt.Fprint(w, "<table><h2><tr><td>Name</td>")
        for key := range container {
            fmt.Fprint(w, "<td>", key, "</td>")
        }
        fmt.Fprint(w, "</tr>")
        break
    }

    for name, container := range allContainers {
        fmt.Fprint(w, "<tr><td>", name, "</td>")

        for _, v := range container {
            fmt.Fprint(w, "<td>", v, "</td>")
        }

        fmt.Fprint(w, "</tr>")
    }

    fmt.Fprint(w, "</table></div>")
}

