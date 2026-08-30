package validator

import (
	"maps"
	"slices"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
)

type ContractIndex struct {
	schemas dsl.SchemasMap
}

func NewContractIndex(fragments []dsl.Fragment) *ContractIndex {
	index := &ContractIndex{schemas: make(dsl.SchemasMap)}

	for _, fragment := range sortedBySource(fragments) {
		index.indexSchemas(fragment)
	}

	return index
}

func (i *ContractIndex) Schema(name string) (dsl.Schema, bool) {
	schema, declared := i.schemas[name]

	return schema, declared
}

func (i *ContractIndex) indexSchemas(fragment dsl.Fragment) {
	for _, name := range slices.Sorted(maps.Keys(fragment.Contract.Schemas)) {
		if _, taken := i.schemas[name]; taken {
			continue
		}

		i.schemas[name] = fragment.Contract.Schemas[name]
	}
}
