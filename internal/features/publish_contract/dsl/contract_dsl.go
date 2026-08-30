package dsl

type Contract struct {
	Provides         Provides            `json:"provides,omitzero"`
	ConsumesServices ConsumesServicesMap `json:"consumes,omitzero"`
	Schemas          SchemasMap          `json:"schemas,omitzero"`
}

// Fragment is one uploaded file: the contract parsed out of it plus the path it came
// from, which every publish error quotes.
type Fragment struct {
	Source   string
	Contract *Contract
}
