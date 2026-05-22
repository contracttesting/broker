package upload_contract

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

const testBrokerURL = "http://broker.test"

func newTestClient(t *testing.T) *Client {
	t.Helper()
	client := NewClient(testBrokerURL)
	gock.InterceptClient(client.HTTPClient())
	t.Cleanup(func() {
		gock.RestoreClient(client.HTTPClient())
		gock.Off()
	})
	return client
}

func TestUpload_Success(t *testing.T) {
	client := newTestClient(t)

	gock.New(testBrokerURL).
		Post("/contracts").
		MatchHeader("Content-Type", "application/json").
		Reply(http.StatusOK).
		JSON(map[string]any{
			"success": true,
			"message": "contract upload successful",
		})

	result, err := client.Upload(context.Background(), []byte(`{"name":"pets","owner":"team"}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Success)
	assert.True(t, result.Success.Success)
	assert.Equal(t, "contract upload successful", result.Success.Message)
	assert.Nil(t, result.BreakingChanges)
	assert.True(t, gock.IsDone())
}

func TestUpload_BreakingChanges(t *testing.T) {
	client := newTestClient(t)

	gock.New(testBrokerURL).
		Post("/contracts").
		Reply(http.StatusUnprocessableEntity).
		JSON(map[string]any{
			"success": false,
			"message": "contract incompatible with stored counterparts",
			"breakingChanges": []map[string]any{
				{
					"contractName":  "broken-app",
					"contractOwner": "app-team",
					"resource": map[string]any{
						"direction":  "consumes",
						"kind":       "rest_response",
						"provider":   "pets-service",
						"endpoint":   "/pets",
						"method":     "get",
						"statusCode": "200",
					},
					"property": "root[].deletedAt",
					"reason":   "missing_in_provider",
				},
			},
		})

	result, err := client.Upload(context.Background(), []byte(`{}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.BreakingChanges)
	assert.Nil(t, result.Success)
	assert.False(t, result.BreakingChanges.Success)
	require.Len(t, result.BreakingChanges.BreakingChanges, 1)
	item := result.BreakingChanges.BreakingChanges[0]
	assert.Equal(t, "broken-app", item.ContractName)
	assert.Equal(t, "app-team", item.ContractOwner)
	assert.Equal(t, "consumes", item.Resource.Direction)
	assert.Equal(t, "rest_response", item.Resource.Kind)
	assert.Equal(t, "pets-service", item.Resource.Provider)
	assert.Equal(t, "/pets", item.Resource.Endpoint)
	assert.Equal(t, "get", item.Resource.Method)
	assert.Equal(t, "200", item.Resource.StatusCode)
	assert.Equal(t, "root[].deletedAt", item.Property)
	assert.Equal(t, "missing_in_provider", item.Reason)
}

func TestUpload_InvalidInput(t *testing.T) {
	client := newTestClient(t)

	gock.New(testBrokerURL).
		Post("/contracts").
		Reply(http.StatusBadRequest).
		JSON(map[string]any{
			"success": false,
			"message": "contract invalid input",
		})

	result, err := client.Upload(context.Background(), []byte(`{}`))

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "contract invalid input")
}

func TestUpload_UnexpectedStatus(t *testing.T) {
	client := newTestClient(t)

	gock.New(testBrokerURL).
		Post("/contracts").
		Reply(http.StatusServiceUnavailable).
		BodyString("upstream down")

	result, err := client.Upload(context.Background(), []byte(`{}`))

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "503")
	assert.Contains(t, err.Error(), "upstream down")
}

func TestUpload_TransportFailure(t *testing.T) {
	client := newTestClient(t)

	gock.New(testBrokerURL).
		Post("/contracts").
		ReplyError(errors.New("connection refused"))

	result, err := client.Upload(context.Background(), []byte(`{}`))

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "post contracts")
}
