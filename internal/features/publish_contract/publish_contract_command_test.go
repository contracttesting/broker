package publish_contract_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/contracttesting/cli/internal/components"
	"github.com/contracttesting/cli/internal/features/publish_contract"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	brokerURL   = "http://localhost:8080"
	participant = "pets-service"
	version     = "v1"
	endpoint    = brokerURL + "/api/contracts"
)

func TestPublishContractCommand(t *testing.T) {
	t.Run("publishes a JSON file as name+version+contract object, prints the message, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := filepath.Join(t.TempDir(), "contract.json")
		require.NoError(t, os.WriteFile(file, []byte(`{"provides":{"rest":{}}}`), 0o600))

		var capturedBody []byte
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				capturedBody = body
				return httpmock.NewStringResponse(http.StatusOK, `{"success":true,"message":"contract publish successful"}`), nil
			})

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--version", version})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		assert.JSONEq(t, `{"name":"pets-service","version":"v1","contract":{"provides":{"rest":{}}}}`, string(capturedBody))
		assert.Contains(t, out.String(), "contract publish successful")
		assert.Empty(t, errOut.String())
	})

	t.Run("transcodes a YAML file into a JSON contract object", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := filepath.Join(t.TempDir(), "contract.yaml")
		require.NoError(t, os.WriteFile(file, []byte("provides:\n  rest: {}\n"), 0o600))

		var capturedBody []byte
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				capturedBody = body
				return httpmock.NewStringResponse(http.StatusOK, `{"success":true,"message":"contract publish successful"}`), nil
			})

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--version", version})

		err := command.Execute()

		require.NoError(t, err)
		var sent map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(capturedBody, &sent))
		assert.JSONEq(t, `{"provides":{"rest":{}}}`, string(sent["contract"]))
	})

	t.Run("missing --participant fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := filepath.Join(t.TempDir(), "contract.json")
		require.NoError(t, os.WriteFile(file, []byte(`{}`), 0o600))

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--version", version})

		err := command.Execute()

		require.Error(t, err)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
}
