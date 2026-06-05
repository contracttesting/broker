package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
)

// checkedResource mirrors the broker's `checked_resource`/`counterpart_resource`
// JSON object on each break. Only the fields the assertions read are decoded.
type checkedResource struct {
	Direction  string `json:"direction"`
	Kind       string `json:"kind"`
	Provider   string `json:"consumed_provider"`
	Endpoint   string `json:"endpoint"`
	Method     string `json:"method"`
	StatusCode string `json:"response_status_code"`
	Version    string `json:"version"`
}

// breakItem mirrors a single break in the new wire shape: the side under check,
// the (optional) stored counterpart, a reason, and a flat details map. The
// removed fields (human_readable, top-level property, consumer_name,
// provider_name, consumer_type, provider_type, left_resource, right_resource)
// are intentionally absent.
type breakItem struct {
	CheckedResource     checkedResource   `json:"checked_resource"`
	CounterpartResource *checkedResource  `json:"counterpart_resource,omitempty"`
	Reason              string            `json:"reason"`
	Details             map[string]string `json:"details,omitempty"`
}

// canIDeployResponse mirrors the can-i-deploy response envelope: the breaks map
// is keyed by consumer name.
type canIDeployResponse struct {
	Deployable bool                   `json:"deployable"`
	Breaks     map[string][]breakItem `json:"breaks"`
}

// api@v1 provides Thing{id}.
const apiV1ProviderContract = `
{
  "provides": {
    "rest": {
      "/things": {
        "get": {
          "responses": {
            "200": "Thing"
          }
        }
      }
    }
  },
  "schemas": {
    "Thing": {
      "type": "object",
      "properties": {
        "id": { "type": "string" }
      }
    }
  }
}`

// front@v1 consumes only the "id" field that api@v1 provides.
const frontV1ConsumerContract = `
{
  "consumes": {
    "api": {
      "rest": {
        "/things": {
          "get": {
            "responses": {
              "200": "Thing"
            }
          }
        }
      }
    }
  },
  "schemas": {
    "Thing": {
      "type": "object",
      "properties": {
        "id": { "type": "string" }
      }
    }
  }
}`

// front@v2 expects "id" as an integer (api@v1 provides a string) and adds "name"
// (api@v1 does not provide it at all): two breaking changes.
const frontV2ConsumerContract = `
{
  "consumes": {
    "api": {
      "rest": {
        "/things": {
          "get": {
            "responses": {
              "200": "Thing"
            }
          }
        }
      }
    }
  },
  "schemas": {
    "Thing": {
      "type": "object",
      "properties": {
        "id": { "type": "integer" },
        "name": { "type": "string" }
      }
    }
  }
}`

func (s *IntegrationSuite) TestCanIDeploy_HappyPath() {
	// api@v1 provides Thing{id}; publish it and deploy it to production so the
	// compatibility check can resolve it as the provider in that environment.
	status, _ := s.post("/api/participants", `{"participant":"api"}`)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", `{"participant":"api","version":"v1","contract":`+apiV1ProviderContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/environments", `{"participant":"production"}`)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/deployments", `{"participant":"api","version":"v1","environment":"production"}`)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/participants", `{"participant":"front"}`)
	s.Require().Equal(http.StatusOK, status)

	// front@v1 consumes only "id", which api@v1 provides: it is deployable.
	status, _ = s.post("/api/contracts", `{"participant":"front","version":"v1","contract":`+frontV1ConsumerContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"Contract checked successfully","deployable":true}`, body)

	// A compatible decision is persisted as a deployable row.
	s.Equal(1, s.countRows("compatibility_matrix"))
	var v1Deployable bool
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT deployable FROM compatibility_matrix WHERE version = 'v1'`).Scan(&v1Deployable))
	s.True(v1Deployable)

	status, _ = s.post("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Require().Equal(http.StatusOK, status)

	// front@v2 is incompatible with api@v1 on two counts, so it is not deployable
	// and the response carries the breaking changes.
	status, _ = s.post("/api/contracts", `{"participant":"front","version":"v2","contract":`+frontV2ConsumerContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, body = s.post("/api/can-i-deploy", `{"participant":"front","version":"v2","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)

	// The consumer under check is front; the provider is read off the consumer
	// side's consumed_provider, not a removed top-level field.
	frontBreaks := got.Breaks["front"]
	s.Require().Len(frontBreaks, 2)

	byReason := map[string]breakItem{}
	for _, b := range frontBreaks {
		// front runs its own can-i-deploy, so the checked side is the consumer.
		s.Equal("consumes", b.CheckedResource.Direction)
		s.Equal("rest_response", b.CheckedResource.Kind)
		s.Equal("api", b.CheckedResource.Provider)
		s.Equal("/things", b.CheckedResource.Endpoint)
		s.Equal("get", b.CheckedResource.Method)
		s.Equal("200", b.CheckedResource.StatusCode)

		// The stored counterpart is the provider resource.
		s.Require().NotNil(b.CounterpartResource)
		s.Equal("provides", b.CounterpartResource.Direction)

		byReason[b.Reason] = b
	}

	// type_mismatch on root.id carries both checked and counterpart types in details.
	typeMismatch, ok := byReason["type_mismatch"]
	s.Require().True(ok)
	s.Equal("root.id", typeMismatch.Details["property"])
	s.Equal("integer", typeMismatch.Details["checked_property_type"])
	s.Equal("string", typeMismatch.Details["counterpart_property_type"])

	// missing_in_provider on root.name carries only the property in details.
	missing, ok := byReason["missing_in_provider"]
	s.Require().True(ok)
	s.Equal("root.name", missing.Details["property"])
	_, hasCheckedType := missing.Details["checked_property_type"]
	s.False(hasCheckedType)
	_, hasCounterpartType := missing.Details["counterpart_property_type"]
	s.False(hasCounterpartType)

	// The incompatible decision is persisted too, as a non-deployable row.
	s.Equal(2, s.countRows("compatibility_matrix"))
	var v2Deployable bool
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT deployable FROM compatibility_matrix WHERE version = 'v2'`).Scan(&v2Deployable))
	s.False(v2Deployable)
}

// providerCheckedConsumerContract consumes api's Thing{id} as an integer while
// api provides a string: one type_mismatch. It is deployed to production so the
// provider's own can-i-deploy run resolves it as a deployed consumer.
const providerCheckedConsumerContract = `
{
  "consumes": {
    "api": {
      "rest": {
        "/things": {
          "get": {
            "responses": {
              "200": "Thing"
            }
          }
        }
      }
    }
  },
  "schemas": {
    "Thing": {
      "type": "object",
      "properties": {
        "id": { "type": "integer" }
      }
    }
  }
}`

// When the participant running can-i-deploy is the PROVIDER, the check fans out
// over the consumers already deployed in the target environment. The side under
// check is then the provider resource, so checked_resource.direction == "provides"
// and the break is still keyed by the consumer's name.
func (s *IntegrationSuite) TestCanIDeploy_ProviderCheckedAgainstDeployedConsumer() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/participants", `{"participant":"front"}`)
	mustPost("/api/environments", `{"participant":"production"}`)

	// front consumes api incompatibly (id as integer vs string) and is deployed
	// to production, so the provider's check will discover it there.
	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+providerCheckedConsumerContract+`}`)
	mustPost("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)

	// api publishes the provider contract, then asks whether it can deploy to
	// production: the check runs api (the provider) against deployed consumers.
	mustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+apiV1ProviderContract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"api","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)

	// The break is keyed by the consumer's name even though the provider is the
	// participant under check.
	frontBreaks := got.Breaks["front"]
	s.Require().Len(frontBreaks, 1)

	breakItem := frontBreaks[0]
	// The participant under check is the provider.
	s.Equal("provides", breakItem.CheckedResource.Direction)
	s.Equal("/things", breakItem.CheckedResource.Endpoint)
	s.Equal("get", breakItem.CheckedResource.Method)
	s.Equal("200", breakItem.CheckedResource.StatusCode)

	// The stored counterpart is the consumer; its consumed_provider names api.
	s.Require().NotNil(breakItem.CounterpartResource)
	s.Equal("consumes", breakItem.CounterpartResource.Direction)
	s.Equal("api", breakItem.CounterpartResource.Provider)

	s.Equal("type_mismatch", breakItem.Reason)
	s.Equal("root.id", breakItem.Details["property"])
	// Types are position-keyed: the checked side is the provider (string),
	// the stored counterpart is the consumer (integer).
	s.Equal("string", breakItem.Details["checked_property_type"])
	s.Equal("integer", breakItem.Details["counterpart_property_type"])
}

// app@v1 consumes one endpoint from each of three providers. None of them is
// published or deployed, so each dependency resolves to a non-deployable,
// provider-not-found row: one matrix record per dependency.
const appV1ThreeDependenciesContract = `
{
  "consumes": {
    "users":   { "rest": { "/users":   { "get": { "responses": { "200": "User" } } } } },
    "auth":    { "rest": { "/auth":    { "get": { "responses": { "200": "Token" } } } } },
    "catalog": { "rest": { "/catalog": { "get": { "responses": { "200": "Product" } } } } }
  },
  "schemas": {
    "User":    { "type": "object", "properties": { "id":    { "type": "string" } } },
    "Token":   { "type": "object", "properties": { "value": { "type": "string" } } },
    "Product": { "type": "object", "properties": { "id":    { "type": "string" } } }
  }
}`

func (s *IntegrationSuite) TestCanIDeploy_RecordsOneRowPerDependency() {
	status, _ := s.post("/api/participants", `{"participant":"app"}`)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/environments", `{"participant":"production"}`)
	s.Require().Equal(http.StatusOK, status)

	// Only this contract is uploaded — none of its three providers exist.
	status, _ = s.post("/api/contracts",
		`{"participant":"app","version":"v1","contract":`+appV1ThreeDependenciesContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/can-i-deploy",
		`{"participant":"app","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	// Not deployable (no provider is present), but the check still fans out to
	// one break per consumed service.
	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)

	providers := make([]string, 0, len(got.Breaks["app"]))
	for _, b := range got.Breaks["app"] {
		s.Equal("provider_resource_not_found", b.Reason)
		// On a not-found break the consumer is the side under check.
		s.Equal("consumes", b.CheckedResource.Direction)
		// No counterpart was resolved, so it is omitted, and details is empty.
		s.Nil(b.CounterpartResource)
		s.Empty(b.Details)
		providers = append(providers, b.CheckedResource.Provider)
	}
	s.ElementsMatch([]string{"users", "auth", "catalog"}, providers)

	// One matrix record per dependency, each non-deployable.
	s.Equal(3, s.countRows("compatibility_matrix"))
	var nonDeployable int
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM compatibility_matrix WHERE version = 'v1' AND NOT deployable`).
		Scan(&nonDeployable))
	s.Equal(3, nonDeployable)
}

// users@v1, auth@v1 and catalog@v1 each provide one endpoint. app@v1 consumes
// all three: it matches users and auth, but expects catalog's "id" as an
// integer while catalog provides a string — one breaking dependency out of three.
const usersV1ProviderContract = `
{
  "provides": { "rest": { "/users": { "get": { "responses": { "200": "User" } } } } },
  "schemas": { "User": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

const authV1ProviderContract = `
{
  "provides": { "rest": { "/auth": { "get": { "responses": { "200": "Token" } } } } },
  "schemas": { "Token": { "type": "object", "properties": { "value": { "type": "string" } } } }
}`

const catalogV1ProviderContract = `
{
  "provides": { "rest": { "/catalog": { "get": { "responses": { "200": "Product" } } } } },
  "schemas": { "Product": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

const appV1MixedDependenciesContract = `
{
  "consumes": {
    "users":   { "rest": { "/users":   { "get": { "responses": { "200": "User" } } } } },
    "auth":    { "rest": { "/auth":    { "get": { "responses": { "200": "Token" } } } } },
    "catalog": { "rest": { "/catalog": { "get": { "responses": { "200": "Product" } } } } }
  },
  "schemas": {
    "User":    { "type": "object", "properties": { "id":    { "type": "string" } } },
    "Token":   { "type": "object", "properties": { "value": { "type": "string" } } },
    "Product": { "type": "object", "properties": { "id":    { "type": "integer" } } }
  }
}`

func (s *IntegrationSuite) TestCanIDeploy_TwoDeployableOneBreaking() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	for _, name := range []string{"users", "auth", "catalog", "app"} {
		mustPost("/api/participants", `{"participant":"`+name+`"}`)
	}
	mustPost("/api/environments", `{"participant":"production"}`)

	// Publish and deploy each provider to production so the check can resolve them.
	mustPost("/api/contracts", `{"participant":"users","version":"v1","contract":`+usersV1ProviderContract+`}`)
	mustPost("/api/deployments", `{"participant":"users","version":"v1","environment":"production"}`)
	mustPost("/api/contracts", `{"participant":"auth","version":"v1","contract":`+authV1ProviderContract+`}`)
	mustPost("/api/deployments", `{"participant":"auth","version":"v1","environment":"production"}`)
	mustPost("/api/contracts", `{"participant":"catalog","version":"v1","contract":`+catalogV1ProviderContract+`}`)
	mustPost("/api/deployments", `{"participant":"catalog","version":"v1","environment":"production"}`)

	mustPost("/api/contracts", `{"participant":"app","version":"v1","contract":`+appV1MixedDependenciesContract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"app","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	// Two dependencies match; catalog breaks on a type mismatch, so app as a
	// whole is not deployable.
	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)

	s.Require().Len(got.Breaks["app"], 1)
	breakItem := got.Breaks["app"][0]
	s.Equal("catalog", breakItem.CheckedResource.Provider)
	s.Equal("type_mismatch", breakItem.Reason)
	s.Equal("root.id", breakItem.Details["property"])

	// Three records, one per dependency: users and auth deployable, catalog not.
	s.Equal(3, s.countRows("compatibility_matrix"))

	rows, err := s.Pool.Query(context.Background(),
		`SELECT p.name, cm.deployable, cm.counterpart_version
		   FROM compatibility_matrix cm
		   JOIN participants p ON p.id = cm.counterpart_participant_id
		  WHERE cm.version = 'v1'`)
	s.Require().NoError(err)
	defer rows.Close()

	deployableByProvider := map[string]bool{}
	versionByProvider := map[string]string{}
	for rows.Next() {
		var name string
		var deployable bool
		var counterpartVersion string
		s.Require().NoError(rows.Scan(&name, &deployable, &counterpartVersion))
		deployableByProvider[name] = deployable
		versionByProvider[name] = counterpartVersion
	}
	s.Require().NoError(rows.Err())

	s.Equal(map[string]bool{"users": true, "auth": true, "catalog": false}, deployableByProvider)
	// counterpart_version records each provider's deployed version (regression:
	// it was previously always NULL because the resolved resource had no Version).
	s.Equal(map[string]string{"users": "v1", "auth": "v1", "catalog": "v1"}, versionByProvider)
}

// providerThingContract@v1 provides Thing{id}. consumerThingContract@v1 consumes
// it compatibly, so the only thing that can block a deploy is the provider not
// being deployed in the target environment yet.
const providerThingContract = `
{
  "provides": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

const consumerThingContract = `
{
  "consumes": { "api": { "rest": { "/things": { "get": { "responses": { "200": "Thing" } } } } } },
  "schemas": { "Thing": { "type": "object", "properties": { "id": { "type": "string" } } } }
}`

// The provider exists and is deployed to staging, but NOT to production. Checking
// the consumer against production is not deployable, and the break names staging
// as where the provider currently lives.
func (s *IntegrationSuite) TestCanIDeploy_ProviderExistsButNotDeployedInTargetEnv() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/participants", `{"participant":"front"}`)
	mustPost("/api/environments", `{"participant":"production"}`)
	mustPost("/api/environments", `{"participant":"staging"}`)

	mustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+providerThingContract+`}`)
	mustPost("/api/deployments", `{"participant":"api","version":"v1","environment":"staging"}`)

	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+consumerThingContract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))

	s.False(got.Deployable)
	s.Require().Len(got.Breaks["front"], 1)

	breakItem := got.Breaks["front"][0]
	// The consumer is the side under check; its consumed_provider names api.
	s.Equal("consumes", breakItem.CheckedResource.Direction)
	s.Equal("api", breakItem.CheckedResource.Provider)
	s.Equal("provider_resource_not_deployed_in_environment", breakItem.Reason)
	// The provider is deployed to staging, joined into the details map.
	s.Equal("staging", breakItem.Details["deployed_environments"])
}

// The provider exists but has never been deployed anywhere. The break still fires
// (not deployable) and reports no deployed environments at all (details absent).
func (s *IntegrationSuite) TestCanIDeploy_ProviderExistsButDeployedNowhere() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/participants", `{"participant":"front"}`)
	mustPost("/api/environments", `{"participant":"production"}`)

	mustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+providerThingContract+`}`)
	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+consumerThingContract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))

	s.False(got.Deployable)
	s.Require().Len(got.Breaks["front"], 1)

	breakItem := got.Breaks["front"][0]
	s.Equal("provider_resource_not_deployed_in_environment", breakItem.Reason)
	// With no deployed environments there is no deployed_environments entry, and
	// since that is the only detail this break would carry, details is absent.
	_, hasDeployedEnvironments := breakItem.Details["deployed_environments"]
	s.False(hasDeployedEnvironments)
	s.Empty(breakItem.Details)
}
