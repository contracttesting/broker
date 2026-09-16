package dsl_test

import (
	"testing"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func schemaDocument(t *testing.T, source string) dsl.Document {
	t.Helper()

	var document map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(source), &document))

	return document
}

func TestSchemaFromDocument_Primitivo(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `type: string
description: nome do pet
`))

	assert.Equal(t, dsl.Schema{Type: "string", Description: "nome do pet"}, schema)
	assert.True(t, schema.IsPrimitive())
	assert.False(t, schema.IsObject())
	assert.False(t, schema.IsArray())
	assert.False(t, schema.IsRef())
}

func TestSchemaFromDocument_CadaTipoPrimitivo(t *testing.T) {
	for _, primitive := range []string{"string", "integer", "float", "boolean"} {
		t.Run(primitive, func(t *testing.T) {
			schema := dsl.SchemaFromDocument(dsl.Document{"type": primitive})

			assert.True(t, schema.IsPrimitive())
		})
	}
}

func TestSchemaFromDocument_TipoDesconhecidoNaoEhNada(t *testing.T) {
	schema := dsl.SchemaFromDocument(dsl.Document{"type": "auid"})

	assert.Equal(t, dsl.Schema{Type: "auid"}, schema)
	assert.False(t, schema.IsPrimitive())
	assert.False(t, schema.IsObject())
	assert.False(t, schema.IsArray())
	assert.False(t, schema.IsRef())
}

func TestSchemaFromDocument_TypeQueNaoEhStringEhIgnorado(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `type: 42
`))

	assert.Equal(t, dsl.Schema{}, schema)
}

func TestSchemaFromDocument_ObjetoComProperties(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `type: object
properties:
  id:
    type: string
  name:
    type: string
    optional: true
`))

	assert.Equal(t, dsl.Schema{
		Type: "object",
		Properties: map[string]dsl.Schema{
			"id":   {Type: "string"},
			"name": {Type: "string", Optional: true},
		},
	}, schema)
	assert.True(t, schema.IsObject())
	assert.False(t, schema.IsArray())
	assert.False(t, schema.IsPrimitive())
	assert.False(t, schema.IsRef())
}

func TestSchemaFromDocument_ObjetoSemTypeMasComPropertiesEhObjeto(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `properties: {}
`))

	assert.NotNil(t, schema.Properties)
	assert.Empty(t, schema.Properties)
	assert.True(t, schema.IsObject())
}

func TestSchemaFromDocument_PropriedadeQueNaoEhMapaViraSchemaZero(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `type: object
properties:
  id: string
`))

	assert.Equal(t, map[string]dsl.Schema{"id": {}}, schema.Properties)
}

func TestSchemaFromDocument_PropertiesQueNaoEhMapaEhIgnorado(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `properties: id
`))

	assert.Nil(t, schema.Properties)
	assert.False(t, schema.IsObject())
}

func TestSchemaFromDocument_ArrayComItems(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `type: array
items:
  type: integer
`))

	assert.Equal(t, dsl.Schema{Type: "array", Items: &dsl.Schema{Type: "integer"}}, schema)
	assert.True(t, schema.IsArray())
	assert.False(t, schema.IsObject())
	assert.False(t, schema.IsPrimitive())
	assert.False(t, schema.IsRef())
}

func TestSchemaFromDocument_ArrayComItemsVazio(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `type: array
items: {}
`))

	require.NotNil(t, schema.Items)
	assert.Equal(t, dsl.Schema{}, *schema.Items)
	assert.True(t, schema.IsArray())
}

func TestSchemaFromDocument_ArraySemItems(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `type: array
`))

	assert.Nil(t, schema.Items)
	assert.True(t, schema.IsArray())
}

func TestSchemaFromDocument_ItemsQueNaoEhMapaEhIgnorado(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `type: array
items: string
`))

	assert.Equal(t, dsl.Schema{Type: "array"}, schema)
}

func TestSchemaFromDocument_ItemsSemTypeEhArray(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `items:
  type: string
`))

	assert.Equal(t, dsl.Schema{Items: &dsl.Schema{Type: "string"}}, schema)
	assert.True(t, schema.IsArray())
	assert.False(t, schema.IsObject())
}

func TestSchemaFromDocument_Ref(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `ref: Pet
optional: true
`))

	assert.Equal(t, dsl.Schema{Ref: "Pet", Optional: true}, schema)
	assert.True(t, schema.IsRef())
	assert.False(t, schema.IsObject())
	assert.False(t, schema.IsArray())
	assert.False(t, schema.IsPrimitive())
}

func TestSchemaFromDocument_RefComTypeOuEstruturaNaoEhRef(t *testing.T) {
	for name, source := range map[string]string{
		"type":       "ref: Pet\ntype: object\n",
		"properties": "ref: Pet\nproperties: {}\n",
		"items":      "ref: Pet\nitems: {}\n",
	} {
		t.Run(name, func(t *testing.T) {
			schema := dsl.SchemaFromDocument(schemaDocument(t, source))

			assert.Equal(t, "Pet", schema.Ref)
			assert.False(t, schema.IsRef())
		})
	}
}

func TestSchemaFromDocument_Optional(t *testing.T) {
	assert.True(t, dsl.SchemaFromDocument(dsl.Document{"type": "string", "optional": true}).Optional)
	assert.False(t, dsl.SchemaFromDocument(dsl.Document{"type": "string", "optional": false}).Optional)
	assert.False(t, dsl.SchemaFromDocument(dsl.Document{"type": "string", "optional": "true"}).Optional)
	assert.False(t, dsl.SchemaFromDocument(dsl.Document{"type": "string"}).Optional)
}

func TestSchemaFromDocument_RecursaoAninhada(t *testing.T) {
	schema := dsl.SchemaFromDocument(schemaDocument(t, `type: object
properties:
  users:
    type: array
    items:
      type: object
      properties:
        tags:
          type: array
          items:
            ref: Tag
            optional: true
`))

	assert.Equal(t, dsl.Schema{
		Type: "object",
		Properties: map[string]dsl.Schema{
			"users": {
				Type: "array",
				Items: &dsl.Schema{
					Type: "object",
					Properties: map[string]dsl.Schema{
						"tags": {
							Type:  "array",
							Items: &dsl.Schema{Ref: "Tag", Optional: true},
						},
					},
				},
			},
		},
	}, schema)
}

func TestSchemaFromDocument_DocumentoNilOuVazioDaSchemaZero(t *testing.T) {
	assert.Equal(t, dsl.Schema{}, dsl.SchemaFromDocument(nil))
	assert.Equal(t, dsl.Schema{}, dsl.SchemaFromDocument(dsl.Document{}))

	zero := dsl.SchemaFromDocument(nil)
	assert.False(t, zero.IsObject())
	assert.False(t, zero.IsArray())
	assert.False(t, zero.IsPrimitive())
	assert.False(t, zero.IsRef())
}
