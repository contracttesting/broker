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

// singleEndpointResource runs the transformation half of the publish pipeline over one
// valid file: the mapping, which normalizes each endpoint as it keys the resource.
// Rejection is the validator's business, tested there.
func singleEndpointResource(t *testing.T, raw string) model.UploadedResource {
	t.Helper()

	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(raw), &dslContract))

	resources, err := fragmentmapper.ToResourceModels([]dsl.Fragment{{Source: "things.yaml", Contract: &dslContract}})
	require.NoError(t, err)
	require.Len(t, resources, 1)

	return resources[0]
}

func TestEndpointRules_Accepts(t *testing.T) {
	for _, endpoint := range []string{"/users", "/users/*", "/users/*/orders/*", "/"} {
		t.Run(endpoint, func(t *testing.T) {
			resource := singleEndpointResource(t, endpointProvidesJSON(endpoint))

			assert.Equal(t, endpoint, resource.Endpoint)
		})
	}
}

func TestEndpointRules_TrailingSlash_NormalizesToSameResource(t *testing.T) {
	slashed := singleEndpointResource(t, endpointProvidesJSON("/users/"))
	plain := singleEndpointResource(t, endpointProvidesJSON("/users"))

	assert.Equal(t, "/users", slashed.Endpoint)
	assert.Equal(t, plain.ProviderHash(), slashed.ProviderHash())
}
