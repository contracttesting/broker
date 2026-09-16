package descriptor

import (
	"maps"
	"slices"

	"github.com/contracttesting/broker/internal/features/publish_contract/violation"
)

type Key struct {
	Code  string
	Check func(key string) error
}

func (k Key) reject(key string, path string) *violation.Violation {
	if k.Check == nil {
		return nil
	}

	err := k.Check(key)
	if err == nil {
		return nil
	}

	return &violation.Violation{
		Code: k.Code,
		Path: path,
		Details: map[string]string{
			"key":   key,
			"error": err.Error(),
		},
	}
}

func Map(key Key, value Node) Node {
	return mapNode{key: key, value: value}
}

type mapNode struct {
	key   Key
	value Node
}

func (m mapNode) validate(value any, path string) []violation.Violation {
	if value == nil {
		return nil
	}

	mapping, ok := value.(map[string]any)
	if !ok {
		return []violation.Violation{invalidKind(value, path, "mapping")}
	}

	var violations []violation.Violation

	for _, key := range slices.Sorted(maps.Keys(mapping)) {
		if rejected := m.key.reject(key, joinPath(path, key)); rejected != nil {
			violations = append(violations, *rejected)

			continue
		}

		violations = append(violations, m.value.validate(mapping[key], joinPath(path, key))...)
	}

	return violations
}
