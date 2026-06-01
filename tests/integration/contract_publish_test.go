package integration_test

import (
	"context"
	"net/http"
)

const contractBody = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": {
            "200": "Pet"
          }
        }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": {
        "id": { "type": "string" },
        "name": { "type": "string" }
      }
    }
  }
}`

const contractBodyAlt = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": {
            "200": "Pet"
          }
        }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": {
        "id": { "type": "integer" },
        "name": { "type": "string" }
      }
    }
  }
}`

func (s *IntegrationSuite) TestHappyPath_PublishContract() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", `{"participant":"pets-service","version":"1","contract":`+contractBody+`}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"success":true,"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("contracts"))
	s.Equal(1, s.countRows("resources"))
	s.GreaterOrEqual(s.countRows("properties"), 1)

	var version string
	err := s.Pool.QueryRow(context.Background(),
		"SELECT version FROM contracts LIMIT 1",
	).Scan(&version)
	s.Require().NoError(err)
	s.Equal("1", version)
}

func (s *IntegrationSuite) TestPublish_SameVersionSameContent_Returns200NoNewRow() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", `{"participant":"pets-service","version":"1","contract":`+contractBody+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", `{"participant":"pets-service","version":"1","contract":`+contractBody+`}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"success":true,"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublish_SameVersionDifferentContent_Returns409() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", `{"participant":"pets-service","version":"1","contract":`+contractBody+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", `{"participant":"pets-service","version":"1","contract":`+contractBodyAlt+`}`)
	s.Equal(http.StatusConflict, status)
	s.JSONEq(`{"success":false,"message":"contract version already exists with different content"}`, body)

	s.Equal(1, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_MissingContract() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", `{"participant":"pets-service","version":"a1b2c3d"}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"success":false,"message":"contract invalid input"}`, body)
}

func (s *IntegrationSuite) TestPublishContract_CommitHashVersion() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", `{"participant":"pets-service","version":"a1b2c3d4e5f6","contract":`+contractBody+`}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"success":true,"message":"contract publish successful"}`, body)

	var version string
	err := s.Pool.QueryRow(context.Background(),
		"SELECT version FROM contracts LIMIT 1",
	).Scan(&version)
	s.Require().NoError(err)
	s.Equal("a1b2c3d4e5f6", version)
}

func (s *IntegrationSuite) TestPublishContract_UnknownParticipant() {
	status, body := s.post("/api/contracts", `{"participant":"ghost-service","version":"1","contract":`+contractBody+`}`)
	s.Equal(http.StatusNotFound, status)
	s.JSONEq(`{"success":false,"message":"contract participant not found"}`, body)
}
