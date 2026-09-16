package descriptor

import (
	"maps"
	"slices"

	"github.com/contracttesting/broker/internal/features/publish_contract/violation"
)

type Fields map[string]Node

type Written map[string]any

func (w Written) HasAny(names ...string) bool {
	for _, name := range names {
		if w[name] != nil {
			return true
		}
	}

	return false
}

// A rule fills Code and Details; the walker fills Path on its own copy.
type Rule func(written Written) *violation.Violation

type ObjectNode struct {
	fields Fields
	rules  []Rule
}

func Object(fields Fields) ObjectNode {
	return ObjectNode{fields: fields}
}

func (o ObjectNode) Rules(rules ...Rule) ObjectNode {
	o.rules = rules

	return o
}

func (o ObjectNode) validate(value any, path string) []violation.Violation {
	mapping, ok := value.(map[string]any)
	if !ok && value != nil {
		return []violation.Violation{invalidKind(value, path, "mapping")}
	}

	violations := o.validateFields(mapping, path)

	return append(violations, o.applyRules(writtenFields(mapping), path)...)
}

func (o ObjectNode) validateFields(mapping map[string]any, path string) []violation.Violation {
	var violations []violation.Violation

	for _, key := range slices.Sorted(maps.Keys(mapping)) {
		field, known := o.fields[key]
		if !known {
			violations = append(violations, violation.Violation{
				Code:    "key.unknown",
				Path:    joinPath(path, key),
				Details: map[string]string{"key": key},
			})

			continue
		}

		violations = append(violations, field.validate(mapping[key], joinPath(path, key))...)
	}

	return violations
}

func (o ObjectNode) applyRules(written Written, path string) []violation.Violation {
	var violations []violation.Violation

	for _, rule := range o.rules {
		broken := rule(written)
		if broken == nil {
			continue
		}

		found := *broken
		found.Path = path
		violations = append(violations, found)
	}

	return violations
}

func writtenFields(mapping map[string]any) Written {
	written := Written{}

	for key, value := range mapping {
		if value != nil {
			written[key] = value
		}
	}

	return written
}
