package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
)

const aliasedProviderThings = `
{
  "provides": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

const aliasedProviderThingsBreaking = `
{
  "provides": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "integer" } } } }
}`

const aliasedConsumerThings = `
{
  "consumes": { "api": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

func (s *IntegrationSuite) TestCanIDeploy_AliasedVersion_IsCheckedAndDeployable() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/environments", `{"participant":"production"}`)

	mustPost("/api/contracts", `{"participant":"api","version":"sha1","contract":`+aliasedProviderThings+`}`)
	mustPost("/api/can-i-deploy", `{"participant":"api","version":"sha1","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"api","version":"sha1","environment":"production"}`)

	// same content under a new version: alias only
	mustPost("/api/contracts", `{"participant":"api","version":"sha2","contract":`+aliasedProviderThings+`}`)
	s.Equal(1, s.countRows("contracts"))

	// the aliased version gets a compatibility check, not contract_not_found
	status, body := s.post("/api/can-i-deploy", `{"participant":"api","version":"sha2","environment":"production"}`)
	s.Require().Equal(http.StatusOK, status)
	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.True(got.Deployable)

	// the record-deployment gate finds the check for the aliased version
	status, body = s.post("/api/deployments", `{"participant":"api","version":"sha2","environment":"production"}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"deployment recorded"}`, body)
}

func (s *IntegrationSuite) TestCanIDeploy_ProviderDeployedUnderAliasedVersion_ResolvesSnapshot() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	canIDeployFront := func() canIDeployResponse {
		status, body := s.post("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)
		s.Require().Equal(http.StatusOK, status)
		var got canIDeployResponse
		s.Require().NoError(json.Unmarshal([]byte(body), &got))
		return got
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/participants", `{"participant":"front"}`)
	mustPost("/api/environments", `{"participant":"production"}`)

	mustPost("/api/contracts", `{"participant":"api","version":"sha1","contract":`+aliasedProviderThings+`}`)
	mustPost("/api/contracts", `{"participant":"api","version":"sha2","contract":`+aliasedProviderThings+`}`)
	mustPost("/api/can-i-deploy", `{"participant":"api","version":"sha2","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"api","version":"sha2","environment":"production"}`)

	// a later breaking snapshot exists but is not deployed
	mustPost("/api/contracts", `{"participant":"api","version":"sha3","contract":`+aliasedProviderThingsBreaking+`}`)

	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+aliasedConsumerThings+`}`)

	// the deployed alias sha2 resolves to the compatible snapshot, not the latest one
	got := canIDeployFront()
	s.True(got.Deployable)
	s.Require().Len(got.Results, 1)
	s.True(got.Results["api"].Deployable)
	s.Require().NotNil(got.Results["api"].ParticipantVersion)
	s.Equal("sha2", *got.Results["api"].ParticipantVersion)
}

func (s *IntegrationSuite) TestRevert_AliasedVersion_EndToEnd() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/participants", `{"participant":"front"}`)
	mustPost("/api/environments", `{"participant":"production"}`)

	// five commits, same contract content: one snapshot, five aliases
	for _, version := range []string{"hash1", "hash2", "hash3", "hash4", "hash5"} {
		mustPost("/api/contracts", `{"participant":"api","version":"`+version+`","contract":`+aliasedProviderThings+`}`)
	}
	s.Equal(1, s.countRows("contracts"))
	s.Equal(5, s.countRows("contract_versions"))

	mustPost("/api/can-i-deploy", `{"participant":"api","version":"hash3","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"api","version":"hash3","environment":"production"}`)
	mustPost("/api/can-i-deploy", `{"participant":"api","version":"hash5","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"api","version":"hash5","environment":"production"}`)

	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+aliasedConsumerThings+`}`)
	mustPost("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)

	// revert: can-i-deploy hash3 runs a fresh check against the deployed counterpart
	checksBefore := s.countRows("compatibility_checks")
	status, body := s.post("/api/can-i-deploy", `{"participant":"api","version":"hash3","environment":"production"}`)
	s.Require().Equal(http.StatusOK, status)
	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.True(got.Deployable)
	s.Require().Len(got.Results, 1)
	s.True(got.Results["front"].Deployable)
	s.Equal(checksBefore+1, s.countRows("compatibility_checks"))

	status, body = s.post("/api/deployments", `{"participant":"api","version":"hash3","environment":"production"}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"deployment recorded"}`, body)

	// hash3 is latest-deployed, and re-deploying it there is a rollback
	var version string
	var rollback bool
	err := s.Pool.QueryRow(context.Background(),
		`SELECT d.version, d.rollback
		 FROM deployments d
		 JOIN participants p ON p.id = d.participant_id
		 WHERE p.name = 'api'
		 ORDER BY d.deployed_at DESC
		 LIMIT 1`,
	).Scan(&version, &rollback)
	s.Require().NoError(err)
	s.Equal("hash3", version)
	s.True(rollback)
}

func (s *IntegrationSuite) TestCanIDeploy_ConsumerDeployedUnderAliasedVersion_ResolvesSnapshot() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/participants", `{"participant":"front"}`)
	mustPost("/api/environments", `{"participant":"production"}`)

	// api is not deployed yet, so front's verdict is red: force the deployment
	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+aliasedConsumerThings+`}`)
	mustPost("/api/contracts", `{"participant":"front","version":"v2","contract":`+aliasedConsumerThings+`}`)
	mustPost("/api/can-i-deploy", `{"participant":"front","version":"v2","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"front","version":"v2","environment":"production","force":true}`)

	mustPost("/api/contracts", `{"participant":"api","version":"sha1","contract":`+aliasedProviderThings+`}`)

	// the deployed consumer alias v2 resolves to its snapshot for the provider's check
	status, body := s.post("/api/can-i-deploy", `{"participant":"api","version":"sha1","environment":"production"}`)
	s.Require().Equal(http.StatusOK, status)
	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.True(got.Deployable)
	s.Require().Len(got.Results, 1)
	s.True(got.Results["front"].Deployable)
	s.Require().NotNil(got.Results["front"].ParticipantVersion)
	s.Equal("v2", *got.Results["front"].ParticipantVersion)
}
