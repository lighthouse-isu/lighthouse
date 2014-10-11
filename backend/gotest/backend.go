package main

import (
    "fmt"
    "net/http"
    "encoding/json"
    "strings"
    "strconv"
    "github.com/fsouza/go-dockerclient"
    "github.com/gorilla/mux"
)

// For testing purposes.  Connects to localhost
func GetDefaultClient() (*docker.Client, error) {
  return GetClientInstanceForIP("unix:///var/run/docker.sock",
        CertificateList{CertPem: "", KeyPem: "", CaPem: ""})
}

// Setting handles for the various Docker commands
func main() {
    v := "/api/v0.1"

    r := mux.NewRouter()
    r.HandleFunc(v + "/", GetInfo).Methods("GET")
    r.HandleFunc(v + "/images", GetImages).Methods("GET")
    r.HandleFunc(v + "/containers", GetContainers).Methods("GET")
    r.HandleFunc(v + "/containers/create", CreateContainer).Methods("POST")
    r.HandleFunc(v + "/containers/create/{name}", CreateContainer).Methods("POST")
    r.HandleFunc(v + "/containers/stop/{id}", StopContainer).Methods("POST")
    r.HandleFunc(v + "/containers/remove/{id}", RemoveContainer).Methods("DELETE")

    http.Handle("/", r)
    // Can use http.ListenAndServeTLS to enforce HTTPS once we have certificates
    http.ListenAndServe(":5000", nil)
}

type DockerInfo struct {
    Client      map[string]string
    Images      []docker.APIImages
    Containers  []docker.APIContainers
}

/*
    Writes JSON of general client information including Docker
    system information, images, and running containers
*/
func GetInfo(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    client, err := GetDefaultClient()
    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    env, _ := client.Info() // json can't automatically marshal client
    clientInfo := make(map[string]string)
    for _, s := range *env {
        kv := strings.Split(s, "=")
        clientInfo[kv[0]] = kv[1]
    }

    images, _ := client.ListImages(false)
    containers, _ := client.ListContainers(docker.ListContainersOptions{ All: false, Size: false })

    dockerInfo := DockerInfo{
        Client: clientInfo, Images: images, Containers: containers,
        }

    j, err := json.MarshalIndent(dockerInfo, "", "  ")

    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    fmt.Fprint(w, string(j))
}

type ImageInfo struct {
    Tags []string
    Image docker.Image
}

/*
    Writes JSON of an array of in-depth Image information
*/
func GetImages(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    client, err := GetDefaultClient()

    apiImages, err := client.ListImages(false)

    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }
    if len(apiImages) == 0 {
        fmt.Fprint(w, "{}")
        return
    }

    infos := make([]ImageInfo, len(apiImages))

    for i, apiImage := range apiImages {
        image, err := client.InspectImage(apiImage.ID)

        if err != nil {
            continue
        }

        infos[i] = ImageInfo{Tags: apiImage.RepoTags, Image: *image}
    }

    j, err := json.MarshalIndent(infos, "", "  ")

    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    fmt.Fprint(w, "{\n",
                  "  \"Images\":\"", string(j), "\"\n",
                  "}")
}

/*
    Writes JSON of an array of in-depth Container information
*/
func GetContainers(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    client, err := GetDefaultClient()

    options := docker.ListContainersOptions{ All: true, Size: false }
    apiContainers, err := client.ListContainers(options)

    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }
    if len(apiContainers) == 0 {
        fmt.Fprint(w, "{}")
        return
    }

    containers := make([]docker.Container, len(apiContainers))

    for i, apiContainer := range apiContainers {
        container, err := client.InspectContainer(apiContainer.ID)

        if err != nil {
            continue
        }

        containers[i] = *container
    }

    j, err := json.MarshalIndent(containers, "", "  ")

    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    fmt.Fprint(w, "{\n",
                  "  \"Containers\":\"", string(j), "\"\n",
                  "}")
}

/*
    Creates a new container as specified by "Payload" in the POST data
*/
func CreateContainer(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    client, err := GetDefaultClient()

    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    r.ParseForm()
    var config docker.Config
    payload := r.PostForm["Payload"]

    if len(payload) == 0 {
        fmt.Fprint(w, "{\"Error\": \"No Docker Payload in POST\" }")
        return
    }

	err = json.Unmarshal([]byte(payload[0]), &config)
    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    name := mux.Vars(r)["name"] // If "name" was not given, sets to ""
    options := docker.CreateContainerOptions{Name: name, Config: &config}

    container, err := client.CreateContainer(options)
    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    j, err := json.MarshalIndent(container, "", "  ")
    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    fmt.Fprint(w, "{\n",
                  "  \"Container\":\"", string(j), "\"\n",
                  "}")
}

/*
    Stops the Container "test_container".  Stopping does not remove the Container, so it will also
    need to be removed before CreateContainer is called. (Future work will allow the Container to restart)

    QUERY PARAMS:
        t: optional timeout in seconds (default = 10)
*/
func StopContainer(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    client, err := GetDefaultClient()

    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    r.ParseForm()

    var timeout uint
    tStr := GetFormParameterOrDefault(r.Form, "t", "10")
    t64, err := strconv.ParseUint(tStr, 10, 32)
    if err != nil {
        timeout = 10
    } else {
        timeout = uint(t64)
    }

    id := mux.Vars(r)["id"]

    err = client.StopContainer(id, uint(timeout))
    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    // Doocker returns nothing on success. I could just
    // write something to return if we want, otherwise we can
    // assume that no errors means success.
}

/*
    Removes the Container "test_container" from the connected machine.  If the
    Container hasn't been stopped yet this will forcibly stop and remove.

    QUERY PARAMS:
        v: optional remove volumes flag (default = false)
        force: optional force remove (kill) flag (default = false)
*/
func RemoveContainer(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    client, err := GetDefaultClient()

    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    r.ParseForm()

    var rmVols bool
    vStr := GetFormParameterOrDefault(r.Form, "v", "0")
    rmVols, err = strconv.ParseBool(vStr)
    if err != nil {
        rmVols = false
    }

    var force bool
    fStr := GetFormParameterOrDefault(r.Form, "force", "0")
    rmVols, err = strconv.ParseBool(fStr)
    if err != nil {
        force = false
    }

    id := mux.Vars(r)["id"]

    options := docker.RemoveContainerOptions{ID: id, RemoveVolumes: rmVols, Force: force}

    err = client.RemoveContainer(options)
    if err != nil {
        fmt.Fprint(w, "{\"Error\":", err, "}")
        return
    }

    // Doocker returns nothing on success. I could just
    // write something to return if we want, otherwise we can
    // assume that no errors means success.
}

// Creates and authenticates a new connection to a Docker client
func GetClientInstanceForIP(ip string, certificates CertificateList) (*docker.Client, error) {
    client, err := docker.NewVersionedClient(ip, "1.12")
    if err != nil {
        return nil, err
    }

    client.HTTPClient.Transport = CreateDockerTLS(certificates)
    return client, nil
}

// Returns the value of the first instance of key. Returns def if key is not in the form
func GetFormParameterOrDefault(form map[string][]string, key string, def string) string {
    if len(form[key]) > 0 {
        return form[key][0]
    } else {
        return def
    }
}
