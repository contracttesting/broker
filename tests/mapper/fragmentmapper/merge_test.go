package fragmentmapper_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/mapper/fragmentmapper"
	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const listModuleJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "InvoiceList" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceList": {
      "type": "object",
      "properties": {
        "id":   { "type": "string" },
        "name": { "type": "string" }
      }
    }
  }
}`

const detailModuleJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "InvoiceDetail" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceDetail": {
      "type": "object",
      "properties": {
        "id":         { "type": "string" },
        "created_at": { "type": "string" }
      }
    }
  }
}`

const createModuleJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "post": { "request": "InvoiceCreate", "responses": { "201": "InvoiceCreate" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceCreate": {
      "type": "object",
      "properties": {
        "id":   { "type": "string" },
        "name": { "type": "string" }
      }
    }
  }
}`

const importModuleJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "post": { "request": "InvoiceImport", "responses": { "201": "InvoiceImport" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceImport": {
      "type": "object",
      "properties": {
        "id":    { "type": "string" },
        "email": { "type": "string" }
      }
    }
  }
}`

const optionalReaderJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "InvoiceOptional" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceOptional": {
      "type": "object",
      "properties": {
        "total":  { "type": "integer", "optional": true },
        "status": { "type": "string", "optional": true }
      }
    }
  }
}`

const requiredReaderJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "InvoiceRequired" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceRequired": {
      "type": "object",
      "properties": {
        "total": { "type": "integer" }
      }
    }
  }
}`

const otherOptionalReaderJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": { "responses": { "200": "InvoiceOtherOptional" } }
        }
      }
    }
  },
  "schemas": {
    "InvoiceOtherOptional": {
      "type": "object",
      "properties": {
        "total": { "type": "integer", "optional": true }
      }
    }
  }
}`

const optionalSenderJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "put": { "request": "PutOptional" }
        }
      }
    }
  },
  "schemas": {
    "PutOptional": {
      "type": "object",
      "properties": {
        "id":   { "type": "string" },
        "note": { "type": "string", "optional": true }
      }
    }
  }
}`

const requiredSenderJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "put": { "request": "PutRequired" }
        }
      }
    }
  },
  "schemas": {
    "PutRequired": {
      "type": "object",
      "properties": {
        "id":   { "type": "string" },
        "note": { "type": "string" }
      }
    }
  }
}`

const providedPetsJSON = `{
  "provides": {
    "rest": {
      "/pets": {
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

const providedPetsSchemalessJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "200": "Pet" } }
      }
    }
  }
}`

const fourModulesChecksum = "d55e2d0621975b40535271dc6fa098aa237646bb87924674e6b700a5fef23563"

type mergedFile struct {
	source string
	raw    string
}

func mergeFragments(t *testing.T, files ...mergedFile) []dsl.Fragment {
	t.Helper()

	fragments := make([]dsl.Fragment, 0, len(files))
	for _, file := range files {
		contract := &dsl.Contract{}
		require.NoError(t, json.Unmarshal([]byte(file.raw), contract))

		fragments = append(fragments, dsl.Fragment{Source: file.source, Contract: contract})
	}

	return fragments
}

func mergeResources(t *testing.T, files ...mergedFile) []model.UploadedResource {
	t.Helper()

	resources, err := fragmentmapper.ToResourceModels(mergeFragments(t, files...))
	require.NoError(t, err)

	return resources
}

func mergeContract(t *testing.T, files ...mergedFile) *model.UploadedContract {
	t.Helper()

	contract := model.NewUploadedContract(0, "front_app", "1", "")
	for _, resource := range mergeResources(t, files...) {
		require.NoError(t, contract.AddResource(&resource))
	}

	return contract
}

func mergedResource(t *testing.T, resources []model.UploadedResource, interaction model.Interaction) model.UploadedResource {
	t.Helper()

	for _, resource := range resources {
		if resource.Interaction == interaction {
			return resource
		}
	}

	t.Fatalf("no %v resource was built", interaction)

	return model.UploadedResource{}
}

func TestMerge_TwoConsumerModulesReadingOneResponse_UnionOfProperties(t *testing.T) {
	resources := mergeResources(t,
		mergedFile{"list.json", listModuleJSON},
		mergedFile{"detail.json", detailModuleJSON},
	)

	require.Len(t, resources, 1)

	resource := mergedResource(t, resources, model.RestResponse)
	assert.Equal(t, "/invoices", resource.Endpoint)
	assert.Equal(t, "200", resource.ResponseStatusCode.String)

	assert.Equal(t, map[string]model.Property{
		"$":            {Path: "$", Type: "object"},
		"$.id":         {Path: "$.id", Type: "string"},
		"$.name":       {Path: "$.name", Type: "string"},
		"$.created_at": {Path: "$.created_at", Type: "string"},
	}, resource.Properties)
}

func TestMerge_TwoConsumerModulesSendingOneRequest_FieldSentByOneIsOptional(t *testing.T) {
	resources := mergeResources(t,
		mergedFile{"create.json", createModuleJSON},
		mergedFile{"import.json", importModuleJSON},
	)

	request := mergedResource(t, resources, model.RestRequest)

	assert.Equal(t, map[string]model.Property{
		"$":       {Path: "$", Type: "object"},
		"$.id":    {Path: "$.id", Type: "string"},
		"$.name":  {Path: "$.name", Type: "string", Optional: true},
		"$.email": {Path: "$.email", Type: "string", Optional: true},
	}, request.Properties)
}

func TestMerge_ResponseRequiredByOneReader_StaysRequired(t *testing.T) {
	resources := mergeResources(t,
		mergedFile{"optional.json", optionalReaderJSON},
		mergedFile{"required.json", requiredReaderJSON},
	)

	resource := mergedResource(t, resources, model.RestResponse)

	assert.False(t, resource.Properties["$.total"].Optional)
	assert.True(t, resource.Properties["$.status"].Optional)
}

func TestMerge_ResponseOptionalForEveryReader_StaysOptional(t *testing.T) {
	resources := mergeResources(t,
		mergedFile{"optional.json", optionalReaderJSON},
		mergedFile{"other_optional.json", otherOptionalReaderJSON},
	)

	resource := mergedResource(t, resources, model.RestResponse)

	assert.True(t, resource.Properties["$.total"].Optional)
	assert.True(t, resource.Properties["$.status"].Optional)
}

func TestMerge_RequestOptionalForOneSender_IsOptionalForAll(t *testing.T) {
	resources := mergeResources(t,
		mergedFile{"optional.json", optionalSenderJSON},
		mergedFile{"required.json", requiredSenderJSON},
	)

	request := mergedResource(t, resources, model.RestRequest)

	assert.False(t, request.Properties["$.id"].Optional)
	assert.True(t, request.Properties["$.note"].Optional)
}

func TestMerge_IdenticalDeclarationInTwoFiles_BuildsOneResource(t *testing.T) {
	merged := mergeResources(t,
		mergedFile{"a.json", listModuleJSON},
		mergedFile{"b.json", listModuleJSON},
	)
	single := mergeResources(t, mergedFile{"a.json", listModuleJSON})

	require.Len(t, merged, 1)
	assert.Equal(t, single, merged)
}

func TestMerge_FragmentOrderReversed_ProducesTheSameContract(t *testing.T) {
	forward := mergeContract(t,
		mergedFile{"list.json", listModuleJSON},
		mergedFile{"detail.json", detailModuleJSON},
		mergedFile{"create.json", createModuleJSON},
		mergedFile{"import.json", importModuleJSON},
	)
	backward := mergeContract(t,
		mergedFile{"import.json", importModuleJSON},
		mergedFile{"create.json", createModuleJSON},
		mergedFile{"detail.json", detailModuleJSON},
		mergedFile{"list.json", listModuleJSON},
	)

	assert.Equal(t, forward.Resources, backward.Resources)
	assert.Equal(t, fourModulesChecksum, forward.Checksum())
	assert.Equal(t, fourModulesChecksum, backward.Checksum())
}

func TestMerge_ProvidedResourceDeclaredTwice_BreaksTheInvariant(t *testing.T) {
	_, err := fragmentmapper.ToResourceModels(mergeFragments(t,
		mergedFile{"a.json", providedPetsJSON},
		mergedFile{"b.json", providedPetsSchemalessJSON},
	))

	require.EqualError(t, err, "resource already added: provides GET /pets 200 from a.json and b.json")
}
