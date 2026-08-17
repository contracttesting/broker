package dsl_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const normalizeProvidesJSON = `{
  "provides": {
    "rest": {
      "/users/": {
        "get": { "responses": { "200": "User" } }
      },
      "/": {
        "get": { "responses": { "200": "User" } }
      }
    }
  }
}`

const normalizeConsumesJSON = `{
  "consumes": {
    "users-service": {
      "rest": {
        "/users/": {
          "get": { "responses": { "200": "User" } }
        }
      }
    }
  }
}`

const normalizeCollidingJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "200": "Pet" } }
      },
      "/pets/": {
        "post": { "responses": { "201": "Pet" } }
      }
    }
  }
}`

func normalizeFragment(t *testing.T, source, raw string) dsl.Fragment {
	t.Helper()

	contract := &dsl.Contract{}
	require.NoError(t, json.Unmarshal([]byte(raw), contract))

	return dsl.Fragment{Source: source, Contract: contract}
}

func TestNormalize_TrimsTrailingSlashKeepingRoot(t *testing.T) {
	fragment := normalizeFragment(t, "users.json", normalizeProvidesJSON)

	require.Empty(t, dsl.Normalize([]dsl.Fragment{fragment}))

	assert.Contains(t, fragment.Contract.Provides.Rest, "/users")
	assert.NotContains(t, fragment.Contract.Provides.Rest, "/users/")
	assert.Contains(t, fragment.Contract.Provides.Rest, "/")
}

func TestNormalize_TrimsConsumedEndpoints(t *testing.T) {
	fragment := normalizeFragment(t, "users.json", normalizeConsumesJSON)

	require.Empty(t, dsl.Normalize([]dsl.Fragment{fragment}))

	assert.Contains(t, fragment.Contract.ConsumesServices["users-service"].Rest, "/users")
	assert.NotContains(t, fragment.Contract.ConsumesServices["users-service"].Rest, "/users/")
}

func TestNormalize_BothSpellingsInOneFile_KeepsFirstAndReports(t *testing.T) {
	fragment := normalizeFragment(t, "pets.json", normalizeCollidingJSON)

	errs := dsl.Normalize([]dsl.Fragment{fragment})

	require.Len(t, errs, 1)
	assert.EqualError(t, errs[0], "duplicate endpoint: /pets declared twice in pets.json")

	require.Len(t, fragment.Contract.Provides.Rest, 1)

	kept := fragment.Contract.Provides.Rest["/pets"]
	assert.True(t, kept.Get.IsNonZero())
	assert.False(t, kept.Post.IsNonZero())
}

func TestNormalize_BothSpellingsInDifferentFiles_NormalizeQuietly(t *testing.T) {
	plain := normalizeFragment(t, "a.json", normalizeCollidingJSON)
	delete(plain.Contract.Provides.Rest, "/pets/")

	slashed := normalizeFragment(t, "b.json", normalizeCollidingJSON)
	delete(slashed.Contract.Provides.Rest, "/pets")

	require.Empty(t, dsl.Normalize([]dsl.Fragment{plain, slashed}))

	assert.Contains(t, plain.Contract.Provides.Rest, "/pets")
	assert.Contains(t, slashed.Contract.Provides.Rest, "/pets")
}
