package upload_contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"resty.dev/v3"

	"github.com/contracttesting/cli/internal/features/upload_contract/wireout"
)

type Client struct {
	rc        *resty.Client
	brokerURL string
}

func NewClient(brokerURL string) *Client {
	return &Client{rc: resty.New(), brokerURL: brokerURL}
}

func (c *Client) HTTPClient() *http.Client {
	return c.rc.Client()
}

type Result struct {
	Success         *wireout.UploadResponse
	BreakingChanges *wireout.BreakingChangesResponse
}

func (c *Client) Upload(ctx context.Context, contractJSON []byte) (*Result, error) {
	resp, err := c.rc.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetBody(contractJSON).
		Post(c.brokerURL + "/contracts")
	if err != nil {
		return nil, fmt.Errorf("post contracts: %w", err)
	}

	body := resp.Bytes()

	switch resp.StatusCode() {
	case http.StatusOK:
		var ok wireout.UploadResponse
		if err := json.Unmarshal(body, &ok); err != nil {
			return nil, fmt.Errorf("decode 200 response: %w", err)
		}
		return &Result{Success: &ok}, nil

	case http.StatusUnprocessableEntity:
		var breaks wireout.BreakingChangesResponse
		if err := json.Unmarshal(body, &breaks); err != nil {
			return nil, fmt.Errorf("decode 422 response: %w", err)
		}
		return &Result{BreakingChanges: &breaks}, nil

	case http.StatusBadRequest:
		var bad wireout.UploadResponse
		_ = json.Unmarshal(body, &bad)
		return nil, fmt.Errorf("broker rejected contract: %s", bad.Message)

	default:
		return nil, fmt.Errorf("broker returned unexpected status %d: %s", resp.StatusCode(), string(body))
	}
}
