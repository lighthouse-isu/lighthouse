package main

import (
    "strings"
    "github.com/fsouza/go-dockerclient"
)

func GetClientInfo(client *docker.Client) (map[string]string, error) {
    info, err := client.Info()

    if err != nil {
        return nil, err
    }

    return info.Map(), nil
}

func GetImageInfo(client *docker.Client) (map[string]map[string]string, error) {
   	
    allInfo := make(map[string]map[string]string)
    apiImages, err := client.ListImages(false)
    if err != nil {
        return nil, err
    }

    for _, apiImage := range apiImages {
        imgTagPair := strings.Split(apiImage.RepoTags[0], ":")
        imgName, imgTag := imgTagPair[0], imgTagPair[1]

        image, err := client.InspectImage(apiImage.ID)
        if err != nil {
            continue
        }

        thisInfo := make(map[string]string)
        allInfo[imgName] = thisInfo

        thisInfo["Tag"] = imgTag
        thisInfo["ID"] = image.ID[0:10]
        thisInfo["Parent"] = image.Parent[0:10]
        thisInfo["Comment"] = image.Comment
        thisInfo["Container"] = image.Container[0:10]
    }

    return allInfo, nil
}

func GetContainerInfo(client *docker.Client) (map[string]map[string]string, error) {

    allInfo := make(map[string]map[string]string)
    options := docker.ListContainersOptions{ All: true, Size: false }

    apiContainers, err := client.ListContainers(options)

    if err != nil {
        return nil, err
    }

    for _, apiContainer := range apiContainers {

        container, err := client.InspectContainer(apiContainer.ID)
        if err != nil {
            continue
        }

        thisInfo := make(map[string]string)
        allInfo[container.Name] = thisInfo

        thisInfo["ID"] = container.ID[0:10]
        thisInfo["State"] = container.State.String()
        thisInfo["Image"] = container.Image[0:10]
        thisInfo["Path"] = container.Path
    }

    return allInfo, nil
}

