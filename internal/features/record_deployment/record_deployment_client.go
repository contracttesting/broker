package record_deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/contracttesting/cli/internal/components"
)

type RecordDeploymentClient struct {
	httpClient *components.HTTPClient
}

func NewRecordDeploymentClient(httpClient *components.HTTPClient) *RecordDeploymentClient {
	return &RecordDeploymentClient{httpClient: httpClient}
}

func (c *RecordDeploymentClient) Record(ctx context.Context, input *RecordDeploymentInput) (string, error) {
	body, err := json.Marshal(recordDeploymentBody{
		Name:        input.Participant,
		Version:     input.Version,
		Environment: input.Environment,
	})
	if err != nil {
		return "", fmt.Errorf("cannot serialize deployment to JSON: %w", err)
	}

	resp, err := c.httpClient.Post(ctx, "/api/deployments", body)
	if err != nil {
		return "", fmt.Errorf("cannot post deployment to broker: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("cannot post deployment to broker: %s", components.BrokerMessage(resp.Bytes()))
	}

	var result RecordDeploymentResponse
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return "", fmt.Errorf("cannot parse deployment response: %w", err)
	}

	return result.Message, nil
}
