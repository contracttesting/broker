package dsl

type Consumes struct {
	Rest Rest `json:"rest,omitzero"`
	// Message is a placeholder for messaging contracts: nothing downstream reads it.
	Message map[string]string `json:"message,omitzero"`
}

type ConsumesServicesMap map[string]Consumes
