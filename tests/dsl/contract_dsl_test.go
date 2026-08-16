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
        "self": { "ref": "Pet" }
      }
    }
  }
}`

func TestHydrateContract_Happy_MaterializesResources(t *testing.T) {
	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(happyContractJSON), &dslContract))

	contract := model.NewUploadedContract(0, "petstore-app", "1", happyContractJSON)
	require.NoError(t, dsl.HydrateFragments(
		[]dsl.Fragment{{Source: "api.json", Contract: &dslContract}},
		contract,
	))

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
    "PetRef": { "ref": "Pet" },
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

	contract := model.NewUploadedContract(0, "petstore-app", "1", raw)
	require.NoError(t, dsl.HydrateFragments(
		[]dsl.Fragment{{Source: "api.json", Contract: &dslContract}},
		contract,
	))
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

const wideShallowContractJSON = `{
  "provides": {
    "rest": {
      "/users": {
        "get": {
          "responses": { "200": "UsersList" }
        }
      }
    }
  },
  "schemas": {
    "UsersList": {
      "type": "object",
      "properties": {
        "users": {
          "type": "array",
          "items": { "ref": "User" }
        }
      }
    },
    "User": {
      "type": "object",
      "properties": {
        "userId":        { "type": "string" },
        "another":       { "type": "object" },
        "somethingElse": { "type": "string" },
        "list": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "userId":        { "type": "string" },
              "another":       { "type": "object" },
              "somethingElse": { "type": "string" },
              "list": {
                "type": "array",
                "items": { "type": "object" }
              }
            }
          }
        }
      }
    }
  }
}`

func TestHydrateContract_WideShallowSchema_DoesNotPanicOnDepth(t *testing.T) {
	contract := hydrate(t, wideShallowContractJSON)

	require.Len(t, contract.Resources, 1)

	var resource model.UploadedResource
	for _, r := range contract.Resources {
		resource = r
	}

	require.Contains(t, resource.Properties, "$.users[].list[].list[]")
	assert.Equal(t, "array", resource.Properties["$.users[].list[].list"].Type)
	assert.Equal(t, "string", resource.Properties["$.users[].list[].userId"].Type)
}

const deeplyNestedContractJSON = `{
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

func TestHydrateFragments_Unhappy_RejectsDeeplyNestedSchema(t *testing.T) {
	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(deeplyNestedContractJSON), &dslContract))

	contract := model.NewUploadedContract(0, "petstore-app", "1", deeplyNestedContractJSON)
	err := dsl.HydrateFragments(
		[]dsl.Fragment{{Source: "schemas.yaml", Contract: &dslContract}},
		contract,
	)

	require.EqualError(t, err, "schema Pet is too deep with more than 10 levels (schemas.yaml)")
}

func TestHydrateFragments_Unhappy_RejectsCyclicSchema(t *testing.T) {
	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(cyclicContractJSON), &dslContract))

	contract := model.NewUploadedContract(0, "petstore-app", "1", cyclicContractJSON)
	err := dsl.HydrateFragments(
		[]dsl.Fragment{{Source: "schemas.yaml", Contract: &dslContract}},
		contract,
	)

	require.EqualError(t, err, "schema Pet is too deep with more than 10 levels (schemas.yaml)")
}
