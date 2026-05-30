package publish_contract

import "encoding/json"

type PublishContractInput struct {
	Participant  string
	Version      string
	ContractJSON []byte
}

type publishContractBody struct {
	Name     string          `json:"name"`
	Version  string          `json:"version"`
	Contract json.RawMessage `json:"contract"`
}

type PublishContractResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
