package dsl

type Provides struct {
	Rest    Rest              `json:"rest,omitzero"`
	Message map[string]string `json:"message,omitzero"`
}
