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

func (c *PublishContractClient) PublishContract(ctx context.Context, requestBody *PublishContractRequestBody) (string, error) {
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("cannot serialize contract to JSON: %w", err)
	}

	response, err := c.httpClient.Post(ctx, "/api/contracts", requestBodyJSON)
	if err != nil {
		return "", fmt.Errorf("cannot post contract to broker: %w", err)
	}

	var responseBody PublishContractResponseBody
	if err := json.Unmarshal(response.Bytes(), &responseBody); err != nil {
		return "", fmt.Errorf("cannot parse contract response: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("cannot post contract to broker: %s", responseBody.Message)
	}

	return responseBody.Message, nil
}
