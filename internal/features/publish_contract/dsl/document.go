package dsl

import (
	"maps"
	"slices"
)

type Document map[string]any

func (d Document) Mapping(key string) Document {
	mapping, _ := d[key].(map[string]any)

	return mapping
}

func (d Document) Text(key string) string {
	text, _ := d[key].(string)

	return text
}

func (d Document) Flag(key string) bool {
	flag, _ := d[key].(bool)

	return flag
}

func (d Document) Keys() []string {
	return slices.Sorted(maps.Keys(d))
}
