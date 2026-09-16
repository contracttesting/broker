package descriptor

import (
	"strings"

	"github.com/contracttesting/broker/internal/features/publish_contract/violation"
)

func Schema() Node {
	return Object(Fields{
		"type":        SchemaType,
		"description": Text,
		"properties":  Map(PropertyName, Lazy(Schema)),
		"items":       Lazy(Schema),
		"ref":         SchemaRef,
		"optional":    Flag,
	}).Rules(schemaHasShape, arrayRequiresItems)
}

func schemaHasShape(written Written) *violation.Violation {
	if written.HasAny("type", "properties", "items", "ref") {
		return nil
	}

	return &violation.Violation{
		Code: SchemaType.Code,
		Details: map[string]string{
			"value":   "",
			"allowed": strings.Join(SchemaType.Allowed, ", "),
		},
	}
}

func arrayRequiresItems(written Written) *violation.Violation {
	if written["type"] != "array" || written["items"] != nil {
		return nil
	}

	return &violation.Violation{Code: "schema.array_without_items"}
}
