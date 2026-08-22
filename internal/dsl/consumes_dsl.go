package dsl

type Consumes struct {
	Rest    Rest              `json:"rest,omitzero"`
	Message map[string]string `json:"message,omitzero"`
}

type ConsumesServicesMap map[string]Consumes
