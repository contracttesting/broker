package can_i_deploy

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanIDeployResponseBodyUnmarshalsResults(t *testing.T) {
	payload := `{
	  "message": "Contract checked successfully",
	  "participant": "checkout",
	  "version": "v1",
	  "environment": "production",
	  "deployable": false,
	  "results": {
	    "payments": {
	      "deployable": false,
	      "incompatibleCounterpart": {"participantName": "payments", "participantVersion": "v2"},
	      "breaks": [
	        {
	          "checkedResource": {"direction": "consumes", "endpoint": "/payments/*", "method": "GET", "responseStatusCode": "200"},
	          "counterpartResource": {"direction": "provides", "endpoint": "/payments/*", "method": "GET", "responseStatusCode": "200"},
	          "reason": "property_type_mismatch",
	          "details": {"property": "amount", "checkedPropertyType": "string", "counterpartPropertyType": "number"}
	        },
	        {
	          "checkedResource": {"direction": "consumes", "endpoint": "/payments/*", "method": "GET", "responseStatusCode": "200"},
	          "reason": "provider_resource_not_deployed_in_environment",
	          "details": {"deployedEnvironments": "staging, qa"}
	        }
	      ]
	    },
	    "users": {
	      "deployable": true,
	      "incompatibleCounterpart": {"participantName": "users", "participantVersion": null},
	      "breaks": []
	    }
	  }
	}`

	var body CanIDeployResponseBody
	require.NoError(t, json.Unmarshal([]byte(payload), &body))

	assert.Equal(t, "Contract checked successfully", body.Message)
	assert.False(t, body.Deployable)
	assert.Equal(t, "production", body.Environment)
	require.Len(t, body.Results, 2)

	payments := body.Results["payments"]
	assert.False(t, payments.Deployable)
	assert.Equal(t, "payments", payments.IncompatibleCounterpart.ParticipantName)
	require.NotNil(t, payments.IncompatibleCounterpart.ParticipantVersion)
	assert.Equal(t, "v2", *payments.IncompatibleCounterpart.ParticipantVersion)

	require.Len(t, payments.Breaks, 2)

	mismatch := payments.Breaks[0]
	assert.Equal(t, "property_type_mismatch", mismatch.Reason)
	require.NotNil(t, mismatch.CheckedResource)
	assert.Equal(t, "consumes", mismatch.CheckedResource.Direction)
	assert.Equal(t, "/payments/*", mismatch.CheckedResource.Endpoint)
	assert.Equal(t, "GET", mismatch.CheckedResource.Method)
	require.NotNil(t, mismatch.CheckedResource.ResponseStatusCode)
	assert.Equal(t, "200", *mismatch.CheckedResource.ResponseStatusCode)
	require.NotNil(t, mismatch.CounterpartResource)
	assert.Equal(t, "provides", mismatch.CounterpartResource.Direction)
	assert.Equal(t, map[string]string{
		"property":                "amount",
		"checkedPropertyType":     "string",
		"counterpartPropertyType": "number",
	}, mismatch.Details)

	notDeployed := payments.Breaks[1]
	assert.Equal(t, "provider_resource_not_deployed_in_environment", notDeployed.Reason)
	assert.Nil(t, notDeployed.CounterpartResource)
	assert.Equal(t, map[string]string{"deployedEnvironments": "staging, qa"}, notDeployed.Details)

	users := body.Results["users"]
	assert.True(t, users.Deployable)
	assert.Nil(t, users.IncompatibleCounterpart.ParticipantVersion)
}
