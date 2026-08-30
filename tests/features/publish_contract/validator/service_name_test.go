package validator_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serviceNameConsumesJSON(service string) string {
	encoded, _ := json.Marshal(service)

	return fmt.Sprintf(`{
  "consumes": {
    %s: {
      "rest": {
        "/things": {
          "get": { "responses": { "200": "Thing" } }
        }
      }
    }
  },
  "schemas": {
    "Thing": { "type": "string" }
  }
}`, encoded)
}

func serviceNameViolations(t *testing.T, raw string) []string {
	t.Helper()

	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(raw), &dslContract))

	fragments := []dsl.Fragment{{Source: "things.yaml", Contract: &dslContract}}

	return validator.NewContextualValidator().Validate(fragments)
}

func TestServiceName_Accepts(t *testing.T) {
	for _, service := range []string{
		"payments",
		"payment_service",
		"service_v2",
		"v2",
		"a",
	} {
		t.Run(service, func(t *testing.T) {
			assert.Empty(t, serviceNameViolations(t, serviceNameConsumesJSON(service)))
		})
	}
}

func TestServiceName_Rejects(t *testing.T) {
	for _, service := range []string{
		"",
		"Payments",
		"payment-service",
		"payment service",
		"_payments",
		"payments_",
		"pay__ments",
		"bil;ling",
		"payments/v2",
	} {
		t.Run(service, func(t *testing.T) {
			violations := serviceNameViolations(t, serviceNameConsumesJSON(service))

			require.Len(t, violations, 1)
			assert.Equal(t, fmt.Sprintf("invalid service name %q: must be snake_case (things.yaml)", service), violations[0])
		})
	}
}

// An invalid service name gates the descent: what it declares is unreachable, so no
// resource or schema-name noise piles on top of the one spelling problem.
func TestServiceName_InvalidName_SuppressesDescent(t *testing.T) {
	raw := `{
  "consumes": {
    "Bad;Svc": {
      "rest": {
        "/things": {
          "get": { "responses": { "200": "Missing" } }
        }
      }
    }
  }
}`

	violations := serviceNameViolations(t, raw)

	assert.Equal(t, []string{
		`invalid service name "Bad;Svc": must be snake_case (things.yaml)`,
	}, violations)
}
