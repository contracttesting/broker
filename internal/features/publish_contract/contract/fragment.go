package contract

import (
	"slices"
	"strings"
)

type Fragment struct {
	Source   string
	Document any
}

func (f Fragment) Root() Document {
	root, _ := f.Document.(map[string]any)

	return root
}

func SortedBySource(fragments []Fragment) []Fragment {
	sorted := slices.Clone(fragments)

	slices.SortStableFunc(sorted, func(a, b Fragment) int {
		return strings.Compare(a.Source, b.Source)
	})

	return sorted
}
