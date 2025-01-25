package portainer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type StackCreateResponse struct {
	Id      int    `json:"Id"`
	Message string `json:"Message"`
}

func (cli *Client) CreateStack(name string, composeFile io.Reader, environment []EnvironmentVariable) (*StackCreateResponse, error) {
	buf := new(bytes.Buffer)
	fw := multipart.NewWriter(buf)
	formComposeFile, err := fw.CreateFormFile("file", "docker-compose.yml")
	if err != nil {
		return nil, fmt.Errorf("Failed to create env compose file in form: %w", err)
	}
	if _, err := io.Copy(formComposeFile, composeFile); err != nil {
		return nil, fmt.Errorf("Failed to copy compose file to form: %w", err)
	}
	envField, err := fw.CreateFormField("Env")
	if err != nil {
		return nil, fmt.Errorf("Failed to create env field in form: %w", err)
	}
	envData, err := json.Marshal(environment)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal environment variable data")
	}
	if _, err := envField.Write(envData); err != nil {
		return nil, fmt.Errorf("Failed to write env field to form: %w", err)
	}
	fw.Close()

	createUrl := cli.newUrl("/api/stacks/create/standalone/file",
		"endpointId", cli.EndpointId,
		"name", name,
	)

	req, err := http.NewRequest(http.MethodPost, createUrl.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Add("X-Api-Key", cli.ApiKey)
	req.Header.Add("Content-Type", fw.FormDataContentType())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Request error: %w", err)
	}
	defer res.Body.Close()
	resValues := new(StackCreateResponse)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body: %w", err)
	}
	if err := json.Unmarshal(body, resValues); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal response: %w", err)
	}
	if resValues.Id == 0 {
		return nil, fmt.Errorf("Failed with message:\n%s", resValues.Message)
	}
	return resValues, nil
}
