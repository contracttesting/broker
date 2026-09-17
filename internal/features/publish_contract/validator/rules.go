package validator

import (
	"github.com/contracttesting/broker/internal/features/publish_contract/mapper/fragmentmapper"
	"github.com/contracttesting/broker/internal/features/publish_contract/violation"
)

type rule func(declarations fragmentmapper.Declarations) []violation.Violation

var rules = []rule{
	duplicateResources,
	conflictingPropertyTypes,
	duplicateSchemas,
	unresolvedSchemaNames,
	unresolvedSchemaRefs,
	schemasTooDeep,
}

func schemaPath(name string) string {
	return "schemas;" + name
}
