package dsl_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const happyContractJSON = `{
  "consumes": {
    "pets-service": {
      "rest": {
        "/pets": {
          "get": {
            "responses": {
              "200": "Pet"
            }
          }
        }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": {
        "id":   { "type": "string" },
        "name": { "type": "string" }
      }
    }
  }
}`

const cyclicContractJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": {
            "200": "Pet"
          }
        }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": {
        "self": { "$ref": "Pet" }
      }
    }
  }
}`

func TestHydrateContract_Happy_MaterializesResources(t *testing.T) {
	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(happyContractJSON), &dslContract))

	contract := model.NewUploadedContract(model.NewParticipant("petstore-app"), "1", happyContractJSON)
	require.NoError(t, dslContract.HydrateContract(contract))

	require.Len(t, contract.Resources, 1)

	var resource model.UploadedResource
	for _, r := range contract.Resources {
		resource = r
	}

	assert.Equal(t, model.Consumes, resource.Direction)
	assert.Equal(t, model.RestResponse, resource.Interaction)
	assert.Equal(t, "pets-service", resource.ConsumedProvider.String)
	assert.Equal(t, "/pets", resource.Endpoint)
	assert.Equal(t, "get", resource.Method)
	assert.Equal(t, "200", resource.ResponseStatusCode.String)

	assert.Contains(t, resource.Properties, "$")
	assert.Contains(t, resource.Properties, "$.id")
	assert.Contains(t, resource.Properties, "$.name")
	assert.Equal(t, "string", resource.Properties["$.id"].Type)
}

const postWithRequestBodyJSON = `{
  "consumes": {
    "pets-service": {
      "rest": {
        "/pets": {
          "post": {
            "request": "Pet",
            "responses": { "201": "Pet" }
          }
        }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": {
        "id":   { "type": "string" },
        "name": { "type": "string" }
      }
    }
  }
}`

const provideRestResponseJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": { "200": "Pet" }
        }
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

const primitiveTopLevelJSON = `{
  "consumes": {
    "ping-service": {
      "rest": {
        "/ping": {
          "get": { "responses": { "200": "Pong" } }
        }
      }
    }
  },
  "schemas": {
    "Pong": { "type": "string" }
  }
}`

const arrayOfObjectsJSON = `{
  "consumes": {
    "pets-service": {
      "rest": {
        "/pets": {
          "get": { "responses": { "200": "PetList" } }
        }
      }
    }
  },
  "schemas": {
    "PetList": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": { "type": "string" }
        }
      }
    }
  }
}`

const refResolvesJSON = `{
  "consumes": {
    "pets-service": {
      "rest": {
        "/pets": {
          "get": { "responses": { "200": "PetRef" } }
        }
      }
    }
  },
  "schemas": {
    "PetRef": { "$ref": "Pet" },
    "Pet": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    }
  }
}`

func hydrate(t *testing.T, raw string) *model.UploadedContract {
	t.Helper()

	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(raw), &dslContract))

	contract := model.NewUploadedContract(model.NewParticipant("petstore-app"), "1", raw)
	require.NoError(t, dslContract.HydrateContract(contract))
	return contract
}

func TestHydrateContract_PostWithRequestBody_EmitsRequestAndResponses(t *testing.T) {
	contract := hydrate(t, postWithRequestBodyJSON)

	require.Len(t, contract.Resources, 2)

	var request, response model.UploadedResource
	for _, r := range contract.Resources {
		switch r.Interaction {
		case model.RestRequest:
			request = r
		case model.RestResponse:
			response = r
		}
	}

	assert.Equal(t, model.Consumes, request.Direction)
	assert.Equal(t, model.RestRequest, request.Interaction)
	assert.Equal(t, "/pets", request.Endpoint)
	assert.Equal(t, "post", request.Method)
	assert.Empty(t, request.ResponseStatusCode)
	assert.Contains(t, request.Properties, "$.id")

	assert.Equal(t, model.Consumes, response.Direction)
	assert.Equal(t, model.RestResponse, response.Interaction)
	assert.Equal(t, "201", response.ResponseStatusCode.String)
	assert.Contains(t, response.Properties, "$.name")
}

func TestHydrateContract_ProvidesSide_EmitsProvidedResource(t *testing.T) {
	contract := hydrate(t, provideRestResponseJSON)

	require.Len(t, contract.Resources, 1)

	var resource model.UploadedResource
	for _, r := range contract.Resources {
		resource = r
	}

	assert.Equal(t, model.Provides, resource.Direction)
	assert.Equal(t, model.RestResponse, resource.Interaction)
	assert.Empty(t, resource.ConsumedProvider)
	assert.Equal(t, "/pets", resource.Endpoint)
	assert.Equal(t, "get", resource.Method)
	assert.Equal(t, "200", resource.ResponseStatusCode.String)
	assert.Contains(t, resource.Properties, "$.id")
}

func TestHydrateContract_PrimitiveTopLevel_EmitsRootPrimitive(t *testing.T) {
	contract := hydrate(t, primitiveTopLevelJSON)

	require.Len(t, contract.Resources, 1)

	var resource model.UploadedResource
	for _, r := range contract.Resources {
		resource = r
	}

	require.Contains(t, resource.Properties, "$")
	assert.Equal(t, "string", resource.Properties["$"].Type)
	assert.Len(t, resource.Properties, 1)
}

func TestHydrateContract_ArrayOfObjects_WalksItemsViaSchemaPointer(t *testing.T) {
	contract := hydrate(t, arrayOfObjectsJSON)

	require.Len(t, contract.Resources, 1)

	var resource model.UploadedResource
	for _, r := range contract.Resources {
		resource = r
	}

	require.Contains(t, resource.Properties, "$")
	require.Contains(t, resource.Properties, "$[]")
	require.Contains(t, resource.Properties, "$[].id")
	assert.Equal(t, "array", resource.Properties["$"].Type)
	assert.Equal(t, "object", resource.Properties["$[]"].Type)
	assert.Equal(t, "string", resource.Properties["$[].id"].Type)
}

func TestHydrateContract_RefResolves_SubstitutesReferencedSchema(t *testing.T) {
	contract := hydrate(t, refResolvesJSON)

	require.Len(t, contract.Resources, 1)

	var resource model.UploadedResource
	for _, r := range contract.Resources {
		resource = r
	}

	require.Contains(t, resource.Properties, "$")
	require.Contains(t, resource.Properties, "$.id")
	assert.Equal(t, "object", resource.Properties["$"].Type)
	assert.Equal(t, "string", resource.Properties["$.id"].Type)
}

func TestHydrateContract_Unhappy_PanicsOnSchemaTooDeep(t *testing.T) {
	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(cyclicContractJSON), &dslContract))

	contract := model.NewUploadedContract(model.NewParticipant("petstore-app"), "1", cyclicContractJSON)

	assert.PanicsWithValue(
		t,
		"schema Pet is too deep with more than 10 levels",
		func() { _ = dslContract.HydrateContract(contract) },
	)
}
