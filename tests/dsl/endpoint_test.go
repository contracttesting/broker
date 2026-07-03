package dsl_test

import (
	"encoding/json"
	"testing"

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

func endpointConsumesJSON(endpoint string) string {
	return `{
  "consumes": {
    "things-service": {
      "rest": {
        "` + endpoint + `": {
          "get": { "responses": { "200": "Thing" } }
        }
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

func hydrateEndpointContract(t *testing.T, raw string) (*model.UploadedContract, error) {
	t.Helper()

	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(raw), &dslContract))

	contract := model.NewUploadedContract(0, "things-app", "1", raw)
	return contract, dslContract.HydrateContract(contract)
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

func TestHydrateContract_EndpointRules_Accepts(t *testing.T) {
	for _, endpoint := range []string{"/users", "/users/*", "/users/*/orders/*", "/"} {
		t.Run(endpoint, func(t *testing.T) {
			contract, err := hydrateEndpointContract(t, endpointProvidesJSON(endpoint))
			require.NoError(t, err)

			resource := singleEndpointResource(t, contract)
			assert.Equal(t, endpoint, resource.Endpoint)
		})
	}
}

func TestHydrateContract_TrailingSlash_NormalizesToSameResource(t *testing.T) {
	slashed, err := hydrateEndpointContract(t, endpointProvidesJSON("/users/"))
	require.NoError(t, err)

	plain, err := hydrateEndpointContract(t, endpointProvidesJSON("/users"))
	require.NoError(t, err)

	slashedResource := singleEndpointResource(t, slashed)
	plainResource := singleEndpointResource(t, plain)

	assert.Equal(t, "/users", slashedResource.Endpoint)
	assert.Equal(t, plainResource.ProviderHash(), slashedResource.ProviderHash())
}

func TestHydrateContract_EndpointRules_Rejects(t *testing.T) {
	cases := []struct {
		endpoint string
		wantErr  string
	}{
		{"/users/{userId}", `invalid endpoint "/users/{userId}": dynamic path segments must use *`},
		{"/users/u*", `invalid endpoint "/users/u*": dynamic path segments must use *`},
		{"users", `invalid endpoint "users": malformed path`},
		{"/users//x", `invalid endpoint "/users//x": malformed path`},
	}

	for _, c := range cases {
		t.Run(c.endpoint, func(t *testing.T) {
			_, err := hydrateEndpointContract(t, endpointProvidesJSON(c.endpoint))
			require.EqualError(t, err, c.wantErr)
		})
	}
}

func TestHydrateContract_ParamEndpoint_ErrorsFromProvidesAndConsumes(t *testing.T) {
	wantErr := `invalid endpoint "/users/{userId}": dynamic path segments must use *`

	_, err := hydrateEndpointContract(t, endpointProvidesJSON("/users/{userId}"))
	require.EqualError(t, err, wantErr)

	_, err = hydrateEndpointContract(t, endpointConsumesJSON("/users/{userId}"))
	require.EqualError(t, err, wantErr)
}
