package dsl

type Provides struct {
	Rest Rest `json:"rest,omitzero"`
	// Message is a placeholder for messaging contracts: nothing downstream reads it.
	Message map[string]string `json:"message,omitzero"`
}
