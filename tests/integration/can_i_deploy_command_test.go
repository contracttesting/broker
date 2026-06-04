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

		responseBody := `{"success":true,"deployable":false,"breaks":{` +
			`"front":[` +
			`{"reason":"provider_resource_not_found",` +
			`"checked_resource":{"direction":"consumes","kind":"rest_request","consumed_provider":"accounts","method":"post","endpoint":"/accounts"}},` +
			`{"reason":"type_mismatch",` +
			`"checked_resource":{"direction":"consumes","kind":"rest_request","consumed_provider":"api","method":"get","endpoint":"/api/users"},` +
			`"counterpart_resource":{"direction":"provides","kind":"rest_request","method":"get","endpoint":"/api/users"},` +
			`"details":{"property":"root.id","consumer_type":"integer","provider_type":"string"}},` +
			`{"reason":"provider_resource_not_deployed_in_environment",` +
			`"checked_resource":{"direction":"consumes","kind":"rest_response","consumed_provider":"billing","method":"get","endpoint":"/invoices","response_status_code":"200"},` +
			`"details":{"deployed_environments":"staging, prod"}}` +
			`],` +
			`"web":[` +
			`{"reason":"missing_in_provider",` +
			`"checked_resource":{"direction":"provides","kind":"rest_request","consumed_provider":"","method":"put","endpoint":"/profile"},` +
			`"counterpart_resource":{"direction":"consumes","kind":"rest_request","consumed_provider":"front","method":"put","endpoint":"/profile"},` +
			`"details":{"property":"root.name"}}` +
			`]}}`
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

		// providers the checked participant depends on (consumer-checked: X=front, Y=consumed_provider)
		assert.Contains(t, output, "front depends on these providers:")
		assert.Contains(t, output, "accounts")
		assert.Contains(t, output, "POST /accounts")
		assert.Contains(t, output, "No POST /accounts (request) was found")
		assert.Contains(t, output, "GET /api/users")
		assert.Contains(t, output, "Consumer front expects root.id as integer but provider api provides string on GET /api/users (request)")
		assert.Contains(t, output, "GET /invoices")
		assert.Contains(t, output, "GET /invoices (response 200) exists but isn't deployed in this environment yet (deployed in: staging, prod)")

		// consumers that depend on the checked participant (provider-checked: X=breaks key, Y=front)
		assert.Contains(t, output, "consumers that depend on front:")
		assert.Contains(t, output, "web")
		assert.Contains(t, output, "PUT /profile")
		assert.Contains(t, output, "Provider front no longer provides property root.name required by consumer web on PUT /profile (request)")

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
