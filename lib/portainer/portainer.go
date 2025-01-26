package portainer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Client struct {
	Scheme     string
	Host       string
	ApiKey     string
	EndpointId string
}

type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type StackDataFile struct {
	Id int `json:"Id"`
}

func GetStackId(path string) (int, error) {
	data, err := os.ReadFile(filepath.Join(path, ".stack"))
	if err != nil {
		return 0, fmt.Errorf("Failed to read portainer ID: %w", err)
	}
	d := new(StackDataFile)
	if err := json.Unmarshal(data, d); err != nil {
		return 0, fmt.Errorf("Failed to unmarshal .stack file: %w", err)
	}
	return d.Id, nil
}

func (cli *Client) newUrl(path string, query ...string) *url.URL {
	if len(query)%2 != 0 {
		panic("Bad use of client.newUrl: variadic args should be even")
	}
	u := new(url.URL)
	if cli.Scheme != "" {
		u.Scheme = cli.Scheme
	} else {
		u.Scheme = "https"
	}
	u.Host = cli.Host
	u.Path = path

	v := url.Values{}
	for i := 0; i < len(query)/2; i += 2 {
		v.Add(query[i], query[i+1])
	}
	u.RawQuery = v.Encode()
	return u
}

func ReadEnvironment(envFile io.Reader) ([]EnvironmentVariable, error) {
	scanner := bufio.NewScanner(envFile)
	kvs := make([]EnvironmentVariable, 0)
	for scanner.Scan() {
		keyVal := strings.Split(scanner.Text(), "=")
		if len(keyVal) != 2 {
			return nil, fmt.Errorf("Encountered invalid key val pair: %v", keyVal)
		}
		kvs = append(kvs, EnvironmentVariable{
			Name:  keyVal[0],
			Value: keyVal[1],
		})
	}
	return kvs, nil
}
