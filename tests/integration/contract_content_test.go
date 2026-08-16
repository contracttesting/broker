package integration_test

import (
	"context"
	"net/http"
)

const contentParticipantBody = `{"participant":"content-service"}`

const compactContract = `{"provides":{"rest":{"/pets":{"get":{"responses":{"200":"Pet"}}}}},"schemas":{"Pet":{"type":"object","properties":{"id":{"type":"string"}}}}}`

// Same hydrated resources as compactContract, written out differently: the two
// versions share a snapshot but must not share the stored contract content.
const expandedContract = `{
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

func (s *IntegrationSuite) contractContentForVersion(version string) string {
	var content string
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		"SELECT contract_content FROM contract_versions WHERE version = $1", version,
	).Scan(&content))

	return content
}

func (s *IntegrationSuite) TestPublish_EachVersionStampsItsOwnContractContent() {
	status, _ := s.post("/api/participants", contentParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts",
		`{"participant":"content-service","version":"v42","contract":`+compactContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts",
		`{"participant":"content-service","version":"v43","contract":`+expandedContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	// same hydrated resources: v43 aliases the snapshot of v42
	s.Equal(1, s.countRows("contracts"))
	s.Equal(2, s.countRows("contract_versions"))

	s.Equal(compactContract, s.contractContentForVersion("v42"))
	s.Equal(expandedContract, s.contractContentForVersion("v43"))
}

func (s *IntegrationSuite) TestPublish_RepublishingTheSameVersionKeepsTheFirstContractContent() {
	status, _ := s.post("/api/participants", contentParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts",
		`{"participant":"content-service","version":"v42","contract":`+compactContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts",
		`{"participant":"content-service","version":"v42","contract":`+expandedContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	s.Equal(1, s.countRows("contract_versions"))
	s.Equal(compactContract, s.contractContentForVersion("v42"))
}
