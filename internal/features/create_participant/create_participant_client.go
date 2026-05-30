package create_participant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/contracttesting/cli/internal/components"
)

type CreateParticipantClient struct {
	httpClient *components.HTTPClient
}

func NewCreateParticipantClient(httpClient *components.HTTPClient) *CreateParticipantClient {
	return &CreateParticipantClient{httpClient: httpClient}
}

func (c *CreateParticipantClient) Create(ctx context.Context, input *CreateParticipantInput) (string, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("cannot serialize participant to JSON: %w", err)
	}

	resp, err := c.httpClient.Post(ctx, "/api/participants", body)
	if err != nil {
		return "", fmt.Errorf("cannot post participant to broker: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("cannot post participant to broker: %s", resp.String())
	}

	var result CreateParticipantResponse
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return "", fmt.Errorf("cannot parse participant response: %w", err)
	}

	return result.Message, nil
}
