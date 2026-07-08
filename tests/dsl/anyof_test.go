package dsl_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func anyOfProvidesJSON(schemas string) string {
	return `{
  "provides": {
    "rest": {
      "/things": {
        "get": { "responses": { "200": "Thing" } }
      }
    }
  },
  "schemas": ` + schemas + `
}`
}

func hydrateAnyOfContract(t *testing.T, raw string) (*model.UploadedContract, error) {
	t.Helper()

	var dslContract dsl.Contract
	require.NoError(t, json.Unmarshal([]byte(raw), &dslContract))

	contract := model.NewUploadedContract(0, "things-app", "1", raw)
	return contract, dslContract.HydrateContract(contract)
}

func singleAnyOfResource(t *testing.T, contract *model.UploadedContract) model.UploadedResource {
	t.Helper()

	require.Len(t, contract.Resources, 1)

	var resource model.UploadedResource
	for _, r := range contract.Resources {
		resource = r
	}

	return resource
}

func TestSchema_AnyOf_Unmarshals(t *testing.T) {
	raw := `{
  "anyOf": [
    { "type": "string" },
    { "$ref": "Pet" }
  ]
}`

	var schema dsl.Schema
	require.NoError(t, json.Unmarshal([]byte(raw), &schema))

	require.Len(t, schema.AnyOf, 2)
	assert.Equal(t, "string", schema.AnyOf[0].Type)
	assert.Equal(t, "Pet", schema.AnyOf[1].Ref)
	assert.True(t, schema.IsAnyOf())
}

func TestSchema_IsAnyOf_Classifier(t *testing.T) {
	variants := []dsl.Schema{{Type: "string"}, {Type: "number"}}

	cases := []struct {
		name   string
		schema dsl.Schema
		want   bool
	}{
		{"anyOf only", dsl.Schema{AnyOf: variants}, true},
		{"empty anyOf", dsl.Schema{AnyOf: []dsl.Schema{}}, false},
		{"no anyOf", dsl.Schema{Type: "string"}, false},
		{"anyOf with type", dsl.Schema{AnyOf: variants, Type: "object"}, false},
		{"anyOf with ref", dsl.Schema{AnyOf: variants, Ref: "Pet"}, false},
		{"anyOf with properties", dsl.Schema{AnyOf: variants, Properties: map[string]dsl.Schema{"id": {Type: "string"}}}, false},
		{"anyOf with items", dsl.Schema{AnyOf: variants, Items: &dsl.Schema{Type: "string"}}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.schema.IsAnyOf())
		})
	}
}

const anyOfInsideItemsSchemas = `{
  "Pet": {
    "type": "object",
    "properties": {
      "uuid": { "type": "string" },
      "name": { "type": "string" },
      "age":  { "type": "integer" }
    }
  },
  "Dog": {
    "type": "object",
    "properties": {
      "uuid": { "type": "string" },
      "name": { "type": "string" }
    }
  },
  "Thing": {
    "type": "object",
    "properties": {
      "items": {
        "type": "array",
        "items": {
          "anyOf": [ { "$ref": "Pet" }, { "$ref": "Dog" } ]
        }
      }
    }
  }
}`

func TestHydrateContract_AnyOfInsideItems_FlattensVariantsUnderIndexedPaths(t *testing.T) {
	contract, err := hydrateAnyOfContract(t, anyOfProvidesJSON(anyOfInsideItemsSchemas))
	require.NoError(t, err)

	resource := singleAnyOfResource(t, contract)

	wantTypes := map[string]string{
		"$":                "object",
		"$.items":          "array",
		"$.items[]":        "anyOf",
		"$.items[]#0":      "object",
		"$.items[]#0.uuid": "string",
		"$.items[]#0.name": "string",
		"$.items[]#0.age":  "integer",
		"$.items[]#1":      "object",
		"$.items[]#1.uuid": "string",
		"$.items[]#1.name": "string",
	}

	require.Len(t, resource.Properties, len(wantTypes))
	for path, wantType := range wantTypes {
		require.Contains(t, resource.Properties, path)
		assert.Equalf(t, wantType, resource.Properties[path].Type, "path %s", path)
	}
}

const anyOfAtPropertySchemas = `{
  "Thing": {
    "type": "object",
    "properties": {
      "prop": {
        "anyOf": [ { "type": "string" }, { "type": "number" } ]
      }
    }
  }
}`

func TestHydrateContract_AnyOfAtProperty_FlattensPrimitiveVariants(t *testing.T) {
	contract, err := hydrateAnyOfContract(t, anyOfProvidesJSON(anyOfAtPropertySchemas))
	require.NoError(t, err)

	resource := singleAnyOfResource(t, contract)

	assert.Equal(t, "anyOf", resource.Properties["$.prop"].Type)
	assert.Equal(t, "string", resource.Properties["$.prop#0"].Type)
	assert.Equal(t, "number", resource.Properties["$.prop#1"].Type)
	assert.Len(t, resource.Properties, 4)
}

const anyOfTopLevelSchemas = `{
  "Pet": {
    "type": "object",
    "properties": { "uuid": { "type": "string" } }
  },
  "NotFound": {
    "type": "object",
    "properties": { "message": { "type": "string" } }
  },
  "Thing": {
    "anyOf": [ { "$ref": "Pet" }, { "$ref": "NotFound" } ]
  }
}`

func TestHydrateContract_AnyOfAsTopLevelSchema_FlattensFromRoot(t *testing.T) {
	contract, err := hydrateAnyOfContract(t, anyOfProvidesJSON(anyOfTopLevelSchemas))
	require.NoError(t, err)

	resource := singleAnyOfResource(t, contract)

	assert.Equal(t, "anyOf", resource.Properties["$"].Type)
	assert.Equal(t, "object", resource.Properties["$#0"].Type)
	assert.Equal(t, "string", resource.Properties["$#0.uuid"].Type)
	assert.Equal(t, "object", resource.Properties["$#1"].Type)
	assert.Equal(t, "string", resource.Properties["$#1.message"].Type)
	assert.Len(t, resource.Properties, 5)
}

func TestHydrateContract_AnyOfRules_Rejects(t *testing.T) {
	cases := []struct {
		name    string
		schemas string
		wantErr string
	}{
		{
			"empty union",
			`{ "Thing": { "type": "object", "properties": { "prop": { "anyOf": [] } } } }`,
			`invalid schema "Thing": anyOf must not be empty`,
		},
		{
			"single variant",
			`{ "Thing": { "type": "object", "properties": { "prop": { "anyOf": [ { "type": "string" } ] } } } }`,
			`invalid schema "Thing": anyOf with a single variant is just that schema`,
		},
		{
			"combined with type",
			`{ "Thing": { "type": "object", "anyOf": [ { "type": "string" }, { "type": "number" } ] } }`,
			`invalid schema "Thing": anyOf cannot be combined with type, $ref, properties or items`,
		},
		{
			"combined with ref",
			`{ "Thing": { "$ref": "Pet", "anyOf": [ { "type": "string" }, { "type": "number" } ] } }`,
			`invalid schema "Thing": anyOf cannot be combined with type, $ref, properties or items`,
		},
		{
			"combined with properties",
			`{ "Thing": { "properties": { "id": { "type": "string" } }, "anyOf": [ { "type": "string" }, { "type": "number" } ] } }`,
			`invalid schema "Thing": anyOf cannot be combined with type, $ref, properties or items`,
		},
		{
			"combined with items",
			`{ "Thing": { "items": { "type": "string" }, "anyOf": [ { "type": "string" }, { "type": "number" } ] } }`,
			`invalid schema "Thing": anyOf cannot be combined with type, $ref, properties or items`,
		},
		{
			"anyOf directly inside anyOf",
			`{ "Thing": { "anyOf": [ { "type": "string" }, { "anyOf": [ { "type": "number" }, { "type": "boolean" } ] } ] } }`,
			`invalid schema "Thing": anyOf variant cannot itself be anyOf`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := hydrateAnyOfContract(t, anyOfProvidesJSON(c.schemas))
			require.EqualError(t, err, c.wantErr)
		})
	}
}

func TestHydrateContract_AnyOfDuplicateVariants_Rejects(t *testing.T) {
	wantErr := `invalid schema "Thing": anyOf variants must be structurally distinct`

	duplicatePrimitives := `{
  "Thing": { "anyOf": [ { "type": "string" }, { "type": "string" } ] }
}`
	_, err := hydrateAnyOfContract(t, anyOfProvidesJSON(duplicatePrimitives))
	require.EqualError(t, err, wantErr)

	duplicateRefs := `{
  "Pet":     { "type": "object", "properties": { "uuid": { "type": "string" } } },
  "PetCopy": { "type": "object", "properties": { "uuid": { "type": "string" } } },
  "Thing":   { "anyOf": [ { "$ref": "Pet" }, { "$ref": "PetCopy" } ] }
}`
	_, err = hydrateAnyOfContract(t, anyOfProvidesJSON(duplicateRefs))
	require.EqualError(t, err, wantErr)

	distinctRefs := `{
  "Pet": { "type": "object", "properties": { "uuid": { "type": "string" }, "age": { "type": "integer" } } },
  "Dog": { "type": "object", "properties": { "uuid": { "type": "string" } } },
  "Thing": { "anyOf": [ { "$ref": "Pet" }, { "$ref": "Dog" } ] }
}`
	_, err = hydrateAnyOfContract(t, anyOfProvidesJSON(distinctRefs))
	require.NoError(t, err)
}
