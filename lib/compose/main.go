package compose

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Client struct {
	dir string
}

type ListEntry struct {
	Name        string `json:"Name"`
	Status      string `json:"Status"`
	ConfigFiles string `json:"ConfigFiles"`
}

func New(root string) (*Client, error) {
	stat, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("Invalid root for FSClient: %w", err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	return &Client{dir: root}, nil
}

type PsEntry struct {
	Name   string `json:"Name"`
	Status string `json:"Status"`
}

func (c *Client) List() ([]ListEntry, error) {
	outBytes := new(bytes.Buffer)
	projects := make([]ListEntry, 0)
	cmd := exec.Command("docker", "compose", "ls", "--format", "json")
	cmd.Stdout = outBytes
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Error running docker compose: %w", err)
	}
	if err := json.Unmarshal(outBytes.Bytes(), &projects); err != nil {
		return nil, fmt.Errorf("Error unmarshalling output: %w", err)
	}
	return projects, nil
}

func (c *Client) Ps(app string) ([]PsEntry, error) {
	outBytes := new(bytes.Buffer)
	errBytes := new(bytes.Buffer)
	cmd := exec.Command("docker", "compose", "ps", "--format", "json")
	cmd.Dir = filepath.Join(c.dir, app)
	cmd.Stdout = outBytes
	cmd.Stderr = errBytes
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Error (%s): %s", err, errBytes.String())
	}

	containers := make([]PsEntry, 0)
	scanner := bufio.NewScanner(outBytes)
	for scanner.Scan() {
		proc := PsEntry{}
		if err := json.Unmarshal(scanner.Bytes(), &proc); err != nil {
			return nil, fmt.Errorf("Error unmarshalling output: %w", err)
		}
		containers = append(containers, proc)
	}
	return containers, nil
}
