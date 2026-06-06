package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/contracttesting/cli/internal/components"
	"github.com/contracttesting/cli/internal/features/can_i_deploy"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanIDeployCommand(t *testing.T) {
	const (
		brokerURL   = "http://localhost:8080"
		participant = "front"
		version     = "v1"
		environment = "production"
		endpoint    = brokerURL + "/api/can-i-deploy"
	)

	t.Run("deployable verdict posts participant+version+environment, prints deployable, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		var capturedBody []byte
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				capturedBody = body
				return httpmock.NewStringResponse(http.StatusOK, `{"success":true,"deployable":true}`), nil
			})

		command := can_i_deploy.NewCanIDeployCommand(
			can_i_deploy.NewCanIDeployClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		assert.JSONEq(t, `{"participant":"front","version":"v1","environment":"production"}`, string(capturedBody))
		assert.Contains(t, out.String(), "deployable")
		assert.Empty(t, errOut.String())
	})

	t.Run("not-deployable verdict prints the breaking-change reasons, exits non-zero, no Error: line", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		responseBody := `{
		  "success": true,
		  "deployable": false,
		  "breaks": {
		    "front": [
		      {
		        "reason": "provider_resource_not_found",
		        "checked_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "accounts", "method": "post", "endpoint": "/accounts"}
		      },
		      {
		        "reason": "type_mismatch",
		        "checked_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "api", "method": "get", "endpoint": "/api/users"},
		        "counterpart_resource": {"direction": "provides", "kind": "rest_request", "method": "get", "endpoint": "/api/users"},
		        "details": {"property": "root.id", "checked_property_type": "integer", "counterpart_property_type": "string"}
		      },
		      {
		        "reason": "provider_resource_not_deployed_in_environment",
		        "checked_resource": {"direction": "consumes", "kind": "rest_response", "consumed_provider": "billing", "method": "get", "endpoint": "/invoices", "response_status_code": "200"},
		        "details": {"deployed_environments": "staging, prod"}
		      },
		      {
		        "reason": "provider_resource_not_deployed_in_environment",
		        "checked_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "payments", "method": "get", "endpoint": "/payments"}
		      },
		      {
		        "reason": "missing_in_consumer",
		        "checked_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "api", "method": "get", "endpoint": "/api/users"},
		        "counterpart_resource": {"direction": "provides", "kind": "rest_request", "method": "get", "endpoint": "/api/users"},
		        "details": {"property": "root.tenant"}
		      },
		      {
		        "reason": "exotic_future_reason",
		        "checked_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "legacy", "method": "delete", "endpoint": "/sessions"}
		      }
		    ],
		    "web": [
		      {
		        "reason": "missing_in_provider",
		        "checked_resource": {"direction": "provides", "kind": "rest_request", "consumed_provider": "", "method": "put", "endpoint": "/profile"},
		        "counterpart_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "front", "method": "put", "endpoint": "/profile"},
		        "details": {"property": "root.name"}
		      },
		      {
		        "reason": "type_mismatch",
		        "checked_resource": {"direction": "provides", "kind": "rest_request", "consumed_provider": "", "method": "put", "endpoint": "/profile"},
		        "counterpart_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "front", "method": "put", "endpoint": "/profile"},
		        "details": {"property": "root.age", "checked_property_type": "string", "counterpart_property_type": "integer"}
		      },
		      {
		        "reason": "optional_in_provider_required_in_consumer",
		        "checked_resource": {"direction": "provides", "kind": "rest_request", "consumed_provider": "", "method": "put", "endpoint": "/profile"},
		        "counterpart_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "front", "method": "put", "endpoint": "/profile"},
		        "details": {"property": "root.email"}
		      },
		      {
		        "reason": "optional_in_consumer_required_in_provider",
		        "checked_resource": {"direction": "provides", "kind": "rest_request", "consumed_provider": "", "method": "put", "endpoint": "/profile"},
		        "counterpart_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "front", "method": "put", "endpoint": "/profile"},
		        "details": {"property": "root.phone"}
		      }
		    ]
		  }
		}`
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusOK, responseBody))

		command := can_i_deploy.NewCanIDeployCommand(
			can_i_deploy.NewCanIDeployClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		output := out.String()
		assert.Contains(t, output, "not deployable")

		assert.Contains(t, output, "front depends on these providers:")
		assert.Contains(t, output, "accounts")
		assert.Contains(t, output, "POST /accounts (request)")
		assert.Contains(t, output, "not found in provider")
		assert.Contains(t, output, "GET /api/users (request)")
		assert.Contains(t, output, "root.id: consumer expects integer, provider provides string")
		assert.Contains(t, output, "root.tenant: required, not sent")
		assert.Contains(t, output, "GET /invoices (response 200)")
		assert.Contains(t, output, "not deployed in this environment (deployed in: staging, prod)")
		assert.Contains(t, output, "GET /payments (request)")
		assert.Contains(t, output, "not deployed in any environment")
		assert.Contains(t, output, "DELETE /sessions (request)")
		assert.Contains(t, output, "exotic_future_reason")

		assert.Contains(t, output, "consumers that depend on front:")
		assert.Contains(t, output, "web")
		assert.Contains(t, output, "PUT /profile (request)")
		assert.Contains(t, output, "root.name: required, not provided")
		assert.Contains(t, output, "root.age: consumer expects integer, provider provides string")
		assert.Contains(t, output, "root.email: required, provided as optional")
		assert.Contains(t, output, "root.phone: required, sent as optional")

		assert.NotContains(t, errOut.String(), "Error:")
	})

	t.Run("non-2xx response renders the broker message to stderr and exits non-zero", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusNotFound, `{"success":false,"message":"participant not found"}`))

		command := can_i_deploy.NewCanIDeployCommand(
			can_i_deploy.NewCanIDeployClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{"ghost", "--version", version, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, errOut.String(), "participant not found")
		assert.NotContains(t, errOut.String(), "{")
		assert.NotContains(t, errOut.String(), "\"success\"")
	})

	t.Run("missing --version fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		command := can_i_deploy.NewCanIDeployCommand(
			can_i_deploy.NewCanIDeployClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
}
