package dsl

import (
	"slices"
	"strings"
)

type Fragment struct {
	Source   string
	Document any
}

func (f Fragment) Root() Document {
	switch root := f.Document.(type) {
	case map[string]any:
		return root
	case Document:
		return root
	default:
		return nil
	}
}

func SortedBySource(fragments []Fragment) []Fragment {
	sorted := slices.Clone(fragments)

	slices.SortStableFunc(sorted, func(a, b Fragment) int {
		return strings.Compare(a.Source, b.Source)
	})

	return sorted
}
