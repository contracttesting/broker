package create_environment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/contracttesting/cli/internal/components"
)

type CreateEnvironmentClient struct {
	httpClient *components.HTTPClient
}

func NewCreateEnvironmentClient(httpClient *components.HTTPClient) *CreateEnvironmentClient {
	return &CreateEnvironmentClient{httpClient: httpClient}
}

func (c *CreateEnvironmentClient) Create(ctx context.Context, input *CreateEnvironmentInput) (string, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("cannot serialize environment to JSON: %w", err)
	}

	resp, err := c.httpClient.Post(ctx, "/api/environments", body)
	if err != nil {
		return "", fmt.Errorf("cannot post environment to broker: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("cannot post environment to broker: %s", resp.String())
	}

	var result CreateEnvironmentResponse
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return "", fmt.Errorf("cannot parse environment response: %w", err)
	}

	return result.Message, nil
}
