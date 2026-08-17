package integration_test

import (
	"net/http"
)

const namesParticipantBody = `{"participant":"names-service"}`

const namesEndpointsYAML = `provides:
  rest:
    /pets:
      get:
        responses:
          200: Pets
`

const namesRequestYAML = `provides:
  rest:
    /pets:
      post:
        request: Pet
`

const namesConsumerYAML = `consumes:
  payments:
    rest:
      /invoices:
        get:
          responses:
            200: Pets
`

const namesConsumerRequestYAML = `consumes:
  payments:
    rest:
      /invoices:
        post:
          request: Pet
`

const namesDanglingSchemaYAML = `schemas:
  Invoice:
    type: object
    properties:
      payment:
        ref: Payment
`

const namesResolvedSchemasYAML = `schemas:
  Invoice:
    type: object
    properties:
      payment:
        ref: Payment
  Payment:
    type: object
    properties:
      total:
        type: integer
`

const namesCyclicSchemasYAML = `schemas:
  Pet:
    type: object
    properties:
      owner:
        ref: Owner
  Owner:
    type: object
    properties:
      pet:
        ref: Pet
`

const namesCyclicEndpointsYAML = `provides:
  rest:
    /pets:
      get:
        responses:
          200: Pet
`

const namesSingleFileJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": { "200": "Inexistente" }
        }
      }
    }
  },
  "schemas": {
    "Pet": { "type": "object", "properties": { "id": { "type": "string" } } }
  }
}`

func (s *IntegrationSuite) TestPublishContract_UnresolvedResponseSchema_MultipleFragments() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("names-service", "1",
		contractFragment{"pets.yaml", namesEndpointsYAML},
		contractFragment{"schemas.yaml", namesResolvedSchemasYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract validation failed","errors":["unresolved schema name: Pets referenced at provides GET /pets 200 (pets.yaml)"]}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnresolvedResponseSchema_SingleFile() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("names-service", "1",
		contractFragment{"api.json", namesSingleFileJSON},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract validation failed","errors":["unresolved schema name: Inexistente referenced at provides GET /pets 200 (api.json)"]}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnresolvedRequestSchema() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("names-service", "1",
		contractFragment{"pets.yaml", namesRequestYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract validation failed","errors":["unresolved schema name: Pet referenced at provides POST /pets request (pets.yaml)"]}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnresolvedConsumedResponseSchema() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("names-service", "1",
		contractFragment{"a.yaml", namesConsumerYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract validation failed","errors":["unresolved schema name: Pets referenced at consumes payments GET /invoices 200 (a.yaml)"]}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnresolvedConsumedRequestSchema() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("names-service", "1",
		contractFragment{"a.yaml", namesConsumerRequestYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract validation failed","errors":["unresolved schema name: Pet referenced at consumes payments POST /invoices request (a.yaml)"]}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnresolvedRefInUnreachedSchema() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("names-service", "1",
		contractFragment{"pets.yaml", namesEndpointsYAML},
		contractFragment{"billing.yaml", namesDanglingSchemaYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract validation failed","errors":["unresolved schema name: Payment referenced at Invoice.payment (billing.yaml)","unresolved schema name: Pets referenced at provides GET /pets 200 (pets.yaml)"]}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_CyclicSchema_RejectedBrokerStaysUp() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("names-service", "1",
		contractFragment{"pets.yaml", namesCyclicEndpointsYAML},
		contractFragment{"schemas.yaml", namesCyclicSchemasYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract validation failed","errors":["schema Owner is too deep with more than 10 levels (schemas.yaml)","schema Pet is too deep with more than 10 levels (schemas.yaml)"]}`, body)

	s.Equal(0, s.countRows("contracts"))

	status, _ = s.post("/api/participants", `{"participant":"still-alive"}`)
	s.Equal(http.StatusOK, status)
}
