package dsl_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/builder"
	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func endpointProvidesJSON(endpoint string) string {
	return `{
  "provides": {
    "rest": {
      "` + endpoint + `": {
        "get": { "responses": { "200": "Thing" } }
      }
    }
  },
  "schemas": {
    "Thing": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    }
  }
}`
}

// endpointContract runs the transformation half of the publish pipeline over one valid
// file: the build, which normalizes each endpoint as it keys the resource. Rejection is
// the validator's business, tested there.
func endpointContract(t *testing.T, raw string) *model.UploadedContract {
	t.Helper()

	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(raw), &dslContract))

	fragments := []dsl.Fragment{{Source: "things.yaml", Contract: &dslContract}}

	contract := model.NewUploadedContract(0, "things-app", "1", raw)
	require.NoError(t, builder.Hydrate(fragments, contract))

	return contract
}

func singleEndpointResource(t *testing.T, contract *model.UploadedContract) model.UploadedResource {
	t.Helper()

	require.Len(t, contract.Resources, 1)

	var resource model.UploadedResource
	for _, r := range contract.Resources {
		resource = r
	}

	return resource
}

func TestEndpointRules_Accepts(t *testing.T) {
	for _, endpoint := range []string{"/users", "/users/*", "/users/*/orders/*", "/"} {
		t.Run(endpoint, func(t *testing.T) {
			contract := endpointContract(t, endpointProvidesJSON(endpoint))

			resource := singleEndpointResource(t, contract)
			assert.Equal(t, endpoint, resource.Endpoint)
		})
	}
}

func TestEndpointRules_TrailingSlash_NormalizesToSameResource(t *testing.T) {
	slashed := endpointContract(t, endpointProvidesJSON("/users/"))
	plain := endpointContract(t, endpointProvidesJSON("/users"))

	slashedResource := singleEndpointResource(t, slashed)
	plainResource := singleEndpointResource(t, plain)

	assert.Equal(t, "/users", slashedResource.Endpoint)
	assert.Equal(t, plainResource.ProviderHash(), slashedResource.ProviderHash())
}
