package wireout

type ContractMessage string

const (
	ContractUploadSuccessful ContractMessage = "contract upload successful"
	ContractInvalidInput     ContractMessage = "contract invalid input"
	ContractIncompatible     ContractMessage = "contract incompatible with stored counterparts"
)

type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type BreakingChangesResponse struct {
	Success         bool                 `json:"success"`
	Message         string               `json:"message"`
	BreakingChanges []BreakingChangeItem `json:"breakingChanges"`
}

type BreakingChangeItem struct {
	ContractName  string         `json:"contractName"`
	ContractOwner string         `json:"contractOwner"`
	Resource      BrokenResource `json:"resource"`
	Property      string         `json:"property,omitempty"`
	Reason        string         `json:"reason"`
	ExpectedType  string         `json:"expectedType,omitempty"`
	ActualType    string         `json:"actualType,omitempty"`
}

type BrokenResource struct {
	Direction  string `json:"direction"`
	Kind       string `json:"kind"`
	Provider   string `json:"provider"`
	Endpoint   string `json:"endpoint"`
	Method     string `json:"method"`
	StatusCode string `json:"statusCode"`
}
