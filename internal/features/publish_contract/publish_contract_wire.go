package publish_contract

import (
	"encoding/json"

	"github.com/contracttesting/cli/internal/shared"
)

type PublishContractRequestBody struct {
	Participant string          `json:"participant"`
	Version     string          `json:"version"`
	Contract    json.RawMessage `json:"contract"`
}

type PublishContractResponseBody struct {
	shared.BrokerResponseBody
}
