package publish_contract

const (
	ContractPublishSuccessful   string = "contract publish successful"
	ContractInvalidInput        string = "contract invalid input"
	ContractValidationFailed    string = "contract validation failed"
	ContractVersionConflict     string = "contract version already exists with different content"
	ContractParticipantNotFound string = "contract participant not found"
	ContractPublishFailed       string = "contract publish failed"
)

type ContractFragment struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

type PublishContractRequestBody struct {
	Participant string             `json:"participant"`
	Version     string             `json:"version"`
	Contracts   []ContractFragment `json:"contracts"`
}

type PublishContractResponseBody struct {
	Message string `json:"message"`
}

type PublishContractValidationResponseBody struct {
	Message    string   `json:"message"`
	Violations []string `json:"violations"`
}
