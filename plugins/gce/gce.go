package gce

import (
    "os"
    "fmt"
    "net/http"
    "io/ioutil"
    "encoding/json"

    "github.com/gorilla/mux"

    "code.google.com/p/goauth2/oauth"
    compute "code.google.com/p/google-api-go-client/compute/v1"
)


var oauthConfig *oauth.Config

type DiscoveredVM struct {
    Name string
    Address string
    CanAccessDocker bool
}

func GetCurrentProjectID() (string, error) {
    request, _ := http.NewRequest(
        "GET", "http://metadata.google.internal/computeMetadata/v1/project/project-id", nil)

    request.Header.Add("Metadata-Flavor", "Google")

    client := http.Client{}
    response, err := client.Do(request)
    if err != nil {
        return "", err
    }

    projectID, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return "", err
    }
    response.Body.Close()

    return string(projectID), nil
}

func DiscoverVMs(authCode string) {
    transport := &oauth.Transport{
        Config: oauthConfig,
    }
    transport.Exchange(authCode)

    computeClient, _ := compute.New(transport.Client())

    projectName, err := GetCurrentProjectID()
    if err != nil {
        return
    }

    zones, _ := computeClient.Instances.AggregatedList(projectName).Do()

    var discoveredVMs []*DiscoveredVM

    for _, zone := range zones.Items {
        for _, instance := range zone.Instances {
            // For future reference we need figure out which network interface
            // to use instead of deafulting to the first one.
            network := instance.NetworkInterfaces[0]

            discoveredVMs = append(discoveredVMs, &DiscoveredVM{
                Name: instance.Name,
                Address: network.NetworkIP,
                CanAccessDocker: false,
            })
            
        }
    }

    for _, vm := range discoveredVMs {
        address := fmt.Sprintf("http://%s:2375/v1/_ping", vm.Address)
        resp, err := http.Get(address)

        if err == nil {
            body, _ := ioutil.ReadAll(resp.Body)
            vm.CanAccessDocker = string(body) == "OK"
            resp.Body.Close()
        }
    }

    data, _ := json.Marshal(discoveredVMs)
    ioutil.WriteFile("./plugins/gce/vms.json", data, 0664)
}

func AuthUrl() string {
    return oauthConfig.AuthCodeURL("")
}

func LoadGCEConfig() *oauth.Config {
    var fileName string
    if _, err := os.Stat("/config/plugins/gce.json"); os.IsNotExist(err) {
        fileName = "./config/plugins/gce.json"
    } else {
        fileName = "/config/plugins/gce.json"
    }
    configFile, _ := ioutil.ReadFile(fileName)

    var config oauth.Config
    json.Unmarshal(configFile, &config)
    return &config
}

func Handle(r *mux.Router) {
    oauthConfig = LoadGCEConfig()

    r.HandleFunc("/vms", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")

        vms, err := ioutil.ReadFile("./plugins/gce/vms.json")
        if err != nil {
            vms, _ = json.Marshal([]interface{}{})
        }

        fmt.Fprintf(w, "%s", vms)
    }).Methods("GET")

    r.HandleFunc("/vms/find", func(w http.ResponseWriter, r *http.Request) {
        go DiscoverVMs(r.FormValue("code"))
        http.Redirect(w, r, "/plugins/gce/vms", 302)
    }).Methods("GET")

    r.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, AuthUrl(), 302)
    }).Methods("GET")
}