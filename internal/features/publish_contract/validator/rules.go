package validator

import (
	"strings"

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

func schemaPath(name string, segments ...string) string {
	return strings.Join(append([]string{"schemas", name}, segments...), ";")
}
