package integration_test

import (
	"encoding/json"
	"net/http"
)

const deployedVersionProviderV1 = `
{
  "provides": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

const deployedVersionProviderV2 = `
{
  "provides": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "integer" } } } }
}`

const deployedVersionProviderV3 = `
{
  "provides": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } },
  "schemas": {
    "Thing": {
      "type": "object",
      "properties": {
        "id":   { "type": "string" },
        "name": { "type": "string" }
      }
    }
  }
}`

const deployedVersionConsumerV1 = `
{
  "consumes": { "api": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

func (s *IntegrationSuite) TestCanIDeploy_ConsumerCheckedAgainstDeployedProviderVersion() {
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

	mustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+deployedVersionProviderV1+`}`)
	mustPost("/api/can-i-deploy", `{"participant":"api","version":"v1","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"api","version":"v1","environment":"production"}`)
	mustPost("/api/contracts", `{"participant":"api","version":"v2","contract":`+deployedVersionProviderV2+`}`)

	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+deployedVersionConsumerV1+`}`)

	// v2 broke $.id but only v1 is deployed: the check runs against v1
	got := canIDeployFront()
	s.True(got.Deployable)
	s.Require().Len(got.Results, 1)
	s.True(got.Results["api"].Deployable)
	s.Require().NotNil(got.Results["api"].ParticipantVersion)
	s.Equal("v1", *got.Results["api"].ParticipantVersion)

	// once the breaking v2 is the deployed version, the break is reported
	mustPost("/api/can-i-deploy", `{"participant":"api","version":"v2","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"api","version":"v2","environment":"production"}`)

	got = canIDeployFront()
	s.False(got.Deployable)
	api := got.Results["api"]
	s.False(api.Deployable)
	s.Require().NotNil(api.ParticipantVersion)
	s.Equal("v2", *api.ParticipantVersion)
	breaks := api.Endpoints["/things"]["get"]["200"]
	s.Require().Len(breaks, 1)
	s.Equal("property_type_mismatch", breaks[0].Reason)

	// v3 fixes the break, but v2 stays deployed: the break is still reported
	mustPost("/api/contracts", `{"participant":"api","version":"v3","contract":`+deployedVersionProviderV3+`}`)

	got = canIDeployFront()
	s.False(got.Deployable)
	s.Require().NotNil(got.Results["api"].ParticipantVersion)
	s.Equal("v2", *got.Results["api"].ParticipantVersion)
}

const deployedVersionDependentV1 = `
{
  "consumes": { "api": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

const deployedVersionDependentV2 = `
{
  "consumes": { "api": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "integer" } } } }
}`

const deployedVersionDependentV3 = `
{
  "consumes": { "api": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } } },
  "schemas": {
    "Thing": {
      "type": "object",
      "properties": {
        "id":   { "type": "string" },
        "name": { "type": "string", "optional": true }
      }
    }
  }
}`

const deployedVersionProvidedV1 = `
{
  "provides": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

func (s *IntegrationSuite) TestCanIDeploy_ProviderCheckedAgainstDeployedConsumerVersion() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	canIDeployApi := func() canIDeployResponse {
		status, body := s.post("/api/can-i-deploy", `{"participant":"api","version":"v1","environment":"production"}`)
		s.Require().Equal(http.StatusOK, status)
		var got canIDeployResponse
		s.Require().NoError(json.Unmarshal([]byte(body), &got))
		return got
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/participants", `{"participant":"front"}`)
	mustPost("/api/environments", `{"participant":"production"}`)

	// front's provider is not published yet: force the deployment through its red verdict
	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+deployedVersionDependentV1+`}`)
	mustPost("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"front","version":"v1","environment":"production","force":true}`)
	mustPost("/api/contracts", `{"participant":"front","version":"v2","contract":`+deployedVersionDependentV2+`}`)

	mustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+deployedVersionProvidedV1+`}`)

	// v2 broke the consumer expectation but only v1 is deployed: the check runs against v1
	got := canIDeployApi()
	s.True(got.Deployable)
	s.Require().Len(got.Results, 1)
	s.True(got.Results["front"].Deployable)
	s.Require().NotNil(got.Results["front"].ParticipantVersion)
	s.Equal("v1", *got.Results["front"].ParticipantVersion)

	// once the breaking v2 is the deployed consumer version, the break is reported
	// (api is still undeployed, so front's own verdict stays red: force through it)
	mustPost("/api/can-i-deploy", `{"participant":"front","version":"v2","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"front","version":"v2","environment":"production","force":true}`)

	got = canIDeployApi()
	s.False(got.Deployable)
	front := got.Results["front"]
	s.False(front.Deployable)
	s.Require().NotNil(front.ParticipantVersion)
	s.Equal("v2", *front.ParticipantVersion)
	breaks := front.Endpoints["/things"]["get"]["200"]
	s.Require().Len(breaks, 1)
	s.Equal("property_type_mismatch", breaks[0].Reason)

	// v3 fixes the expectation, but v2 stays deployed: the break is still reported
	mustPost("/api/contracts", `{"participant":"front","version":"v3","contract":`+deployedVersionDependentV3+`}`)

	got = canIDeployApi()
	s.False(got.Deployable)
	s.Require().NotNil(got.Results["front"].ParticipantVersion)
	s.Equal("v2", *got.Results["front"].ParticipantVersion)
}
