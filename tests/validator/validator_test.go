package validator_test

import (
	"encoding/json"
	"testing"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const outOfRangeStatusJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "600": "Pet" } }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    }
  }
}`

const petsProviderJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": { "responses": { "200": "Pet" } }
      }
    }
  }
}`

const petSchemaJSON = `{
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    }
  }
}`

// endpointSpyRule is a custom rule appended through the public composition API: it
// proves an Append'd rule with a hook fires at the node the hook names.
type endpointSpyRule struct{}

func (endpointSpyRule) Code() string { return "custom.endpoint-spy" }

func (endpointSpyRule) CheckEndpoint(endpoint string, vctx validator.ValidationContext) {
	vctx.Errs.Addf("custom rule saw endpoint %s (%s)", endpoint, vctx.Pos.Source)
}

func validatorFragment(t *testing.T, source, raw string) dsl.Fragment {
	t.Helper()

	contract := &dsl.Contract{}
	require.NoError(t, json.Unmarshal([]byte(raw), contract))

	return dsl.Fragment{Source: source, Contract: contract}
}

func validatorMessages(violations []error) []string {
	messages := make([]string, 0, len(violations))
	for _, violation := range violations {
		messages = append(messages, violation.Error())
	}

	return messages
}

func TestValidator_WithoutPolicyRule_StopsReportingIt(t *testing.T) {
	fragments := []dsl.Fragment{validatorFragment(t, "api.json", outOfRangeStatusJSON)}

	relaxed, err := validator.New().Without("status.out-of-range")
	require.NoError(t, err)

	assert.Empty(t, relaxed.Validate(fragments))
}

func TestValidator_Without_LeavesTheReceiverIntact(t *testing.T) {
	fragments := []dsl.Fragment{validatorFragment(t, "api.json", outOfRangeStatusJSON)}

	original := validator.New()
	_, err := original.Without("status.out-of-range")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"invalid status code 600 at provides GET /pets (api.json)",
	}, validatorMessages(original.Validate(fragments)))
}

func TestValidator_WithoutInvariantRule_ErrorsCitingTheCode(t *testing.T) {
	_, err := validator.New().Without("schema.unresolved-ref")

	assert.EqualError(t, err, "cannot remove rule schema.unresolved-ref: invariant rules are locked")
}

func TestValidator_WithoutUnknownCode_ErrorsCitingTheCode(t *testing.T) {
	_, err := validator.New().Without("no.such-rule")

	assert.EqualError(t, err, "cannot remove rule no.such-rule: not in the catalog")
}

func TestValidator_AppendedCustomRule_FiresAtItsNode(t *testing.T) {
	fragments := []dsl.Fragment{validatorFragment(t, "api.json", outOfRangeStatusJSON)}

	extended := validator.New().Append(endpointSpyRule{})

	assert.Equal(t, []string{
		"custom rule saw endpoint /pets (api.json)",
		"invalid status code 600 at provides GET /pets (api.json)",
	}, validatorMessages(extended.Validate(fragments)))
}

func TestValidator_Append_LeavesTheReceiverIntact(t *testing.T) {
	fragments := []dsl.Fragment{validatorFragment(t, "api.json", outOfRangeStatusJSON)}

	original := validator.New()
	original.Append(endpointSpyRule{})

	assert.Equal(t, []string{
		"invalid status code 600 at provides GET /pets (api.json)",
	}, validatorMessages(original.Validate(fragments)))
}

func TestValidator_ConsecutiveValidates_DontLeakStatefulState(t *testing.T) {
	fragments := []dsl.Fragment{
		validatorFragment(t, "a.json", petsProviderJSON),
		validatorFragment(t, "b.json", petsProviderJSON),
		validatorFragment(t, "c.json", petSchemaJSON),
		validatorFragment(t, "d.json", petSchemaJSON),
	}

	shared := validator.New()

	expected := []string{
		"duplicate resource: provides GET /pets 200 declared in a.json and b.json",
		"duplicate schema: Pet declared in c.json and d.json",
	}

	assert.Equal(t, expected, validatorMessages(shared.Validate(fragments)))
	assert.Equal(t, expected, validatorMessages(shared.Validate(fragments)))
}
