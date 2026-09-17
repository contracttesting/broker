package violation

import (
	"cmp"
	"slices"
	"strings"
)

// The broker never renders a sentence out of a violation — the CLI owns the text.
type Violation struct {
	Code    string            `json:"code"`
	Path    string            `json:"path"`
	Source  string            `json:"source"`
	Details map[string]string `json:"details"`
}

func SortedByLocation(violations []Violation) []Violation {
	sorted := slices.Clone(violations)

	slices.SortStableFunc(sorted, func(a, b Violation) int {
		return cmp.Or(strings.Compare(a.Source, b.Source), strings.Compare(a.Path, b.Path))
	})

	return sorted
}
