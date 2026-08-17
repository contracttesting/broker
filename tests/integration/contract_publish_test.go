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

const contractBodyParamEndpoint = `{
  "provides": {
    "rest": {
      "/users/{userId}": {
        "get": {
          "responses": {
            "200": "User"
          }
        }
      }
    }
  },
  "schemas": {
    "User": {
      "type": "object",
      "properties": {
        "id": { "type": "string" }
      }
    }
  }
}`

const contractBodyEndpointsFragment = `{
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
  }
}`

const contractBodySchemasFragment = `{
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

const petsEndpointsYAML = `provides:
  rest:
    /pets:
      get:
        responses:
          200: Pets
`

const petsSchemasYAML = `schemas:
  Pets:
    type: array
    items:
      ref: Pet
  Pet:
    type: object
    properties:
      petId:
        type: integer
`

const storeEndpointsYAML = `provides:
  rest:
    /pets:
      delete:
        responses:
          204: Pets
`

const storeTrailingSlashYAML = `provides:
  rest:
    /pets/:
      get:
        responses:
          200: Pets
`

const billingSchemasYAML = `schemas:
  Pet:
    type: object
    properties:
      petId:
        type: integer
`

const petsRequestAndCreatedYAML = `provides:
  rest:
    /pets:
      post:
        request: Pets
        responses:
          201: Pets
`

const petsNotFoundOnlyYAML = `provides:
  rest:
    /pets:
      post:
        responses:
          404: Pets
`

const invoicesConsumerYAML = `consumes:
  payments:
    rest:
      /invoices:
        get:
          responses:
            200: Invoice
`

const invoicesSchemasYAML = `schemas:
  Invoice:
    type: object
    properties:
      total:
        type: integer
`

func (s *IntegrationSuite) TestHappyPath_PublishContract() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"api.json", contractBody}))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("contracts"))
	s.Equal(1, s.countRows("contract_versions"))
	s.Equal(1, s.countRows("resources"))
	s.GreaterOrEqual(s.countRows("properties"), 1)

	var version string
	err := s.Pool.QueryRow(context.Background(),
		"SELECT version FROM contract_versions LIMIT 1",
	).Scan(&version)
	s.Require().NoError(err)
	s.Equal("1", version)
}

func (s *IntegrationSuite) TestPublish_SameVersionSameContent_Returns200NoNewRow() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"api.json", contractBody}))
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"api.json", contractBody}))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublish_SameVersionDifferentContent_Returns409() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"api.json", contractBody}))
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"api.json", contractBodyAlt}))
	s.Equal(http.StatusConflict, status)
	s.JSONEq(`{"message":"contract version already exists with different content"}`, body)

	s.Equal(1, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_MissingContracts() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", `{"participant":"pets-service","version":"a1b2c3d"}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract invalid input"}`, body)
}

func (s *IntegrationSuite) TestPublishContract_EmptyContracts() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", `{"participant":"pets-service","version":"a1b2c3d","contracts":[]}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract invalid input"}`, body)
}

func (s *IntegrationSuite) TestPublishContract_BlankParticipant() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	for _, participant := range []string{"", "   "} {
		status, body := s.post("/api/contracts", s.publishBody(participant, "1", contractFragment{"api.json", contractBody}))
		s.Equal(http.StatusBadRequest, status)
		s.JSONEq(`{"message":"contract invalid input"}`, body)
	}
}

func (s *IntegrationSuite) TestPublishContract_BlankSource() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"  ", contractBody}))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract invalid input"}`, body)
}

func (s *IntegrationSuite) TestPublishContract_UnsupportedExtension() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"notes.txt", contractBody}))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"unsupported contract file: notes.txt (expected .yaml, .yml or .json)"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_MalformedYAML() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"broken.yaml", "provides: {"}))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"malformed contract file: broken.yaml: [1:11] could not find flow mapping end token '}'\n>  1 | provides: {\n                 ^\n"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_MalformedJSON() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"broken.json", `{"provides":`}))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"malformed contract file: broken.json: unexpected end of JSON input"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_CommitHashVersion() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "a1b2c3d4e5f6", contractFragment{"api.json", contractBody}))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	var version string
	err := s.Pool.QueryRow(context.Background(),
		"SELECT version FROM contract_versions LIMIT 1",
	).Scan(&version)
	s.Require().NoError(err)
	s.Equal("a1b2c3d4e5f6", version)
}

func (s *IntegrationSuite) TestPublishContract_ParamEndpoint_RejectedNothingStored() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1", contractFragment{"api.json", contractBodyParamEndpoint}))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"invalid endpoint \"/users/{userId}\": dynamic path segments must use * (api.json)"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnknownParticipant() {
	status, body := s.post("/api/contracts", s.publishBody("ghost-service", "1", contractFragment{"api.json", contractBody}))
	s.Equal(http.StatusNotFound, status)
	s.JSONEq(`{"message":"contract participant not found"}`, body)
}

func (s *IntegrationSuite) TestPublishContract_RefCrossesFragments() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1",
		contractFragment{"pets.yaml", petsEndpointsYAML},
		contractFragment{"schemas.yaml", petsSchemasYAML},
	))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("resources"))

	rows, err := s.Pool.Query(context.Background(), "SELECT path FROM properties ORDER BY path")
	s.Require().NoError(err)
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		s.Require().NoError(rows.Scan(&path))
		paths = append(paths, path)
	}
	s.Equal([]string{"$", "$[]", "$[].petId"}, paths)
}

func (s *IntegrationSuite) TestPublishContract_EmptyFragmentContributesNothing() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", s.publishBody("pets-service", "1",
		contractFragment{"pets.yaml", petsEndpointsYAML},
		contractFragment{"schemas.yaml", petsSchemasYAML},
		contractFragment{"empty.yaml", ""},
	))
	s.Equal(http.StatusOK, status)

	s.Equal(1, s.countRows("resources"))
}

func (s *IntegrationSuite) TestPublishContract_DifferentMethodsSamePath_Merge() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1",
		contractFragment{"pets.yaml", petsEndpointsYAML},
		contractFragment{"store.yaml", storeEndpointsYAML},
		contractFragment{"schemas.yaml", petsSchemasYAML},
	))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(2, s.countRows("resources"))
}

func (s *IntegrationSuite) TestPublishContract_RequestInOneFragmentStatusInAnother_Merge() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1",
		contractFragment{"pets.yaml", petsRequestAndCreatedYAML},
		contractFragment{"errors.yaml", petsNotFoundOnlyYAML},
		contractFragment{"schemas.yaml", petsSchemasYAML},
	))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(3, s.countRows("resources"))
}

func (s *IntegrationSuite) TestPublishContract_DuplicateResponseResource_Rejected() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1",
		contractFragment{"pets.yaml", petsEndpointsYAML},
		contractFragment{"store.yaml", petsEndpointsYAML},
		contractFragment{"schemas.yaml", petsSchemasYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"duplicate resource: provides GET /pets 200 declared in pets.yaml and store.yaml"}`, body)

	s.Equal(0, s.countRows("contracts"))
	s.Equal(0, s.countRows("resources"))
}

func (s *IntegrationSuite) TestPublishContract_DuplicateRequestResource_Rejected() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1",
		contractFragment{"pets.yaml", petsRequestAndCreatedYAML},
		contractFragment{"store.yaml", petsRequestAndCreatedYAML},
		contractFragment{"schemas.yaml", petsSchemasYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"duplicate resource: provides POST /pets request declared in pets.yaml and store.yaml"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_DuplicateConsumedResource_Rejected() {
	status, _ := s.post("/api/participants", `{"participant":"front"}`)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("front", "1",
		contractFragment{"a.yaml", invoicesConsumerYAML},
		contractFragment{"b.yaml", invoicesConsumerYAML},
		contractFragment{"schemas.yaml", invoicesSchemasYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"duplicate resource: consumes payments GET /invoices 200 declared in a.yaml and b.yaml"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_TrailingSlashCountsAsDuplicate() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1",
		contractFragment{"pets.yaml", petsEndpointsYAML},
		contractFragment{"store.yaml", storeTrailingSlashYAML},
		contractFragment{"schemas.yaml", petsSchemasYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"duplicate resource: provides GET /pets 200 declared in pets.yaml and store.yaml"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_DuplicateSchema_Rejected() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "1",
		contractFragment{"pets.yaml", petsEndpointsYAML},
		contractFragment{"schemas.yaml", petsSchemasYAML},
		contractFragment{"billing.yaml", billingSchemasYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"duplicate schema: Pet declared in schemas.yaml and billing.yaml"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_SameContentSplitInFragments_AliasesTheSnapshot() {
	status, _ := s.post("/api/participants", petsParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", s.publishBody("pets-service", "v42", contractFragment{"api.json", contractBody}))
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("pets-service", "v43",
		contractFragment{"endpoints.json", contractBodyEndpointsFragment},
		contractFragment{"schemas.json", contractBodySchemasFragment},
	))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("contracts"))
	s.Equal(2, s.countRows("contract_versions"))
}
