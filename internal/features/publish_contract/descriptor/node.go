package descriptor

import (
	"github.com/contracttesting/broker/internal/features/publish_contract/violation"
)

type Node interface {
	validate(value any, path string) []violation.Violation
}

func Validate(grammar Node, document any, source string) []violation.Violation {
	violations := grammar.validate(document, "")

	for index := range violations {
		violations[index].Source = source
	}

	return violations
}

// `;` because a segment carries `.` and `/` (endpoints), so it is the only separator a consumer can split on.
func joinPath(path string, key string) string {
	if path == "" {
		return key
	}

	return path + ";" + key
}

func kindOf(value any) string {
	switch value.(type) {
	case map[string]any:
		return "mapping"
	case []any:
		return "sequence"
	case string:
		return "string"
	case bool:
		return "boolean"
	case int, int64, uint64:
		return "integer"
	case float64:
		return "float"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func invalidKind(value any, path string, expected string) violation.Violation {
	return violation.Violation{
		Code: "value.invalid_kind",
		Path: path,
		Details: map[string]string{
			"expected": expected,
			"got":      kindOf(value),
		},
	}
}
