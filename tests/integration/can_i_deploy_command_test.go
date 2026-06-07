package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/contracttesting/cli/internal/components"
	"github.com/contracttesting/cli/internal/features/can_i_deploy"
	"github.com/jarcoal/httpmock"
	"github.com/pterm/pterm"
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
		assert.Contains(t, out.String(), "front can be deployed to production")
		assert.Empty(t, errOut.String())
	})

	t.Run("not-deployable verdict renders the check failure view, exits non-zero, no Error: line", func(t *testing.T) {
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
		        "details": {"property": "$.id", "checked_property_type": "integer", "counterpart_property_type": "string"}
		      },
		      {
		        "reason": "missing_in_consumer",
		        "checked_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "api", "method": "get", "endpoint": "/api/users"},
		        "counterpart_resource": {"direction": "provides", "kind": "rest_request", "method": "get", "endpoint": "/api/users"},
		        "details": {"property": "$.tenant"}
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
		        "reason": "exotic_future_reason",
		        "checked_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "legacy", "method": "delete", "endpoint": "/sessions"}
		      }
		    ],
		    "web": [
		      {
		        "reason": "missing_in_provider",
		        "checked_resource": {"direction": "provides", "kind": "rest_response", "method": "put", "endpoint": "/profile", "response_status_code": "200"},
		        "counterpart_resource": {"direction": "consumes", "kind": "rest_response", "consumed_provider": "front", "method": "put", "endpoint": "/profile", "response_status_code": "200"},
		        "details": {"property": "$.name"}
		      },
		      {
		        "reason": "type_mismatch",
		        "checked_resource": {"direction": "provides", "kind": "rest_response", "method": "put", "endpoint": "/profile", "response_status_code": "200"},
		        "counterpart_resource": {"direction": "consumes", "kind": "rest_response", "consumed_provider": "front", "method": "put", "endpoint": "/profile", "response_status_code": "200"},
		        "details": {"property": "$.age", "checked_property_type": "string", "counterpart_property_type": "integer"}
		      },
		      {
		        "reason": "optional_in_provider_required_in_consumer",
		        "checked_resource": {"direction": "provides", "kind": "rest_response", "method": "put", "endpoint": "/profile", "response_status_code": "200"},
		        "counterpart_resource": {"direction": "consumes", "kind": "rest_response", "consumed_provider": "front", "method": "put", "endpoint": "/profile", "response_status_code": "200"},
		        "details": {"property": "$.email"}
		      },
		      {
		        "reason": "optional_in_consumer_required_in_provider",
		        "checked_resource": {"direction": "provides", "kind": "rest_request", "method": "put", "endpoint": "/profile"},
		        "counterpart_resource": {"direction": "consumes", "kind": "rest_request", "consumed_provider": "front", "method": "put", "endpoint": "/profile"},
		        "details": {"property": "$.phone"}
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

		// Integrations sort by (counterpart, method, path, location) and
		// mismatches by property, so the whole view asserts byte-for-byte.
		want := strings.Join([]string{
			"✗  cannot deploy  front  → production",
			"   6 integrations · 6 mismatches  (2 missing · 2 type · 2 optional)",
			"",
			"  accounts POST /accounts  request",
			"    ✗  accounts does not provide this resource · front consumes it",
			"",
			"  api      GET /api/users  request",
			"    ≠  type     $.id",
			"       api wants string · front sends integer",
			"       send $.id as a string",
			"    ✗  missing  $.tenant",
			"       front omits it · api requires it",
			"       send $.tenant in the request body",
			"",
			"  billing  GET /invoices  response 200",
			"    ✗  billing is not deployed in production · deployed in staging, prod",
			"",
			"  legacy   DELETE /sessions  request",
			"    ✗  exotic_future_reason",
			"",
			"  payments GET /payments  request",
			"    ✗  payments is not deployed in any environment",
			"",
			"  web      PUT /profile  request",
			"    ?  optional $.phone",
			"       front requires it · web sends it as optional",
			"       make $.phone required in the request body",
			"",
			"  web      PUT /profile  response 200",
			"    ≠  type     $.age",
			"       web wants integer · front sends string",
			"       return $.age as a integer",
			"    ?  optional $.email",
			"       web requires it · front sends it as optional",
			"       make $.email required in the response",
			"    ✗  missing  $.name",
			"       front omits it · web requires it",
			"       return $.name in the response",
			"",
			strings.Repeat("─", 64),
			"",
			"✗ 1 check failed — fix the mismatches, publish new pacts, re-run.",
			"",
		}, "\n")
		assert.Equal(t, want, pterm.RemoveColorFromString(out.String()))
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
