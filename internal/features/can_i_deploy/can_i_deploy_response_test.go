package can_i_deploy

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/compatibility_checker"
	"github.com/contracttesting/broker/internal/model"
	"github.com/guregu/null"
	"github.com/stretchr/testify/require"
)

func consumes(participant, provider, interaction, endpoint, method, statusCode, version string) *model.Resource {
	resource := &model.Resource{
		Direction:   model.Consumes,
		Interaction: model.Interaction(interaction),
		Endpoint:    endpoint,
		Method:      method,
		Participant: &model.Participant{Name: participant},
	}

	if provider != "" {
		resource.ConsumedProvider = null.StringFrom(provider)
	}
	if statusCode != "" {
		resource.ResponseStatusCode = null.StringFrom(statusCode)
	}
	if version != "" {
		resource.Version = null.StringFrom(version)
	}

	return resource
}

func provides(participant, interaction, endpoint, method, statusCode, version string) *model.Resource {
	resource := &model.Resource{
		Direction:   model.Provides,
		Interaction: model.Interaction(interaction),
		Endpoint:    endpoint,
		Method:      method,
		Participant: &model.Participant{Name: participant},
	}

	if statusCode != "" {
		resource.ResponseStatusCode = null.StringFrom(statusCode)
	}
	if version != "" {
		resource.Version = null.StringFrom(version)
	}

	return resource
}

func reportWith(breaks ...compatibility_checker.BreakingChange) *compatibility_checker.CompatibilityReport {
	report := compatibility_checker.NewCompatibilityReport()
	for _, breakingChange := range breaks {
		report.Append(breakingChange)
	}

	return report
}

// A3 — the deployable path returns the uniform envelope with an empty (non-nil) array.
func TestBuildIncompatibilities_DeployableEnvelope(t *testing.T) {
	report := compatibility_checker.NewCompatibilityReport()

	body := CanIDeployResponseBody{
		Message:           "Contract checked successfully",
		Deployable:        len(report.Breaks) == 0,
		Environment:       "production",
		Incompatibilities: buildIncompatibilities(report, "v1"),
	}

	got, err := json.Marshal(body)
	require.NoError(t, err)
	require.JSONEq(t,
		`{"message":"Contract checked successfully","deployable":true,"environment":"production","incompatibilities":[]}`,
		string(got))
}

// B1 + B2 + C1 — the dual-role pets-v2 scenario splits into exactly two edges, with the
// checked side carrying checkedVersion, the counterpart its own version, property fields
// lifted to the top level, and breaks/incompatibilities in deterministic sorted order.
func TestBuildIncompatibilities_DualRoleSplitsIntoTwoEdges(t *testing.T) {
	report := reportWith(
		// pets consumes users — response type mismatch (pets is the checked consumer).
		compatibility_checker.BreakingChange{
			CheckedResource:     consumes("pets", "users", "rest_response", "/users/*", "get", "200", ""),
			CounterpartResource: provides("users", "rest_response", "/users/*", "get", "200", "v1"),
			Reason:              compatibility_checker.ReasonPropertyTypeMismatch,
			Details: map[string]string{
				"property":                  "$.userId",
				"checked_property_type":     "string",
				"counterpart_property_type": "integer",
			},
		},
		// pets provides /pets — request property missing in the app consumer (pets is the checked provider).
		compatibility_checker.BreakingChange{
			CheckedResource:     provides("pets", "rest_request", "/pets", "post", "", ""),
			CounterpartResource: consumes("app", "pets", "rest_request", "/pets", "post", "", "v1"),
			Reason:              compatibility_checker.ReasonPropertyMissingInConsumer,
			Details:             map[string]string{"property": "$.breed"},
		},
		// pets provides /pets/* — response property missing in the pets provider.
		compatibility_checker.BreakingChange{
			CheckedResource:     provides("pets", "rest_response", "/pets/*", "get", "200", ""),
			CounterpartResource: consumes("app", "pets", "rest_response", "/pets/*", "get", "200", "v1"),
			Reason:              compatibility_checker.ReasonPropertyMissingInProvider,
			Details:             map[string]string{"property": "$.name"},
		},
	)

	got, err := json.Marshal(buildIncompatibilities(report, "v2"))
	require.NoError(t, err)

	require.JSONEq(t, `[
		{
			"consumer": {"name":"app","version":"v1"},
			"provider": {"name":"pets","version":"v2"},
			"checked": "provider",
			"breakingCount": 2,
			"breaks": [
				{"interaction":"request","method":"post","endpoint":"/pets","statusCode":null,"property":"$.breed","reason":"property_missing_in_consumer"},
				{"interaction":"response","method":"get","endpoint":"/pets/*","statusCode":"200","property":"$.name","reason":"property_missing_in_provider"}
			]
		},
		{
			"consumer": {"name":"pets","version":"v2"},
			"provider": {"name":"users","version":"v1"},
			"checked": "consumer",
			"breakingCount": 1,
			"breaks": [
				{"interaction":"response","method":"get","endpoint":"/users/*","statusCode":"200","property":"$.userId","reason":"property_type_mismatch","details":{"checkedPropertyType":"string","counterpartPropertyType":"integer"}}
			]
		}
	]`, string(got))
}

// B1 + B3 — three not_found breaks that collapse under one consumer key today must become
// three separate incompatibilities (one per provider), each a resource-level break.
func TestBuildIncompatibilities_NotFoundFansOutPerProvider(t *testing.T) {
	report := reportWith(
		compatibility_checker.BreakingChange{
			CheckedResource: consumes("app", "users", "rest_response", "/users", "get", "200", ""),
			Reason:          compatibility_checker.ReasonProviderResourceNotFound,
		},
		compatibility_checker.BreakingChange{
			CheckedResource: consumes("app", "auth", "rest_response", "/auth", "get", "200", ""),
			Reason:          compatibility_checker.ReasonProviderResourceNotFound,
		},
		compatibility_checker.BreakingChange{
			CheckedResource: consumes("app", "catalog", "rest_response", "/catalog", "get", "200", ""),
			Reason:          compatibility_checker.ReasonProviderResourceNotFound,
		},
	)

	got, err := json.Marshal(buildIncompatibilities(report, "v1"))
	require.NoError(t, err)

	require.JSONEq(t, `[
		{"consumer":{"name":"app","version":"v1"},"provider":{"name":"auth","version":null},"checked":"consumer","breakingCount":1,
		 "breaks":[{"interaction":"response","method":"get","endpoint":"/auth","statusCode":"200","property":null,"reason":"provider_resource_not_found"}]},
		{"consumer":{"name":"app","version":"v1"},"provider":{"name":"catalog","version":null},"checked":"consumer","breakingCount":1,
		 "breaks":[{"interaction":"response","method":"get","endpoint":"/catalog","statusCode":"200","property":null,"reason":"provider_resource_not_found"}]},
		{"consumer":{"name":"app","version":"v1"},"provider":{"name":"users","version":null},"checked":"consumer","breakingCount":1,
		 "breaks":[{"interaction":"response","method":"get","endpoint":"/users","statusCode":"200","property":null,"reason":"provider_resource_not_found"}]}
	]`, string(got))
}

// B2 — a provider-checked property type mismatch renders the spec's break shape exactly:
// short reason, top-level property, details without the property key, checked "provider".
func TestBuildIncompatibilities_ProviderCheckedPropertyBreak(t *testing.T) {
	report := reportWith(compatibility_checker.BreakingChange{
		CheckedResource:     provides("api", "rest_response", "/things", "get", "200", ""),
		CounterpartResource: consumes("front", "api", "rest_response", "/things", "get", "200", "v1"),
		Reason:              compatibility_checker.ReasonPropertyTypeMismatch,
		Details: map[string]string{
			"property":                  "$.id",
			"checked_property_type":     "string",
			"counterpart_property_type": "integer",
		},
	})

	got, err := json.Marshal(buildIncompatibilities(report, "v1"))
	require.NoError(t, err)

	require.JSONEq(t, `[{
		"consumer": {"name":"front","version":"v1"},
		"provider": {"name":"api","version":"v1"},
		"checked": "provider",
		"breakingCount": 1,
		"breaks": [{"interaction":"response","method":"get","endpoint":"/things","statusCode":"200","property":"$.id","reason":"property_type_mismatch","details":{"checkedPropertyType":"string","counterpartPropertyType":"integer"}}]
	}]`, string(got))
}

// B3 — the not_deployed resource-level reason rides deployed_environments in details when
// present, and omits details entirely when the provider is deployed nowhere.
func TestBuildIncompatibilities_NotDeployedWithAndWithoutEnvironments(t *testing.T) {
	withEnv := reportWith(compatibility_checker.BreakingChange{
		CheckedResource: consumes("front", "api", "rest_response", "/things", "get", "200", ""),
		Reason:          compatibility_checker.ReasonProviderResourceNotDeployedInEnvironment,
		Details:         map[string]string{"deployed_environments": "staging"},
	})

	got, err := json.Marshal(buildIncompatibilities(withEnv, "v1"))
	require.NoError(t, err)
	require.JSONEq(t, `[{
		"consumer":{"name":"front","version":"v1"},"provider":{"name":"api","version":null},"checked":"consumer","breakingCount":1,
		"breaks":[{"interaction":"response","method":"get","endpoint":"/things","statusCode":"200","property":null,"reason":"provider_resource_not_deployed_in_environment","details":{"deployedEnvironments":"staging"}}]
	}]`, string(got))

	nowhere := reportWith(compatibility_checker.BreakingChange{
		CheckedResource: consumes("front", "api", "rest_response", "/things", "get", "200", ""),
		Reason:          compatibility_checker.ReasonProviderResourceNotDeployedInEnvironment,
	})

	got, err = json.Marshal(buildIncompatibilities(nowhere, "v1"))
	require.NoError(t, err)
	require.JSONEq(t, `[{
		"consumer":{"name":"front","version":"v1"},"provider":{"name":"api","version":null},"checked":"consumer","breakingCount":1,
		"breaks":[{"interaction":"response","method":"get","endpoint":"/things","statusCode":"200","property":null,"reason":"provider_resource_not_deployed_in_environment"}]
	}]`, string(got))
}

// C1 — identical input yields byte-identical output across runs.
func TestBuildIncompatibilities_Deterministic(t *testing.T) {
	build := func() []byte {
		report := reportWith(
			compatibility_checker.BreakingChange{
				CheckedResource:     provides("pets", "rest_response", "/pets/*", "get", "200", ""),
				CounterpartResource: consumes("app", "pets", "rest_response", "/pets/*", "get", "200", "v1"),
				Reason:              compatibility_checker.ReasonPropertyMissingInProvider,
				Details:             map[string]string{"property": "$.name"},
			},
			compatibility_checker.BreakingChange{
				CheckedResource:     provides("pets", "rest_request", "/pets", "post", "", ""),
				CounterpartResource: consumes("app", "pets", "rest_request", "/pets", "post", "", "v1"),
				Reason:              compatibility_checker.ReasonPropertyMissingInConsumer,
				Details:             map[string]string{"property": "$.breed"},
			},
			compatibility_checker.BreakingChange{
				CheckedResource:     consumes("pets", "users", "rest_response", "/users/*", "get", "200", ""),
				CounterpartResource: provides("users", "rest_response", "/users/*", "get", "200", "v1"),
				Reason:              compatibility_checker.ReasonPropertyTypeMismatch,
				Details:             map[string]string{"property": "$.userId", "checked_property_type": "string", "counterpart_property_type": "integer"},
			},
		)
		out, err := json.Marshal(buildIncompatibilities(report, "v2"))
		require.NoError(t, err)
		return out
	}

	require.Equal(t, string(build()), string(build()))
}
