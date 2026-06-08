package publish_contract

import "encoding/json"

type PublishContractRequestBody struct {
	Participant string          `json:"participant"`
	Version     string          `json:"version"`
	Contract    json.RawMessage `json:"contract"`
}

type PublishContractResponseBody struct {
	Message string `json:"message"`
}
