package validator_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/validator"
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

const invalidSchemaTypesJSON = `{
  "schemas": {
    "Owner": { "type": "objct" },
    "Pet": {
      "type": "object",
      "properties": {
        "id": { "type": "strng" },
        "tags": { "type": "array", "items": {} }
      }
    }
  }
}`

const invalidStatusCodesJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "99": "Pet", "200": "Pet", "600": "Pet" } }
      }
    }
  },
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "post": { "responses": { "-1": "Invoice" } }
        }
      }
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

	return validator.NewContextualValidator().Validate(fragments)
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

func TestValidate_InvalidSchemaTypes_Rejected(t *testing.T) {
	messages := validateFiles(t, validatedFile{"schemas.json", invalidSchemaTypesJSON})

	assert.Equal(t, []string{
		`invalid schema type "objct" at Owner (schemas.json)`,
		`invalid schema type "strng" at Pet.id (schemas.json)`,
		`invalid schema type "" at Pet.tags[] (schemas.json)`,
	}, messages)
}

func TestValidate_StatusCodesOutOfRange_Rejected(t *testing.T) {
	messages := validateFiles(t,
		validatedFile{"api.json", invalidStatusCodesJSON},
		validatedFile{"schemas.json", validSchemasJSON},
	)

	assert.Equal(t, []string{
		"invalid status code 99 at provides GET /pets (api.json)",
		"invalid status code 600 at provides GET /pets (api.json)",
		"invalid status code -1 at consumes payments POST /invoices (api.json)",
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

func TestValidate_DuplicateProvidedResources_NameTheResourceAndBothSources(t *testing.T) {
	messages := validateFiles(t,
		validatedFile{"a.json", petsResponseFragment},
		validatedFile{"b.json", petsResponseFragment},
		validatedFile{"c.json", petsRequestFragment},
		validatedFile{"d.json", petsRequestFragment},
		validatedFile{"schemas.json", validSchemasJSON},
	)

	assert.Equal(t, []string{
		"duplicate resource: provides GET /pets 200 declared in a.json and b.json",
		"duplicate resource: provides POST /pets request declared in c.json and d.json",
		"duplicate resource: provides POST /pets 201 declared in c.json and d.json",
	}, messages)
}

// a consumed resource declared twice is merged by union at build time, so validation
// has nothing to say about it
func TestValidate_DuplicateConsumedResource_ReportsNothing(t *testing.T) {
	messages := validateFiles(t,
		validatedFile{"e.json", invoicesConsumerFragment},
		validatedFile{"f.json", invoicesConsumerFragment},
		validatedFile{"schemas.json", validSchemasJSON},
	)

	assert.Empty(t, messages)
}

func TestValidate_BothProvidedSpellingsInOneFile_ReportsDuplicateResource(t *testing.T) {
	bothSpellings := `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "200": "Pet" } }
      },
      "/pets/": {
        "get": { "responses": { "200": "Pet" } }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    }
  }
}`

	messages := validateFiles(t, validatedFile{"pets.json", bothSpellings})

	assert.Equal(t, []string{
		"duplicate resource: provides GET /pets 200 declared twice in pets.json",
	}, messages)
}

func TestValidate_BothConsumedSpellingsInOneFile_ReportsNothing(t *testing.T) {
	bothSpellings := `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "Invoice" } }
        },
        "/invoices/": {
          "get": { "responses": { "200": "Invoice" } }
        }
      }
    }
  },
  "schemas": {
    "Invoice": {
      "type": "object",
      "properties": { "total": { "type": "integer" } }
    }
  }
}`

	messages := validateFiles(t, validatedFile{"invoices.json", bothSpellings})

	assert.Empty(t, messages)
}

func TestValidate_ConsumedResourceWithConflictingTypes_NamesBothDeclarations(t *testing.T) {
	stringID := `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "InvoiceString" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceString": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    }
  }
}`

	integerID := `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "InvoiceInteger" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceInteger": {
      "type": "object",
      "properties": { "id": { "type": "integer" } }
    }
  }
}`

	messages := validateFiles(t,
		validatedFile{"b.json", integerID},
		validatedFile{"a.json", stringID},
	)

	assert.Equal(t, []string{
		"conflicting property type for $.id at consumes payments GET /invoices 200: string (a.json) and integer (b.json)",
	}, messages)
}

func TestValidate_ConsumedResourceWithConflictingTypesInOneFile_NamesTheFileTwice(t *testing.T) {
	bothSpellings := `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "InvoiceString" } }
        },
        "/invoices/": {
          "get": { "responses": { "200": "InvoiceInteger" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceString": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    },
    "InvoiceInteger": {
      "type": "object",
      "properties": { "id": { "type": "integer" } }
    }
  }
}`

	messages := validateFiles(t, validatedFile{"invoices.json", bothSpellings})

	assert.Equal(t, []string{
		"conflicting property type for $.id at consumes payments GET /invoices 200: string (invoices.json) and integer (invoices.json)",
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
