package shared

type BrokerResponseBody struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
