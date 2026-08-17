package validator

import (
	"fmt"

	"github.com/contracttesting/broker/internal/dsl"
)

// Validator is the composition the publish runs: the default catalog hardcoded today,
// the point where per-company rules will be appended tomorrow. It is immutable: Append
// and Without return a new Validator and never mutate the receiver.
type Validator struct {
	entries []catalogEntry
}

// catalogEntry pairs a rule with its stratum: invariant rules keep the DSL buildable
// and are locked; policy rules are opinions a composition may remove.
type catalogEntry struct {
	rule      Rule
	invariant bool
}

func New() Validator {
	return Validator{entries: []catalogEntry{
		{rule: endpointSyntaxRule{}, invariant: true},
		{rule: endpointDuplicateRule{}, invariant: true},
		{rule: schemaDuplicateRule{}, invariant: true},
		{rule: resourceDuplicateRule{}, invariant: true},
		{rule: schemaUnresolvedNameRule{}, invariant: true},
		{rule: schemaUnresolvedRefRule{}, invariant: true},
		{rule: schemaTooDeepRule{}, invariant: true},
		{rule: schemaArrayWithoutItemsRule{}, invariant: true},
		{rule: schemaInvalidTypeRule{}, invariant: true},
		{rule: statusCodeRangeRule{}, invariant: false},
	}}
}

func (v Validator) Append(rule Rule) Validator {
	entries := make([]catalogEntry, 0, len(v.entries)+1)
	entries = append(entries, v.entries...)
	entries = append(entries, catalogEntry{rule: rule})

	return Validator{entries: entries}
}

func (v Validator) Without(code string) (Validator, error) {
	for i, entry := range v.entries {
		if entry.rule.Code() != code {
			continue
		}

		if entry.invariant {
			return Validator{}, fmt.Errorf("cannot remove rule %s: invariant rules are locked", code)
		}

		entries := make([]catalogEntry, 0, len(v.entries)-1)
		entries = append(entries, v.entries[:i]...)
		entries = append(entries, v.entries[i+1:]...)

		return Validator{entries: entries}, nil
	}

	return Validator{}, fmt.Errorf("cannot remove rule %s: not in the catalog", code)
}

// Validate walks every fragment of the raw DSL and reports everything wrong with the
// contract at once: a publish is either rejected with the full list or handed to
// dsl.Normalize and the build as valid DSL.
func (v Validator) Validate(fragments []dsl.Fragment) []error {
	rules := make([]Rule, 0, len(v.entries))
	for _, entry := range v.entries {
		rule := entry.rule
		if stateful, ok := rule.(StatefulRule); ok {
			rule = stateful.Fresh()
		}

		rules = append(rules, rule)
	}

	errs := &ErrorList{}
	index := buildIndex(fragments)

	for _, fragment := range sortedBySource(fragments) {
		walkContract(*fragment.Contract, ValidationContext{
			Index: index,
			Errs:  errs,
			Pos:   Position{Source: fragment.Source},
			rules: rules,
		})
	}

	return errs.All()
}
