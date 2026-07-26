package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
)

const gateApiV1Contract = `
{
  "provides": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

const gateApiV2Contract = `
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

const gateBillingV1Contract = `
{
  "provides": { "rest": { "/billing": { "get": { "responses": { "200": "Invoice" } } } } },
  "schemas": { "Invoice": { "type": "object", "properties": { "amount": { "type": "string" } } } }
}`

const gateFrontConsumerContract = `
{
  "consumes": { "api": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

const gateFrontMixedConsumerContract = `
{
  "consumes": {
    "api":     { "rest": { "/things":  { "get": { "responses": { "200": "Thing" } } } } },
    "billing": { "rest": { "/billing": { "get": { "responses": { "200": "Invoice" } } } } }
  },
  "schemas": {
    "Thing":   { "type": "object", "properties": { "id":     { "type": "string" } } },
    "Invoice": { "type": "object", "properties": { "amount": { "type": "integer" } } }
  }
}`

const gateBadConsumerContract = `
{
  "consumes": { "api": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "integer" } } } }
}`

func (s *IntegrationSuite) gateMustPost(path, body string) {
	status, _ := s.post(path, body)
	s.Require().Equalf(http.StatusOK, status, "POST %s %s", path, body)
}

// gateCheckAndDeploy runs the mandatory can-i-deploy and records the deployment.
func (s *IntegrationSuite) gateCheckAndDeploy(participant, version string) {
	s.gateMustPost("/api/can-i-deploy",
		`{"participant":"`+participant+`","version":"`+version+`","environment":"production"}`)
	s.gateMustPost("/api/deployments",
		`{"participant":"`+participant+`","version":"`+version+`","environment":"production"}`)
}

func (s *IntegrationSuite) gateDeploymentCount(participant string) int {
	var count int
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM deployments
		  WHERE participant_id = (SELECT id FROM participants WHERE name = $1)`,
		participant,
	).Scan(&count))
	return count
}

func (s *IntegrationSuite) gateDeploymentForced(participant, version string) bool {
	var forced bool
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT forced FROM deployments
		  WHERE participant_id = (SELECT id FROM participants WHERE name = $1) AND version = $2`,
		participant, version,
	).Scan(&forced))
	return forced
}

func (s *IntegrationSuite) TestRecordDeploymentGate_NoPriorCheckReturns409() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)
	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)

	status, body := s.post("/api/deployments", `{"participant":"api","version":"v1","environment":"production"}`)
	s.Equal(http.StatusConflict, status)
	s.JSONEq(`{
		"message": "run can-i-deploy for api v1 against production first",
		"reason": "compatibility_check_required",
		"results": {}
	}`, body)

	s.Equal(0, s.countRows("deployments"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_ForceDoesNotBypassMissingCheck() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)
	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)

	status, body := s.post("/api/deployments", `{"participant":"api","version":"v1","environment":"production","force":true}`)
	s.Equal(http.StatusConflict, status)
	s.Contains(body, `"reason":"compatibility_check_required"`)

	s.Equal(0, s.countRows("deployments"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_GreenCheckWithNoCounterpartsRecords() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)
	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)

	s.gateMustPost("/api/can-i-deploy", `{"participant":"api","version":"v1","environment":"production"}`)

	status, body := s.post("/api/deployments", `{"participant":"api","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"deployment recorded"}`, body)

	s.Equal(1, s.gateDeploymentCount("api"))
	s.False(s.gateDeploymentForced("api", "v1"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_GreenCheckWithCounterpartRecords() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/participants", `{"participant":"front"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)

	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)
	s.gateCheckAndDeploy("api", "v1")

	s.gateMustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+gateFrontConsumerContract+`}`)
	s.gateMustPost("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)

	status, body := s.post("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"deployment recorded"}`, body)

	s.Equal(1, s.gateDeploymentCount("front"))
	s.False(s.gateDeploymentForced("front", "v1"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_CounterpartBumpedAfterCheck() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/participants", `{"participant":"front"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)

	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)
	s.gateCheckAndDeploy("api", "v1")

	s.gateMustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+gateFrontConsumerContract+`}`)
	s.gateMustPost("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)

	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v2","contract":`+gateApiV2Contract+`}`)
	s.gateCheckAndDeploy("api", "v2")

	status, body := s.post("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Equal(http.StatusConflict, status)
	s.JSONEq(`{
		"message": "run can-i-deploy for front v1 against production first",
		"reason": "compatibility_check_required",
		"results": {
			"api": { "checkedVersion": "v1", "deployedVersion": "v2" }
		}
	}`, body)

	s.Equal(0, s.gateDeploymentCount("front"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_CheckedNotDeployedThenDeployed() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/participants", `{"participant":"front"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)

	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)
	s.gateMustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+gateFrontConsumerContract+`}`)

	// checked while the provider was not deployed anywhere
	s.gateMustPost("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)

	s.gateCheckAndDeploy("api", "v1")

	status, body := s.post("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Equal(http.StatusConflict, status)
	s.JSONEq(`{
		"message": "run can-i-deploy for front v1 against production first",
		"reason": "compatibility_check_required",
		"results": {
			"api": { "checkedVersion": null, "deployedVersion": "v1" }
		}
	}`, body)

	// the drift is not forceable either
	status, body = s.post("/api/deployments", `{"participant":"front","version":"v1","environment":"production","force":true}`)
	s.Equal(http.StatusConflict, status)
	s.Contains(body, `"reason":"compatibility_check_required"`)

	s.Equal(0, s.gateDeploymentCount("front"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_CounterpartUndeployedAfterCheck() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/participants", `{"participant":"front"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)

	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)
	s.gateCheckAndDeploy("api", "v1")

	s.gateMustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+gateFrontConsumerContract+`}`)
	s.gateMustPost("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)

	_, err := s.Pool.Exec(context.Background(),
		`DELETE FROM deployments
		  WHERE participant_id = (SELECT id FROM participants WHERE name = 'api')`)
	s.Require().NoError(err)

	status, body := s.post("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Equal(http.StatusConflict, status)
	s.JSONEq(`{
		"message": "run can-i-deploy for front v1 against production first",
		"reason": "compatibility_check_required",
		"results": {
			"api": { "checkedVersion": "v1", "deployedVersion": null }
		}
	}`, body)

	s.Equal(0, s.gateDeploymentCount("front"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_OneRedCounterpartBlocks() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/participants", `{"participant":"billing"}`)
	s.gateMustPost("/api/participants", `{"participant":"front"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)

	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)
	s.gateCheckAndDeploy("api", "v1")

	s.gateMustPost("/api/contracts", `{"participant":"billing","version":"v1","contract":`+gateBillingV1Contract+`}`)
	s.gateCheckAndDeploy("billing", "v1")

	s.gateMustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+gateFrontMixedConsumerContract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Require().Equal(http.StatusOK, status)
	var checkGot canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &checkGot))
	s.Require().False(checkGot.Deployable)

	status, body = s.post("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Equal(http.StatusConflict, status)
	s.JSONEq(`{
		"message": "front v1 is not deployable to production",
		"reason": "not_deployable",
		"results": {
			"billing": { "counterpartVersion": "v1", "reason": "property_type_mismatch" }
		}
	}`, body)

	s.Equal(0, s.gateDeploymentCount("front"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_ForceBypassesNotDeployable() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/participants", `{"participant":"front"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)

	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)
	s.gateCheckAndDeploy("api", "v1")

	s.gateMustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+gateBadConsumerContract+`}`)
	s.gateMustPost("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)

	status, body := s.post("/api/deployments", `{"participant":"front","version":"v1","environment":"production","force":true}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"deployment recorded despite a not deployable verdict"}`, body)

	s.Equal(1, s.gateDeploymentCount("front"))
	s.True(s.gateDeploymentForced("front", "v1"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_ForceOnGreenCheckRecordsUnforced() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/participants", `{"participant":"front"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)

	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)
	s.gateCheckAndDeploy("api", "v1")

	s.gateMustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+gateFrontConsumerContract+`}`)
	s.gateMustPost("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)

	status, body := s.post("/api/deployments", `{"participant":"front","version":"v1","environment":"production","force":true}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"deployment recorded"}`, body)

	s.Equal(1, s.gateDeploymentCount("front"))
	s.False(s.gateDeploymentForced("front", "v1"))
}

func (s *IntegrationSuite) TestRecordDeploymentGate_OrphanedRedResultRowIgnored() {
	s.gateMustPost("/api/participants", `{"participant":"api"}`)
	s.gateMustPost("/api/participants", `{"participant":"front"}`)
	s.gateMustPost("/api/environments", `{"participant":"production"}`)

	s.gateMustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+gateApiV1Contract+`}`)
	s.gateMustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+gateBadConsumerContract+`}`)

	// deploy the incompatible consumer behind the gate's back
	_, err := s.Pool.Exec(context.Background(),
		`INSERT INTO deployments (participant_id, version, environment_id, rollback)
		 SELECT p.id, 'v1', e.id, false
		   FROM participants p, environments e
		  WHERE p.name = 'front' AND e.name = 'production'`)
	s.Require().NoError(err)

	// the provider's check picks up the deployed incompatible consumer
	status, body := s.post("/api/can-i-deploy", `{"participant":"api","version":"v1","environment":"production"}`)
	s.Require().Equal(http.StatusOK, status)
	var checkGot canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &checkGot))
	s.Require().False(checkGot.Deployable)

	// the consumer is undeployed before the provider records its deployment
	_, err = s.Pool.Exec(context.Background(),
		`DELETE FROM deployments
		  WHERE participant_id = (SELECT id FROM participants WHERE name = 'front')`)
	s.Require().NoError(err)

	status, body = s.post("/api/deployments", `{"participant":"api","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"deployment recorded"}`, body)

	s.Equal(1, s.gateDeploymentCount("api"))
	s.False(s.gateDeploymentForced("api", "v1"))
}
