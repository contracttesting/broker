package can_i_deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/contracttesting/cli/internal/components"
)

type CanIDeployClient struct {
	httpClient *components.HTTPClient
}

func NewCanIDeployClient(httpClient *components.HTTPClient) *CanIDeployClient {
	return &CanIDeployClient{httpClient: httpClient}
}

func (c *CanIDeployClient) Check(ctx context.Context, input *CanIDeployInput) (CanIDeployResponse, error) {
	body, err := json.Marshal(canIDeployBody{
		Name:        input.Participant,
		Version:     input.Version,
		Environment: input.Environment,
	})
	if err != nil {
		return CanIDeployResponse{}, fmt.Errorf("cannot serialize can-i-deploy request to JSON: %w", err)
	}

	resp, err := c.httpClient.Post(ctx, "/api/can-i-deploy", body)
	if err != nil {
		return CanIDeployResponse{}, fmt.Errorf("cannot query can-i-deploy from broker: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return CanIDeployResponse{}, fmt.Errorf("cannot query can-i-deploy from broker: %s", resp.String())
	}

	var result CanIDeployResponse
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return CanIDeployResponse{}, fmt.Errorf("cannot parse can-i-deploy response: %w", err)
	}

	return result, nil
}
