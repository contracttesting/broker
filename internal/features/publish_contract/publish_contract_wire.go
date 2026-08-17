package publish_contract

const (
	ContractPublishSuccessful   string = "contract publish successful"
	ContractInvalidInput        string = "contract invalid input"
	ContractValidationFailed    string = "contract validation failed"
	ContractVersionConflict     string = "contract version already exists with different content"
	ContractParticipantNotFound string = "contract participant not found"
	ContractPublishFailed       string = "contract publish failed"
)

// ContractFragment is one uploaded file: its path as the publisher spelled it, and its
// original text, byte for byte.
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

// PublishContractValidationResponseBody answers a contract the validation rejected: one
// message for the rejection, one line per violation.
type PublishContractValidationResponseBody struct {
	Message string   `json:"message"`
	Errors  []string `json:"errors"`
}
