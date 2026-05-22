package integration_test

import (
	"context"
	"errors"
	"net/http"

	"gopkg.in/h2non/gock.v1"
)

func (s *Suite) TestUpload_Success() {
	gock.New(BrokerURL).
		Post("/contracts").
		MatchHeader("Content-Type", "application/json").
		Reply(http.StatusOK).
		JSON(map[string]any{
			"success": true,
			"message": "contract upload successful",
		})

	result, err := s.Client.Upload(context.Background(), []byte(`{"name":"pets","owner":"team"}`))

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.Success)
	s.True(result.Success.Success)
	s.Equal("contract upload successful", result.Success.Message)
	s.Nil(result.BreakingChanges)
	s.True(gock.IsDone())
}

func (s *Suite) TestUpload_BreakingChanges() {
	gock.New(BrokerURL).
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

	result, err := s.Client.Upload(context.Background(), []byte(`{}`))

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.BreakingChanges)
	s.Nil(result.Success)
	s.False(result.BreakingChanges.Success)
	s.Require().Len(result.BreakingChanges.BreakingChanges, 1)

	item := result.BreakingChanges.BreakingChanges[0]
	s.Equal("broken-app", item.ContractName)
	s.Equal("app-team", item.ContractOwner)
	s.Equal("consumes", item.Resource.Direction)
	s.Equal("rest_response", item.Resource.Kind)
	s.Equal("pets-service", item.Resource.Provider)
	s.Equal("/pets", item.Resource.Endpoint)
	s.Equal("get", item.Resource.Method)
	s.Equal("200", item.Resource.StatusCode)
	s.Equal("root[].deletedAt", item.Property)
	s.Equal("missing_in_provider", item.Reason)
}

func (s *Suite) TestUpload_InvalidInput() {
	gock.New(BrokerURL).
		Post("/contracts").
		Reply(http.StatusBadRequest).
		JSON(map[string]any{
			"success": false,
			"message": "contract invalid input",
		})

	result, err := s.Client.Upload(context.Background(), []byte(`{}`))

	s.Require().Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "contract invalid input")
}

func (s *Suite) TestUpload_UnexpectedStatus() {
	gock.New(BrokerURL).
		Post("/contracts").
		Reply(http.StatusServiceUnavailable).
		BodyString("upstream down")

	result, err := s.Client.Upload(context.Background(), []byte(`{}`))

	s.Require().Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "503")
	s.Contains(err.Error(), "upstream down")
}

func (s *Suite) TestUpload_TransportFailure() {
	gock.New(BrokerURL).
		Post("/contracts").
		ReplyError(errors.New("connection refused"))

	result, err := s.Client.Upload(context.Background(), []byte(`{}`))

	s.Require().Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "post contracts")
}
