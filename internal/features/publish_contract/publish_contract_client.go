package publish_contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/contracttesting/cli/internal/components"
)

type PublishContractClient struct {
	httpClient *components.HTTPClient
}

func NewPublishContractClient(httpClient *components.HTTPClient) *PublishContractClient {
	return &PublishContractClient{
		httpClient: httpClient,
	}
}

func (c *PublishContractClient) PublishContract(ctx context.Context, input *PublishContractInput) (string, error) {
	body, err := json.Marshal(publishContractBody{
		Name:     input.Participant,
		Version:  input.Version,
		Contract: input.ContractJSON,
	})
	if err != nil {
		return "", fmt.Errorf("cannot serialize contract to JSON: %w", err)
	}

	resp, err := c.httpClient.Post(ctx, "/api/contracts", body)
	if err != nil {
		return "", fmt.Errorf("cannot post contract to broker: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("cannot post contract to broker: %s", resp.String())
	}

	var result PublishContractResponse
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return "", fmt.Errorf("cannot parse contract response: %w", err)
	}

	return result.Message, nil
}
