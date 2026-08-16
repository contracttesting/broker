package integration_test

import (
	"context"
	"net/http"
)

const rawParticipantBody = `{"participant":"raw-service"}`

const rawCompactContract = `{"provides":{"rest":{"/pets":{"get":{"responses":{"200":"Pet"}}}}},"schemas":{"Pet":{"type":"object","properties":{"id":{"type":"string"}}}}}`

// Same hydrated resources as rawCompactContract, written out differently: the two
// versions share a snapshot but must not share a raw payload.
const rawExpandedContract = `{
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
    }
  }
}`

func (s *IntegrationSuite) rawPayloadForVersion(version string) string {
	var payload string
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		"SELECT raw_payload FROM contract_versions WHERE version = $1", version,
	).Scan(&payload))

	return payload
}

func (s *IntegrationSuite) TestPublish_EachVersionStampsItsOwnRawPayload() {
	status, _ := s.post("/api/participants", rawParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts",
		`{"participant":"raw-service","version":"v42","contract":`+rawCompactContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts",
		`{"participant":"raw-service","version":"v43","contract":`+rawExpandedContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	// same hydrated resources: v43 aliases the snapshot of v42
	s.Equal(1, s.countRows("contracts"))
	s.Equal(2, s.countRows("contract_versions"))

	s.Equal(rawCompactContract, s.rawPayloadForVersion("v42"))
	s.Equal(rawExpandedContract, s.rawPayloadForVersion("v43"))
}

func (s *IntegrationSuite) TestPublish_RepublishingTheSameVersionKeepsTheFirstRawPayload() {
	status, _ := s.post("/api/participants", rawParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts",
		`{"participant":"raw-service","version":"v42","contract":`+rawCompactContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts",
		`{"participant":"raw-service","version":"v42","contract":`+rawExpandedContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	s.Equal(1, s.countRows("contract_versions"))
	s.Equal(rawCompactContract, s.rawPayloadForVersion("v42"))
}
