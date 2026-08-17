package dsl

import (
	"maps"
	"slices"
)

type Consumes struct {
	Rest    Rest              `json:"rest,omitzero"`
	Message map[string]string `json:"message,omitzero"`
}

// Message is a placeholder for messaging contracts: nothing downstream reads it, so the
// walk stops at Rest.
func (c Consumes) Validate(vctx ValidationContext) {
	c.Rest.Validate(vctx)
}

type ConsumesServicesMap map[string]Consumes

func (c ConsumesServicesMap) Validate(vctx ValidationContext) {
	for _, service := range slices.Sorted(maps.Keys(c)) {
		c[service].Validate(vctx.At("consumes").At(service).atResource("consumes", service))
	}
}
