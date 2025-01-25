package portainer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type updateStackRequest struct {
	Env       []EnvironmentVariable `json:"env"`
	Prune     bool                  `json:"prune"`
	PullImage bool                  `json:"pullImage"`
	StackFile string                `json:"stackFileContent"`
}

type StackUpdateResponse struct {
	Id      int    `json:"Id"`
	Message string `json:"Message"`
}

func (cli *Client) UpdateStack(stackId int, composeFile io.Reader, environment []EnvironmentVariable) (*StackUpdateResponse, error) {
	b := strings.Builder{}
	if _, err := io.Copy(&b, composeFile); err != nil {
		return nil, fmt.Errorf("Failed to copy compose file: %w", err)
	}
	reqData := updateStackRequest{
		Env:       environment,
		Prune:     true,
		PullImage: false,
		StackFile: b.String(),
	}
	buf, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal request: %w", err)
	}

	u := cli.newUrl(fmt.Sprintf("/api/stacks/%d", stackId),
		"endpointId", cli.EndpointId,
	)

	req, err := http.NewRequest(http.MethodPut, u.String(), bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Add("X-Api-Key", cli.ApiKey)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error making request: %w", err)
	}
	defer res.Body.Close()
	resValues := new(StackUpdateResponse)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body: %w", err)
	}
	if err := json.Unmarshal(body, resValues); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal response: %w", err)
	}
	if resValues.Id == 0 {
		return nil, fmt.Errorf("Failed to create stack with message:\n%s", resValues.Message)
	}
	return resValues, nil
}
