package components

import "encoding/json"

// BrokerMessage returns the human-readable message from the broker's standard
// {"success":false,"message":...} error envelope, falling back to the raw body
// when the response is not that envelope (e.g. a proxy error page).
func BrokerMessage(body []byte) string {
	var envelope struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Message == "" {
		return string(body)
	}
	return envelope.Message
}
