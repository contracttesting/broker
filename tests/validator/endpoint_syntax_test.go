package validator_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/features/publish_contract/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func syntaxProvidesJSON(endpoint string) string {
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

func syntaxConsumesJSON(endpoint string) string {
	return `{
  "consumes": {
    "things_service": {
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

func syntaxViolations(t *testing.T, raw string) []string {
	t.Helper()

	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(raw), &dslContract))

	fragments := []dsl.Fragment{{Source: "things.yaml", Contract: &dslContract}}

	return validator.NewDslValidator().Validate(fragments)
}

func TestEndpointSyntax_Accepts(t *testing.T) {
	for _, endpoint := range []string{"/users", "/users/", "/users/*", "/users/*/orders/*", "/"} {
		t.Run(endpoint, func(t *testing.T) {
			assert.Empty(t, syntaxViolations(t, syntaxProvidesJSON(endpoint)))
		})
	}
}

func TestEndpointSyntax_Rejects(t *testing.T) {
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
			errs := syntaxViolations(t, syntaxProvidesJSON(c.endpoint))

			require.Len(t, errs, 1)
			assert.Equal(t, c.wantErr, errs[0])
		})
	}
}

func TestEndpointSyntax_ParamEndpoint_ErrorsFromProvidesAndConsumes(t *testing.T) {
	wantErr := `invalid endpoint "/users/{userId}": dynamic path segments must use * (things.yaml)`

	errs := syntaxViolations(t, syntaxProvidesJSON("/users/{userId}"))
	require.Len(t, errs, 1)
	assert.Equal(t, wantErr, errs[0])

	errs = syntaxViolations(t, syntaxConsumesJSON("/users/{userId}"))
	require.Len(t, errs, 1)
	assert.Equal(t, wantErr, errs[0])
}
