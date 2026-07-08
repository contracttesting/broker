package dsl

import (
	"fmt"
	"maps"

	"github.com/contracttesting/broker/internal/model"
)

func validateSchemas(schemas SchemasMap) error {
	for name, schema := range schemas {
		if err := validateSchema(name, schemas, schema); err != nil {
			return err
		}
	}

	return nil
}

func validateSchema(name string, schemas SchemasMap, schema Schema) error {
	if schema.AnyOf != nil {
		if len(schema.AnyOf) == 0 {
			return fmt.Errorf("invalid schema %q: anyOf must not be empty", name)
		}

		if schema.Type != "" || schema.Ref != "" || schema.Properties != nil || schema.Items != nil {
			return fmt.Errorf("invalid schema %q: anyOf cannot be combined with type, $ref, properties or items", name)
		}

		if len(schema.AnyOf) == 1 {
			return fmt.Errorf("invalid schema %q: anyOf with a single variant is just that schema", name)
		}

		for _, variant := range schema.AnyOf {
			if variant.AnyOf != nil {
				return fmt.Errorf("invalid schema %q: anyOf variant cannot itself be anyOf", name)
			}
		}

		if hasDuplicateVariants(name, schemas, schema.AnyOf) {
			return fmt.Errorf("invalid schema %q: anyOf variants must be structurally distinct", name)
		}
	}

	for _, property := range schema.Properties {
		if err := validateSchema(name, schemas, property); err != nil {
			return err
		}
	}

	if schema.Items != nil {
		if err := validateSchema(name, schemas, *schema.Items); err != nil {
			return err
		}
	}

	for _, variant := range schema.AnyOf {
		if err := validateSchema(name, schemas, variant); err != nil {
			return err
		}
	}

	return nil
}

// hasDuplicateVariants flattens each variant with refs inlined; two variants
// producing identical flat property maps carry no meaning as a union.
func hasDuplicateVariants(name string, schemas SchemasMap, variants []Schema) bool {
	flattened := make([]map[string]model.Property, len(variants))
	for index, variant := range variants {
		flattened[index] = buildSchema(
			NewDepthCounter(name),
			schemas,
			make(map[string]model.Property),
			NewPropertyPath("$"),
			variant,
		)
	}

	for i := range flattened {
		for j := i + 1; j < len(flattened); j++ {
			if maps.Equal(flattened[i], flattened[j]) {
				return true
			}
		}
	}

	return false
}
