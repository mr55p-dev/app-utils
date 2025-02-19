package compose

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
	projects := make([]ListEntry, 0)
	outBytes, err := command("/", "ls")
	if err != nil {
		return nil, fmt.Errorf("Error running compose ls: %w", err)
	}
	if err := json.Unmarshal(outBytes, &projects); err != nil {
		return nil, fmt.Errorf("Error unmarshalling output: %w", err)
	}
	return projects, nil
}

func (c *Client) Ps(path string) ([]PsEntry, error) {
	output, err := command(path, "ps")
	if err != nil {
		return nil, fmt.Errorf("Error running docker compose ps: %w", err)
	}

	containers := make([]PsEntry, 0)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		proc := PsEntry{}
		if err := json.Unmarshal(scanner.Bytes(), &proc); err != nil {
			return nil, fmt.Errorf("Error unmarshalling output: %w", err)
		}
		containers = append(containers, proc)
	}
	return containers, nil
}

func (c *Client) Up(path string) error {
	_, err := command(path, "up", "-d")
	if err != nil {
		return fmt.Errorf("Error running compose up: %w", err)
	}
	return nil
}
