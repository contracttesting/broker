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

// endpointContract runs the publish pipeline over one file: normalize, validate, and
// only build what validation accepted.
func endpointContract(t *testing.T, raw string) (*model.UploadedContract, []error) {
	t.Helper()

	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(raw), &dslContract))

	fragments := []dsl.Fragment{{Source: "things.yaml", Contract: &dslContract}}

	if errs := dsl.NewValidator().Validate(fragments); len(errs) > 0 {
		return nil, errs
	}

	dsl.Normalize(fragments)

	contract := model.NewUploadedContract(0, "things-app", "1", raw)
	require.NoError(t, dsl.HydrateFragments(fragments, contract))

	return contract, nil
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
			contract, errs := endpointContract(t, endpointProvidesJSON(endpoint))
			require.Empty(t, errs)

			resource := singleEndpointResource(t, contract)
			assert.Equal(t, endpoint, resource.Endpoint)
		})
	}
}

func TestEndpointRules_TrailingSlash_NormalizesToSameResource(t *testing.T) {
	slashed, errs := endpointContract(t, endpointProvidesJSON("/users/"))
	require.Empty(t, errs)

	plain, errs := endpointContract(t, endpointProvidesJSON("/users"))
	require.Empty(t, errs)

	slashedResource := singleEndpointResource(t, slashed)
	plainResource := singleEndpointResource(t, plain)

	assert.Equal(t, "/users", slashedResource.Endpoint)
	assert.Equal(t, plainResource.ProviderHash(), slashedResource.ProviderHash())
}

func TestEndpointRules_Rejects(t *testing.T) {
	cases := []struct {
		endpoint string
		wantErr  string
	}{
		{"/users/{userId}", `invalid endpoint "/users/{userId}": dynamic path segments must use * (things.yaml)`},
		{"/users/u*", `invalid endpoint "/users/u*": dynamic path segments must use * (things.yaml)`},
		{"users", `invalid endpoint "users": malformed path (things.yaml)`},
		{"/users//x", `invalid endpoint "/users//x": malformed path (things.yaml)`},
		// the message quotes the raw spelling even though the identity under judgment
		// is the normalized one
		{"/users//", `invalid endpoint "/users//": malformed path (things.yaml)`},
	}

	for _, c := range cases {
		t.Run(c.endpoint, func(t *testing.T) {
			_, errs := endpointContract(t, endpointProvidesJSON(c.endpoint))

			require.Len(t, errs, 1)
			assert.EqualError(t, errs[0], c.wantErr)
		})
	}
}

func TestEndpointRules_ParamEndpoint_ErrorsFromProvidesAndConsumes(t *testing.T) {
	wantErr := `invalid endpoint "/users/{userId}": dynamic path segments must use * (things.yaml)`

	_, errs := endpointContract(t, endpointProvidesJSON("/users/{userId}"))
	require.Len(t, errs, 1)
	assert.EqualError(t, errs[0], wantErr)

	_, errs = endpointContract(t, endpointConsumesJSON("/users/{userId}"))
	require.Len(t, errs, 1)
	assert.EqualError(t, errs[0], wantErr)
}
