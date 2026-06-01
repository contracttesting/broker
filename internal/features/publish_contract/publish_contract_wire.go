package publish_contract

import (
	"encoding/json"

	"github.com/contracttesting/broker/internal/shared"
)

const (
	ContractPublishSuccessful   string = "contract publish successful"
	ContractInvalidInput        string = "contract invalid input"
	ContractVersionConflict     string = "contract version already exists with different content"
	ContractParticipantNotFound string = "contract participant not found"
)

type PublishContractRequestBody struct {
	Participant string          `json:"participant"`
	Version     string          `json:"version"`
	Contract    json.RawMessage `json:"contract"`
}

type PublishContractResponseBody struct {
	shared.BrokerResponseBody
}
