package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mr55p-dev/app-utils/config"
	"github.com/mr55p-dev/app-utils/lib/portainer"
	"github.com/mr55p-dev/gonk"
)

type CliConfig struct {
	PortainerHost       string
	PortainerToken      string
	PortainerEndpointId string
}

type StackDataFile struct {
	Id int `json:"Id"`
}

func iter(basePath string, cli *portainer.Client) {
	configFile, err := config.NewAppConfig(basePath)
	if err != nil {
		panic(err)
	}
	name := configFile.App
	composeFile, err := os.Open(filepath.Join(basePath, "docker-compose.yml"))
	if err != nil {
		panic(err)
	}
	defer composeFile.Close()
	envFile, err := os.Open(filepath.Join(basePath, "stack.env"))
	if err != nil {
		panic(err)
	}
	defer envFile.Close()
	envPairs, err := portainer.ReadEnvironment(envFile)
	if err != nil {
		panic(err)
	}

	stackId := getStackId(basePath)
	if stackId == 0 {
		fmt.Println("No existing stack found. Creating a new one.")
		res, err := cli.CreateStack(name, composeFile, envPairs)
		if err != nil {
			panic(err)
		}
		fmt.Println("Created stack", name, "with id", res.Id)
		data, err := json.Marshal(res)
		err = os.WriteFile(filepath.Join(basePath, ".stack"), data, 0o644)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("Updating existing stack with id", stackId)
		res, err := cli.UpdateStack(stackId, composeFile, envPairs)
		if err != nil {
			panic(err)
		}
		fmt.Println("Updated stack", res)
	}
}

func main() {
	config := new(CliConfig)
	if err := gonk.LoadConfig(config, gonk.EnvLoader("")); err != nil {
		panic("Failed to load portainer data from env")
	}

	cli := &portainer.Client{
		Scheme:     "https",
		Host:       config.PortainerHost,
		ApiKey:     config.PortainerToken,
		EndpointId: config.PortainerEndpointId,
	}

	for _, basePath := range flag.Args() {
		iter(basePath, cli)
	}

}

func getStackId(path string) int {
	data, err := os.ReadFile(filepath.Join(path, ".stack"))
	if err != nil {
		return 0
	}
	d := new(StackDataFile)
	if err := json.Unmarshal(data, d); err != nil {
		return 0
	}
	return d.Id
}
