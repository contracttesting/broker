package integration_test

import (
	"fmt"
	"net/http"
	"strings"
)

const limitParticipantBody = `{"participant":"limit-service"}`

const limitEndpointCount = 200

// largeContractFragments builds a contract whose wire body lands between fiber's 4 MiB
// default and the 8 MiB the broker configures: many endpoints plus a schema carrying a
// long description, the way a monolith's contract grows.
func largeContractFragments(paddingBytes int) []contractFragment {
	endpoints := &strings.Builder{}
	endpoints.WriteString("provides:\n  rest:\n")
	for index := range limitEndpointCount {
		fmt.Fprintf(endpoints, "    /resources/%04d:\n      get:\n        responses:\n          200: Resource\n", index)
	}

	schemas := fmt.Sprintf(
		"schemas:\n  Resource:\n    type: object\n    description: %s\n    properties:\n      id:\n        type: string\n",
		strings.Repeat("x", paddingBytes),
	)

	return []contractFragment{
		{"endpoints.yaml", endpoints.String()},
		{"schemas.yaml", schemas},
	}
}

func (s *IntegrationSuite) TestPublish_BodyOverFiberDefaultLimit_IsAccepted() {
	status, _ := s.post("/api/participants", limitParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	body := s.publishBody("limit-service", "v1", largeContractFragments(5*1024*1024)...)
	s.Require().Greater(len(body), 4*1024*1024)
	s.Require().Less(len(body), 8*1024*1024)

	status, response := s.post("/api/contracts", body)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, response)

	s.Equal(limitEndpointCount, s.countRows("resources"))
}
