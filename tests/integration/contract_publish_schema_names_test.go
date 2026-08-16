package integration_test

import (
	"net/http"
)

const namesParticipantBody = `{"participant":"names-service"}`

const namesEndpointsJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": { "200": "Pets" }
        }
      }
    }
  },
  "schemas": {
    "Invoice": {
      "type": "object",
      "properties": { "total": { "type": "integer" } }
    }
  }
}`

const namesRequestJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "post": { "request": "Pet" }
      }
    }
  }
}`

const namesConsumerJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "get": {
            "responses": { "200": "Pets" }
          }
        }
      }
    }
  }
}`

const namesConsumerRequestJSON = `{
  "consumes": {
    "payments": {
      "rest": {
        "/invoices": {
          "post": { "request": "Pet" }
        }
      }
    }
  }
}`

const namesDanglingSchemaJSON = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": { "200": "Pet" }
        }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": { "id": { "type": "string" } }
    },
    "Invoice": {
      "type": "object",
      "properties": { "payment": { "ref": "Payment" } }
    }
  }
}`

func (s *IntegrationSuite) TestPublishContract_UnresolvedResponseSchema() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts",
		`{"participant":"names-service","version":"1","contract":`+namesEndpointsJSON+`}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"unresolved schema name: Pets referenced at provides GET /pets 200"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnresolvedRequestSchema() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts",
		`{"participant":"names-service","version":"1","contract":`+namesRequestJSON+`}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"unresolved schema name: Pet referenced at provides POST /pets request"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnresolvedConsumedResponseSchema() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts",
		`{"participant":"names-service","version":"1","contract":`+namesConsumerJSON+`}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"unresolved schema name: Pets referenced at consumes payments GET /invoices 200"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnresolvedConsumedRequestSchema() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts",
		`{"participant":"names-service","version":"1","contract":`+namesConsumerRequestJSON+`}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"unresolved schema name: Pet referenced at consumes payments POST /invoices request"}`, body)

	s.Equal(0, s.countRows("contracts"))
}

func (s *IntegrationSuite) TestPublishContract_UnresolvedRefInUnreachedSchema() {
	status, _ := s.post("/api/participants", namesParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts",
		`{"participant":"names-service","version":"1","contract":`+namesDanglingSchemaJSON+`}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"unresolved schema name: Payment referenced at Invoice.payment"}`, body)

	s.Equal(0, s.countRows("contracts"))
}
