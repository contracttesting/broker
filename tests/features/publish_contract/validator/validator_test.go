package validator_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const petsProviderJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "200": "Pet" } }
      }
    }
  }
}`

const petSchemaJSON = `{
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    }
  }
}`

// everyRuleJSONA and everyRuleJSONB together violate all eleven catalog rules: a wrong
// registration segment would silence its rule's message, so this list is the catalog.
const everyRuleJSONA = `{
  "provides": {
    "rest": {
      "/dup": { "get": { "responses": { "200": "Pet" } } },
      "/dup/": { "get": { "responses": { "200": "Pet" } } },
      "/bad//x": { "get": { "responses": { "200": "Pet" } } },
      "/pets": { "get": { "responses": { "200": "Missing", "600": "Pet" } } }
    }
  },
  "consumes": {
    "Bad;Svc": { "rest": { "/c": { "get": { "responses": { "200": "Pet" } } } } },
    "payments": { "rest": { "/invoices": { "get": { "responses": { "200": "Invoice" } } } } }
  },
  "schemas": {
    "Invoice": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    },
    "Loop": { "ref": "Loop" },
    "Pet": {
      "type": "object",
      "properties": {
        "kind": { "type": "auid" },
        "owner": { "ref": "Ghost" },
        "tags": { "type": "array" }
      }
    }
  }
}`

const everyRuleJSONB = `{
  "provides": {
    "rest": {
      "/pets/": { "get": { "responses": { "200": "Pet" } } }
    }
  },
  "consumes": {
    "payments": { "rest": { "/invoices": { "get": { "responses": { "200": "Charge" } } } } }
  },
  "schemas": {
    "Charge": { "type": "object", "properties": { "id": { "type": "integer" } } },
    "Pet": { "type": "object", "properties": { "id": { "type": "string" } } }
  }
}`

func validatorFragment(t *testing.T, source, raw string) dsl.Fragment {
	t.Helper()

	contract := &dsl.Contract{}
	require.NoError(t, json.Unmarshal([]byte(raw), contract))

	return dsl.Fragment{Source: source, Contract: contract}
}

func TestValidator_EveryCatalogRule_FiresAtItsRegisteredSegment(t *testing.T) {
	violations := validator.NewContextualValidator().Validate([]dsl.Fragment{
		validatorFragment(t, "b.json", everyRuleJSONB),
		validatorFragment(t, "a.json", everyRuleJSONA),
	})

	assert.Equal(t, []string{
		`invalid endpoint "/bad//x": malformed path (a.json)`,
		"duplicate resource: provides GET /dup 200 declared twice in a.json",
		"unresolved schema name: Missing referenced at provides GET /pets 200 (a.json)",
		"invalid status code 600 at provides GET /pets (a.json)",
		`invalid service name "Bad;Svc": must be snake_case (a.json)`,
		"schema Loop is too deep with more than 10 levels (a.json)",
		`invalid schema type "auid" at Pet.kind (a.json)`,
		"unresolved schema name: Ghost referenced at Pet.owner (a.json)",
		"array schema without items at Pet.tags (a.json)",
		"duplicate resource: provides GET /pets 200 declared in a.json and b.json",
		"conflicting property type for $.id at consumes payments GET /invoices 200: string (a.json) and integer (b.json)",
		"duplicate schema: Pet declared in a.json and b.json",
	}, violations)
}

const stringIDConsumerJSON = `{
  "consumes": {
    "payments": { "rest": { "/invoices": { "get": { "responses": { "200": "InvoiceString" } } } } }
  },
  "schemas": {
    "InvoiceString": { "type": "object", "properties": { "id": { "type": "string" } } }
  }
}`

const integerIDConsumerJSON = `{
  "consumes": {
    "payments": { "rest": { "/invoices": { "get": { "responses": { "200": "InvoiceInteger" } } } } }
  },
  "schemas": {
    "InvoiceInteger": { "type": "object", "properties": { "id": { "type": "integer" } } }
  }
}`

// a ContextualValidator is one run: consecutive publishes each build their own, so
// what one run remembered must never surface in the next
func TestValidator_EachRunStartsWithFreshDuplicateTracking(t *testing.T) {
	fragments := []dsl.Fragment{
		validatorFragment(t, "a.json", petsProviderJSON),
		validatorFragment(t, "b.json", petsProviderJSON),
		validatorFragment(t, "c.json", petSchemaJSON),
		validatorFragment(t, "d.json", petSchemaJSON),
		validatorFragment(t, "e.json", stringIDConsumerJSON),
		validatorFragment(t, "f.json", integerIDConsumerJSON),
	}

	expected := []string{
		"duplicate resource: provides GET /pets 200 declared in a.json and b.json",
		"duplicate schema: Pet declared in c.json and d.json",
		"conflicting property type for $.id at consumes payments GET /invoices 200: string (e.json) and integer (f.json)",
	}

	assert.Equal(t, expected, validator.NewContextualValidator().Validate(fragments))
	assert.Equal(t, expected, validator.NewContextualValidator().Validate(fragments))
}
