package validator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/contracttesting/broker/internal/dsl"
)

type ErrorList struct {
	errors []error
}

func (e *ErrorList) Addf(format string, args ...any) {
	e.errors = append(e.errors, fmt.Errorf(format, args...))
}

func (e *ErrorList) All() []error {
	return e.errors
}

type Position struct {
	Source  string
	Segment string
	Depth   DepthCounter
}

func joinWhere(where, segment string) string {
	if where == "" || segment == "" {
		return where + segment
	}

	return where + " " + segment
}

func sortedBySource(fragments []dsl.Fragment) []dsl.Fragment {
	sorted := slices.Clone(fragments)

	slices.SortFunc(sorted, func(a, b dsl.Fragment) int {
		return strings.Compare(a.Source, b.Source)
	})

	return sorted
}
