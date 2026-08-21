package validator_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
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
    "Bad;Svc": { "rest": { "/c": { "get": { "responses": { "200": "Pet" } } } } }
  },
  "schemas": {
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
  "schemas": {
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
	violations := validator.NewDslValidator().Validate([]dsl.Fragment{
		validatorFragment(t, "b.json", everyRuleJSONB),
		validatorFragment(t, "a.json", everyRuleJSONA),
	})

	assert.Equal(t, []string{
		"duplicate endpoint: /dup declared twice in a.json",
		`invalid endpoint "/bad//x": malformed path (a.json)`,
		"invalid status code 600 at provides GET /pets (a.json)",
		"unresolved schema name: Missing referenced at provides GET /pets 200 (a.json)",
		`invalid service name "Bad;Svc": must be snake_case (a.json)`,
		"schema Loop is too deep with more than 10 levels (a.json)",
		`invalid schema type "auid" at Pet.kind (a.json)`,
		"unresolved schema name: Ghost referenced at Pet.owner (a.json)",
		"array schema without items at Pet.tags (a.json)",
		"duplicate resource: provides GET /pets 200 declared in a.json and b.json",
		"duplicate schema: Pet declared in a.json and b.json",
	}, violations)
}

func TestValidator_ConsecutiveValidates_DontLeakStatefulState(t *testing.T) {
	fragments := []dsl.Fragment{
		validatorFragment(t, "a.json", petsProviderJSON),
		validatorFragment(t, "b.json", petsProviderJSON),
		validatorFragment(t, "c.json", petSchemaJSON),
		validatorFragment(t, "d.json", petSchemaJSON),
	}

	shared := validator.NewDslValidator()

	expected := []string{
		"duplicate resource: provides GET /pets 200 declared in a.json and b.json",
		"duplicate schema: Pet declared in c.json and d.json",
	}

	assert.Equal(t, expected, shared.Validate(fragments))
	assert.Equal(t, expected, shared.Validate(fragments))
}
