package can_i_deploy_test

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

const (
	brokerURL   = "http://localhost:8080"
	participant = "front"
	version     = "v1"
	environment = "production"
	endpoint    = brokerURL + "/api/can-i-deploy"
)

func TestCanIDeployCommand(t *testing.T) {
	t.Run("deployable verdict posts name+version+environment, prints deployable, exits 0", func(t *testing.T) {
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
		assert.JSONEq(t, `{"name":"front","version":"v1","environment":"production"}`, string(capturedBody))
		assert.Equal(t, "deployable\n", out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("not-deployable verdict prints the breaking-change reasons, exits non-zero, no Error: line", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		responseBody := `{"success":true,"deployable":false,"breaks":{"front":[` +
			`{"reason":"type_mismatch","property":"root.id","human_readable":"Property root.id type mismatch, provider api expects string but consumer front expects integer"},` +
			`{"reason":"missing_in_provider","property":"root.name","human_readable":"Property root.name is missing in provider api"}` +
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
		assert.Contains(t, output, "not deployable\n")
		assert.Contains(t, output, "Property root.id type mismatch, provider api expects string but consumer front expects integer")
		assert.Contains(t, output, "Property root.name is missing in provider api")
		assert.NotContains(t, errOut.String(), "Error:")
	})

	t.Run("non-2xx response writes the broker body to stderr and exits non-zero", func(t *testing.T) {
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
