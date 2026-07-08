package compatibility_checker_test

import (
	"sort"
	"testing"

	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func variantConsumerResource(interaction model.Interaction, properties map[string]model.Property) *model.PersistedResource {
	return &model.PersistedResource{
		Direction:          model.Consumes,
		Interaction:        interaction,
		ConsumedProvider:   null.StringFrom("api"),
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: null.StringFrom("200"),
		ParticipantName:    "front",
		Properties:         properties,
	}
}

func variantProviderResource(interaction model.Interaction, properties map[string]model.Property) *model.PersistedResource {
	return &model.PersistedResource{
		Direction:          model.Provides,
		Interaction:        interaction,
		Endpoint:           "/things",
		Method:             "get",
		ResponseStatusCode: null.StringFrom("200"),
		ParticipantName:    "api",
		Properties:         properties,
	}
}

func propertyMap(types map[string]string) map[string]model.Property {
	properties := make(map[string]model.Property, len(types))
	for path, propertyType := range types {
		properties[path] = model.Property{Path: path, Type: propertyType}
	}
	return properties
}

func sortedByProperty(breaks []compatibility_checker.ContractBreakingChange) []compatibility_checker.ContractBreakingChange {
	sort.Slice(breaks, func(i, j int) bool {
		return breaks[i].Details["property"] < breaks[j].Details["property"]
	})
	return breaks
}

func TestUnmatchedProviderVariantBreakDetails(t *testing.T) {
	consumer := variantConsumerResource(model.RestResponse, propertyMap(map[string]string{
		"$":        "object",
		"$.prop":   "anyOf",
		"$.prop#0": "number",
		"$.prop#1": "string",
	}))
	provider := variantProviderResource(model.RestResponse, propertyMap(map[string]string{
		"$":        "object",
		"$.prop":   "anyOf",
		"$.prop#0": "string",
		"$.prop#1": "boolean",
	}))

	consumerChecked := compatibility_checker.NewUnmatchedProviderVariantBreak(consumer, provider, "$.prop#1")
	providerChecked := compatibility_checker.NewUnmatchedProviderVariantBreak(provider, consumer, "$.prop#1")

	wantDetails := map[string]string{
		"property":             "$.prop#1",
		"consumerPropertyType": "anyOf<number, string>",
		"providerPropertyType": "boolean",
	}

	assert.Equal(t, compatibility_checker.BreakingReason("property_not_matching_any_variant"), consumerChecked.Reason)
	assert.Equal(t, wantDetails, consumerChecked.Details)
	assert.Equal(t, wantDetails, providerChecked.Details)
	assert.Equal(t, "front", consumerChecked.ConsumerName())
	assert.Equal(t, "api", consumerChecked.ProviderName())
}

func TestUnmatchedConsumerVariantBreakDetails(t *testing.T) {
	consumer := variantConsumerResource(model.RestRequest, propertyMap(map[string]string{
		"$":        "object",
		"$.prop":   "anyOf",
		"$.prop#0": "string",
		"$.prop#1": "boolean",
	}))
	provider := variantProviderResource(model.RestRequest, propertyMap(map[string]string{
		"$":        "object",
		"$.prop":   "anyOf",
		"$.prop#0": "string",
		"$.prop#1": "number",
	}))

	consumerChecked := compatibility_checker.NewUnmatchedConsumerVariantBreak(consumer, provider, "$.prop#1")
	providerChecked := compatibility_checker.NewUnmatchedConsumerVariantBreak(provider, consumer, "$.prop#1")

	wantDetails := map[string]string{
		"property":             "$.prop#1",
		"consumerPropertyType": "boolean",
		"providerPropertyType": "anyOf<string, number>",
	}

	assert.Equal(t, compatibility_checker.ReasonPropertyNotMatchingAnyVariant, consumerChecked.Reason)
	assert.Equal(t, wantDetails, consumerChecked.Details)
	assert.Equal(t, wantDetails, providerChecked.Details)
}

func TestVariantBreakRendersNestedVariantTypes(t *testing.T) {
	consumer := variantConsumerResource(model.RestResponse, propertyMap(map[string]string{
		"$":          "object",
		"$.prop":     "anyOf",
		"$.prop#0":   "string",
		"$.prop#1":   "array",
		"$.prop#1[]": "integer",
	}))
	provider := variantProviderResource(model.RestResponse, propertyMap(map[string]string{
		"$":      "object",
		"$.prop": "boolean",
	}))

	change := compatibility_checker.NewUnmatchedProviderVariantBreak(consumer, provider, "$.prop")

	assert.Equal(t, map[string]string{
		"property":             "$.prop",
		"consumerPropertyType": "anyOf<string, array<integer>>",
		"providerPropertyType": "boolean",
	}, change.Details)
}

// Responses flow provider → consumer: every variant the provider may return
// must be accepted by at least one consumer variant. A consumer accepting
// more than the provider returns is safe.
func TestCheckResources_PrimitiveUnion_Response(t *testing.T) {
	providerUnion := map[string]string{
		"$":        "object",
		"$.prop":   "anyOf",
		"$.prop#0": "string",
		"$.prop#1": "number",
	}
	providerPlain := map[string]string{
		"$":      "object",
		"$.prop": "string",
	}

	cases := []struct {
		name       string
		consumer   map[string]string
		provider   map[string]string
		wantBreaks []map[string]string
	}{
		{
			"consumer accepting more than the provider returns is compatible",
			map[string]string{"$": "object", "$.prop": "anyOf", "$.prop#0": "string", "$.prop#1": "number"},
			providerPlain,
			nil,
		},
		{
			"consumer union equal to the provider union is compatible",
			map[string]string{"$": "object", "$.prop": "anyOf", "$.prop#0": "string", "$.prop#1": "number"},
			providerUnion,
			nil,
		},
		{
			"provider may return a number the consumer does not accept",
			map[string]string{"$": "object", "$.prop": "string"},
			providerUnion,
			[]map[string]string{
				{"property": "$.prop#1", "consumerPropertyType": "string", "providerPropertyType": "number"},
			},
		},
		{
			"provider union where the consumer union misses one variant",
			map[string]string{"$": "object", "$.prop": "anyOf", "$.prop#0": "string", "$.prop#1": "boolean"},
			providerUnion,
			[]map[string]string{
				{"property": "$.prop#1", "consumerPropertyType": "anyOf<string, boolean>", "providerPropertyType": "number"},
			},
		},
		{
			"consumer accepting neither provider variant breaks once per variant",
			map[string]string{"$": "object", "$.prop": "boolean"},
			providerUnion,
			[]map[string]string{
				{"property": "$.prop#0", "consumerPropertyType": "boolean", "providerPropertyType": "string"},
				{"property": "$.prop#1", "consumerPropertyType": "boolean", "providerPropertyType": "number"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			consumer := variantConsumerResource(model.RestResponse, propertyMap(c.consumer))
			provider := variantProviderResource(model.RestResponse, propertyMap(c.provider))

			consumerChecked := compatibility_checker.CheckResources(consumer, provider)
			providerChecked := compatibility_checker.CheckResources(provider, consumer)

			for _, breaks := range [][]compatibility_checker.ContractBreakingChange{consumerChecked, providerChecked} {
				require.Len(t, breaks, len(c.wantBreaks))

				for at, contractBreak := range sortedByProperty(breaks) {
					assert.Equal(t, compatibility_checker.ReasonPropertyNotMatchingAnyVariant, contractBreak.Reason)
					assert.Equal(t, c.wantBreaks[at], contractBreak.Details)
				}
			}
		})
	}
}

// Requests flow consumer → provider: every variant the consumer may send
// must be accepted by at least one provider variant. A provider accepting
// more than the consumer sends is safe.
func TestCheckResources_PrimitiveUnion_Request(t *testing.T) {
	providerUnion := map[string]string{
		"$":        "object",
		"$.prop":   "anyOf",
		"$.prop#0": "string",
		"$.prop#1": "number",
	}
	providerPlain := map[string]string{
		"$":      "object",
		"$.prop": "string",
	}

	cases := []struct {
		name       string
		consumer   map[string]string
		provider   map[string]string
		wantBreaks []map[string]string
	}{
		{
			"consumer sending one of the accepted variants is compatible",
			map[string]string{"$": "object", "$.prop": "string"},
			providerUnion,
			nil,
		},
		{
			"consumer union equal to the provider union is compatible",
			map[string]string{"$": "object", "$.prop": "anyOf", "$.prop#0": "string", "$.prop#1": "number"},
			providerUnion,
			nil,
		},
		{
			"consumer may send a number the plain provider does not accept",
			map[string]string{"$": "object", "$.prop": "anyOf", "$.prop#0": "string", "$.prop#1": "number"},
			providerPlain,
			[]map[string]string{
				{"property": "$.prop#1", "consumerPropertyType": "number", "providerPropertyType": "string"},
			},
		},
		{
			"consumer union with a variant no provider variant accepts",
			map[string]string{"$": "object", "$.prop": "anyOf", "$.prop#0": "string", "$.prop#1": "boolean"},
			providerUnion,
			[]map[string]string{
				{"property": "$.prop#1", "consumerPropertyType": "boolean", "providerPropertyType": "anyOf<string, number>"},
			},
		},
		{
			"consumer sending a type outside the provider union",
			map[string]string{"$": "object", "$.prop": "boolean"},
			providerUnion,
			[]map[string]string{
				{"property": "$.prop", "consumerPropertyType": "boolean", "providerPropertyType": "anyOf<string, number>"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			consumer := variantConsumerResource(model.RestRequest, propertyMap(c.consumer))
			provider := variantProviderResource(model.RestRequest, propertyMap(c.provider))

			consumerChecked := compatibility_checker.CheckResources(consumer, provider)
			providerChecked := compatibility_checker.CheckResources(provider, consumer)

			for _, breaks := range [][]compatibility_checker.ContractBreakingChange{consumerChecked, providerChecked} {
				require.Len(t, breaks, len(c.wantBreaks))

				for at, contractBreak := range sortedByProperty(breaks) {
					assert.Equal(t, compatibility_checker.ReasonPropertyNotMatchingAnyVariant, contractBreak.Reason)
					assert.Equal(t, c.wantBreaks[at], contractBreak.Details)
				}
			}
		})
	}
}

// petDogProviderUnion is the Pet/Dog union as a top-level response schema:
// anyOf [Pet{uuid,name,age}, Dog{uuid,name}].
func petDogProviderUnion() map[string]string {
	return map[string]string{
		"$":        "anyOf",
		"$#0":      "object",
		"$#0.uuid": "string",
		"$#0.name": "string",
		"$#0.age":  "integer",
		"$#1":      "object",
		"$#1.uuid": "string",
		"$#1.name": "string",
	}
}

func TestCheckResources_StructuralUnion_Response(t *testing.T) {
	provider := variantProviderResource(model.RestResponse, propertyMap(petDogProviderUnion()))

	t.Run("consumer reading fields present in every variant is compatible", func(t *testing.T) {
		consumer := variantConsumerResource(model.RestResponse, propertyMap(map[string]string{
			"$":      "object",
			"$.name": "string",
		}))

		assert.Empty(t, compatibility_checker.CheckResources(consumer, provider))
		assert.Empty(t, compatibility_checker.CheckResources(provider, consumer))
	})

	t.Run("consumer requiring a field no variant guarantees breaks per provider variant", func(t *testing.T) {
		consumer := variantConsumerResource(model.RestResponse, propertyMap(map[string]string{
			"$":     "object",
			"$.age": "string",
		}))

		breaks := sortedByProperty(compatibility_checker.CheckResources(consumer, provider))
		require.Len(t, breaks, 2)
		for _, contractBreak := range breaks {
			assert.Equal(t, compatibility_checker.ReasonPropertyNotMatchingAnyVariant, contractBreak.Reason)
		}
		assert.Equal(t, map[string]string{
			"property":             "$#0",
			"consumerPropertyType": "object",
			"providerPropertyType": "object",
		}, breaks[0].Details)
		assert.Equal(t, "$#1", breaks[1].Details["property"])
	})

	t.Run("consumer union covering every provider variant is compatible", func(t *testing.T) {
		consumer := variantConsumerResource(model.RestResponse, propertyMap(map[string]string{
			"$":        "anyOf",
			"$#0":      "object",
			"$#0.uuid": "string",
			"$#0.age":  "integer",
			"$#1":      "object",
			"$#1.uuid": "string",
		}))

		assert.Empty(t, compatibility_checker.CheckResources(consumer, provider))
		assert.Empty(t, compatibility_checker.CheckResources(provider, consumer))
	})
}

// Request direction applies the same union rule with the request leaf rules:
// whatever body shape the consumer may send must satisfy some provider
// variant, and fields unknown to that variant are ignored.
func TestCheckResources_StructuralUnion_Request(t *testing.T) {
	provider := variantProviderResource(model.RestRequest, propertyMap(petDogProviderUnion()))

	t.Run("consumer sending a full Pet body is compatible", func(t *testing.T) {
		consumer := variantConsumerResource(model.RestRequest, propertyMap(map[string]string{
			"$":      "object",
			"$.uuid": "string",
			"$.name": "string",
			"$.age":  "integer",
		}))

		assert.Empty(t, compatibility_checker.CheckResources(consumer, provider))
	})

	t.Run("consumer sending a Dog body with an extra field matches the Dog variant", func(t *testing.T) {
		consumer := variantConsumerResource(model.RestRequest, propertyMap(map[string]string{
			"$":      "object",
			"$.uuid": "string",
			"$.name": "string",
			"$.age":  "string",
		}))

		assert.Empty(t, compatibility_checker.CheckResources(consumer, provider))
	})

	t.Run("consumer missing required fields of every variant breaks once", func(t *testing.T) {
		consumer := variantConsumerResource(model.RestRequest, propertyMap(map[string]string{
			"$":      "object",
			"$.uuid": "string",
		}))

		for _, breaks := range [][]compatibility_checker.ContractBreakingChange{
			compatibility_checker.CheckResources(consumer, provider),
			compatibility_checker.CheckResources(provider, consumer),
		} {
			require.Len(t, breaks, 1)
			assert.Equal(t, compatibility_checker.ReasonPropertyNotMatchingAnyVariant, breaks[0].Reason)
			assert.Equal(t, map[string]string{
				"property":             "$",
				"consumerPropertyType": "object",
				"providerPropertyType": "anyOf<object, object>",
			}, breaks[0].Details)
		}
	})

	t.Run("consumer union of full variant bodies is compatible", func(t *testing.T) {
		consumer := variantConsumerResource(model.RestRequest, propertyMap(map[string]string{
			"$":        "anyOf",
			"$#0":      "object",
			"$#0.uuid": "string",
			"$#0.name": "string",
			"$#0.age":  "integer",
			"$#1":      "object",
			"$#1.uuid": "string",
			"$#1.name": "string",
		}))

		assert.Empty(t, compatibility_checker.CheckResources(consumer, provider))
	})
}

// nestedUnionProvider nests a union inside a variant's object properties and
// inside array items: $.pets[] is anyOf [Pet, Dog] and Pet.prop is itself
// anyOf [string, number].
func nestedUnionProvider() map[string]string {
	return map[string]string{
		"$":                 "object",
		"$.pets":            "array",
		"$.pets[]":          "anyOf",
		"$.pets[]#0":        "object",
		"$.pets[]#0.uuid":   "string",
		"$.pets[]#0.prop":   "anyOf",
		"$.pets[]#0.prop#0": "string",
		"$.pets[]#0.prop#1": "number",
		"$.pets[]#1":        "object",
		"$.pets[]#1.uuid":   "string",
	}
}

func TestCheckResources_NestedUnions_ResolveThroughRecursion(t *testing.T) {
	provider := variantProviderResource(model.RestResponse, propertyMap(nestedUnionProvider()))

	t.Run("consumer accepting every shape the provider may return is compatible", func(t *testing.T) {
		properties := propertyMap(map[string]string{
			"$":               "object",
			"$.pets":          "array",
			"$.pets[]":        "object",
			"$.pets[].uuid":   "string",
			"$.pets[].prop":   "anyOf",
			"$.pets[].prop#0": "string",
			"$.pets[].prop#1": "number",
		})
		// prop is optional because the Dog variant does not return it
		properties["$.pets[].prop"] = model.Property{Path: "$.pets[].prop", Type: "anyOf", Optional: true}
		consumer := variantConsumerResource(model.RestResponse, properties)

		assert.Empty(t, compatibility_checker.CheckResources(consumer, provider))
		assert.Empty(t, compatibility_checker.CheckResources(provider, consumer))
	})

	t.Run("consumer not accepting the inner union breaks at every unhandled provider variant", func(t *testing.T) {
		properties := propertyMap(map[string]string{
			"$":             "object",
			"$.pets":        "array",
			"$.pets[]":      "object",
			"$.pets[].uuid": "string",
			"$.pets[].prop": "boolean",
		})
		properties["$.pets[].prop"] = model.Property{Path: "$.pets[].prop", Type: "boolean", Optional: true}
		consumer := variantConsumerResource(model.RestResponse, properties)

		breaks := sortedByProperty(compatibility_checker.CheckResources(consumer, provider))
		require.Len(t, breaks, 1)
		assert.Equal(t, compatibility_checker.ReasonPropertyNotMatchingAnyVariant, breaks[0].Reason)
		assert.Equal(t, map[string]string{
			"property":             "$.pets[]#0",
			"consumerPropertyType": "object",
			"providerPropertyType": "object",
		}, breaks[0].Details)
	})
}

func TestCheckResources_InnerUnion_BreaksAtInnerVariantPath(t *testing.T) {
	// The inner union reached through plain object recursion: the unmatched
	// provider variant path carries the #N index of the variant the consumer
	// does not accept.
	provider := variantProviderResource(model.RestResponse, propertyMap(map[string]string{
		"$":        "object",
		"$.prop":   "anyOf",
		"$.prop#0": "string",
		"$.prop#1": "number",
	}))
	consumer := variantConsumerResource(model.RestResponse, propertyMap(map[string]string{
		"$":        "object",
		"$.prop":   "anyOf",
		"$.prop#0": "string",
		"$.prop#1": "boolean",
	}))

	breaks := compatibility_checker.CheckResources(consumer, provider)
	require.Len(t, breaks, 1)
	assert.Equal(t, compatibility_checker.ReasonPropertyNotMatchingAnyVariant, breaks[0].Reason)
	assert.Equal(t, map[string]string{
		"property":             "$.prop#1",
		"consumerPropertyType": "anyOf<string, boolean>",
		"providerPropertyType": "number",
	}, breaks[0].Details)
}
