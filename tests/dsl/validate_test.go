package dsl_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validEndpointsJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "200": "Pet" } },
        "post": { "request": "Pet", "responses": { "201": "Pet" } }
      }
    }
  },
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "Invoice" } }
        }
      }
    }
  }
}`

const validSchemasJSON = `{
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    },
    "Invoice": {
      "type": "object",
      "properties": { "total": { "type": "integer" } }
    }
  }
}`

const unresolvedNamesJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "200": "Missing" } },
        "post": { "request": "AlsoMissing" }
      }
    }
  },
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "put": { "request": "Gone", "responses": { "200": "Vanished" } }
        }
      }
    }
  }
}`

const unreachedRefJSON = `{
  "schemas": {
    "Invoice": {
      "type": "object",
      "properties": {
        "payment": { "ref": "Payment" },
        "lines": {
          "type": "array",
          "items": { "ref": "Line" }
        }
      }
    }
  }
}`

const cyclicSchemasJSON = `{
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": { "owner": { "ref": "Owner" } }
    },
    "Owner": {
      "type": "object",
      "properties": { "pet": { "ref": "Pet" } }
    }
  }
}`

const deeplyNestedJSON = `{
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": { "l1": {
        "type": "object",
        "properties": { "l2": {
          "type": "object",
          "properties": { "l3": {
            "type": "object",
            "properties": { "l4": {
              "type": "object",
              "properties": { "l5": {
                "type": "object",
                "properties": { "l6": {
                  "type": "object",
                  "properties": { "l7": {
                    "type": "object",
                    "properties": { "l8": {
                      "type": "object",
                      "properties": { "l9": {
                        "type": "object",
                        "properties": { "l10": {
                          "type": "object",
                          "properties": { "l11": { "type": "string" } }
                        } }
                      } }
                    } }
                  } }
                } }
              } }
            } }
          } }
        } }
      } }
    }
  }
}`

const invalidEndpointJSON = `{
  "provides": {
    "rest": {
      "/users/{userId}": {
        "get": { "responses": { "200": "Missing" } }
      }
    }
  }
}`

const arrayWithoutItemsJSON = `{
  "schemas": {
    "Pets": { "type": "array" },
    "Owner": {
      "type": "object",
      "properties": { "pets": { "type": "array" } }
    }
  }
}`

const petsResponseFragment = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "200": "Pet" } }
      }
    }
  }
}`

const petsRequestFragment = `{
  "provides": {
    "rest": {
      "/pets": {
        "post": { "request": "Pet", "responses": { "201": "Pet" } }
      }
    }
  }
}`

const invoicesConsumerFragment = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "Invoice" } }
        }
      }
    }
  }
}`

type validatedFile struct {
	source string
	raw    string
}

func validateFiles(t *testing.T, files ...validatedFile) []string {
	t.Helper()

	fragments := make([]dsl.Fragment, 0, len(files))
	for _, file := range files {
		contract := &dsl.Contract{}
		require.NoError(t, json.Unmarshal([]byte(file.raw), contract))

		fragments = append(fragments, dsl.Fragment{Source: file.source, Contract: contract})
	}

	violations := dsl.Normalize(fragments)
	violations = append(violations, dsl.Validate(fragments)...)

	messages := make([]string, 0, len(violations))
	for _, violation := range violations {
		messages = append(messages, violation.Error())
	}

	return messages
}

func TestValidate_ValidContractAcrossFragments_ReportsNothing(t *testing.T) {
	messages := validateFiles(t,
		validatedFile{"endpoints.json", validEndpointsJSON},
		validatedFile{"schemas.json", validSchemasJSON},
	)

	assert.Empty(t, messages)
}

func TestValidate_UnresolvedSchemaNames_QuoteTheResourceTheyName(t *testing.T) {
	messages := validateFiles(t, validatedFile{"api.json", unresolvedNamesJSON})

	assert.Equal(t, []string{
		"unresolved schema name: Missing referenced at provides GET /pets 200 (api.json)",
		"unresolved schema name: AlsoMissing referenced at provides POST /pets request (api.json)",
		"unresolved schema name: Gone referenced at consumes payments PUT /invoices request (api.json)",
		"unresolved schema name: Vanished referenced at consumes payments PUT /invoices 200 (api.json)",
	}, messages)
}

func TestValidate_UnresolvedRefInNeverReferencedSchema_ReportsEachPath(t *testing.T) {
	messages := validateFiles(t, validatedFile{"billing.json", unreachedRefJSON})

	assert.Equal(t, []string{
		"unresolved schema name: Line referenced at Invoice.lines[] (billing.json)",
		"unresolved schema name: Payment referenced at Invoice.payment (billing.json)",
	}, messages)
}

func TestValidate_CyclicSchemas_ReportOneTooDeepPerRoot(t *testing.T) {
	messages := validateFiles(t, validatedFile{"schemas.json", cyclicSchemasJSON})

	assert.Equal(t, []string{
		"schema Owner is too deep with more than 10 levels (schemas.json)",
		"schema Pet is too deep with more than 10 levels (schemas.json)",
	}, messages)
}

func TestValidate_NestingPastMaxDepth_ReportsTooDeep(t *testing.T) {
	messages := validateFiles(t, validatedFile{"schemas.json", deeplyNestedJSON})

	assert.Equal(t, []string{
		"schema Pet is too deep with more than 10 levels (schemas.json)",
	}, messages)
}

func TestValidate_InvalidEndpoint_ReportedOnceAndMethodsNotVisited(t *testing.T) {
	messages := validateFiles(t, validatedFile{"api.json", invalidEndpointJSON})

	assert.Equal(t, []string{
		`invalid endpoint "/users/{userId}": dynamic path segments must use * (api.json)`,
	}, messages)
}

func TestValidate_ArrayWithoutItems_Rejected(t *testing.T) {
	messages := validateFiles(t, validatedFile{"schemas.json", arrayWithoutItemsJSON})

	assert.Equal(t, []string{
		"array schema without items at Owner.pets (schemas.json)",
		"array schema without items at Pets (schemas.json)",
	}, messages)
}

func TestValidate_DuplicateSchema_NamesBothSources(t *testing.T) {
	messages := validateFiles(t,
		validatedFile{"schemas.json", validSchemasJSON},
		validatedFile{"billing.json", validSchemasJSON},
	)

	assert.Equal(t, []string{
		"duplicate schema: Invoice declared in billing.json and schemas.json",
		"duplicate schema: Pet declared in billing.json and schemas.json",
	}, messages)
}

func TestValidate_DuplicateResources_NameTheResourceAndBothSources(t *testing.T) {
	messages := validateFiles(t,
		validatedFile{"a.json", petsResponseFragment},
		validatedFile{"b.json", petsResponseFragment},
		validatedFile{"c.json", petsRequestFragment},
		validatedFile{"d.json", petsRequestFragment},
		validatedFile{"e.json", invoicesConsumerFragment},
		validatedFile{"f.json", invoicesConsumerFragment},
		validatedFile{"schemas.json", validSchemasJSON},
	)

	assert.Equal(t, []string{
		"duplicate resource: provides GET /pets 200 declared in a.json and b.json",
		"duplicate resource: provides POST /pets request declared in c.json and d.json",
		"duplicate resource: provides POST /pets 201 declared in c.json and d.json",
		"duplicate resource: consumes payments GET /invoices 200 declared in e.json and f.json",
	}, messages)
}

func TestValidate_TrailingSlashInAnotherFile_CollidesAsDuplicateResource(t *testing.T) {
	slashed := `{
  "provides": {
    "rest": {
      "/pets/": {
        "get": { "responses": { "200": "Pet" } }
      }
    }
  }
}`

	messages := validateFiles(t,
		validatedFile{"a.json", petsResponseFragment},
		validatedFile{"b.json", slashed},
		validatedFile{"schemas.json", validSchemasJSON},
	)

	assert.Equal(t, []string{
		"duplicate resource: provides GET /pets 200 declared in a.json and b.json",
	}, messages)
}

func TestValidate_SameInputTwice_ReportsTheSameOrder(t *testing.T) {
	files := []validatedFile{
		{"api.json", unresolvedNamesJSON},
		{"billing.json", unreachedRefJSON},
		{"schemas.json", cyclicSchemasJSON},
	}

	first := validateFiles(t, files...)
	second := validateFiles(t, files...)

	require.Len(t, first, 8)
	assert.Equal(t, first, second)
}
