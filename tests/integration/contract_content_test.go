package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
)

const contentParticipantBody = `{"participant":"content_service"}`

const singleFileYAML = `# the whole service in one file
provides:
  rest:
    /pets:
      get:
        responses:
          200: Pet   # the only status we serve
schemas:
  Pet:
    type: object
    properties:
      id:
        type: string
`

const endpointsYAML = `# endpoints only
provides:
  rest:
    /pets:
      get:
        responses:
          200: Pet
`

const schemasYAML = `# schemas only
schemas:
  Pet:
    type: object
    properties:
      id:
        type: string
`

func (s *IntegrationSuite) contractContentForVersion(version string) []contractFragment {
	var content string
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		"SELECT contract_content FROM contract_versions WHERE version = $1", version,
	).Scan(&content))

	var fragments []contractFragment
	s.Require().NoError(json.Unmarshal([]byte(content), &fragments))

	return fragments
}

func (s *IntegrationSuite) TestPublish_EachVersionStampsItsOwnFiles() {
	status, _ := s.post("/api/participants", contentParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", s.publishBody("content_service", "v42",
		contractFragment{"api.yaml", singleFileYAML},
	))
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", s.publishBody("content_service", "v43",
		contractFragment{".contracts/api/pets.yaml", endpointsYAML},
		contractFragment{".contracts/api/schemas.yaml", schemasYAML},
	))
	s.Require().Equal(http.StatusOK, status)

	// same hydrated resources: v43 aliases the snapshot of v42
	s.Equal(1, s.countRows("contracts"))
	s.Equal(2, s.countRows("contract_versions"))

	s.Equal(
		[]contractFragment{{"api.yaml", singleFileYAML}},
		s.contractContentForVersion("v42"),
	)
	s.Equal(
		[]contractFragment{
			{".contracts/api/pets.yaml", endpointsYAML},
			{".contracts/api/schemas.yaml", schemasYAML},
		},
		s.contractContentForVersion("v43"),
	)
}

func (s *IntegrationSuite) TestPublish_RepublishingTheSameVersionKeepsTheFirstFiles() {
	status, _ := s.post("/api/participants", contentParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", s.publishBody("content_service", "v42",
		contractFragment{"api.yaml", singleFileYAML},
	))
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", s.publishBody("content_service", "v42",
		contractFragment{".contracts/api/pets.yaml", endpointsYAML},
		contractFragment{".contracts/api/schemas.yaml", schemasYAML},
	))
	s.Require().Equal(http.StatusOK, status)

	s.Equal(1, s.countRows("contract_versions"))
	s.Equal(
		[]contractFragment{{"api.yaml", singleFileYAML}},
		s.contractContentForVersion("v42"),
	)
}
