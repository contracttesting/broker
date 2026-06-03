package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
)

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

	type brokenResource struct {
		Direction  string `json:"direction"`
		Kind       string `json:"kind"`
		Provider   string `json:"consumed_provider"`
		Endpoint   string `json:"endpoint"`
		Method     string `json:"method"`
		StatusCode string `json:"response_status_code"`
	}

	type breakItem struct {
		LeftResource  brokenResource `json:"left_resource"`
		RightResource brokenResource `json:"right_resource"`
		Reason        string         `json:"reason"`
		Property      string         `json:"property"`
		HumanReadable string         `json:"human_readable"`
		ConsumerName  string         `json:"consumer_name"`
		ProviderName  string         `json:"provider_name"`
		ConsumerType  string         `json:"consumer_type"`
		ProviderType  string         `json:"provider_type"`
	}

	var got struct {
		Deployable bool                   `json:"deployable"`
		Breaks     map[string][]breakItem `json:"breaks"`
	}

	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)

	consumer := brokenResource{
		Direction:  "consumes",
		Kind:       "rest_response",
		Provider:   "api",
		Endpoint:   "/things",
		Method:     "get",
		StatusCode: "200",
	}

	provider := brokenResource{
		Direction:  "provides",
		Kind:       "rest_response",
		Provider:   "",
		Endpoint:   "/things",
		Method:     "get",
		StatusCode: "200",
	}

	// The break order is not deterministic (the checker ranges over a property map).
	s.ElementsMatch([]breakItem{
		{
			LeftResource:  consumer,
			RightResource: provider,
			Reason:        "type_mismatch",
			Property:      "root.id",
			HumanReadable: "Property root.id type mismatch on GET /things (response 200), provider api expects string but consumer front expects integer",
			ConsumerName:  "front",
			ProviderName:  "api",
			ConsumerType:  "integer",
			ProviderType:  "string",
		},
		{
			LeftResource:  consumer,
			RightResource: provider,
			Reason:        "missing_in_provider",
			Property:      "root.name",
			HumanReadable: "Property root.name is missing in provider api on GET /things (response 200)",
			ConsumerName:  "front",
			ProviderName:  "api",
		},
	}, got.Breaks["front"])

	// The incompatible decision is persisted too, as a non-deployable row.
	s.Equal(2, s.countRows("compatibility_matrix"))
	var v2Deployable bool
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT deployable FROM compatibility_matrix WHERE version = 'v2'`).Scan(&v2Deployable))
	s.False(v2Deployable)
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
	var got struct {
		Deployable bool `json:"deployable"`
		Breaks     map[string][]struct {
			LeftResource struct {
				Provider string `json:"consumed_provider"`
			} `json:"left_resource"`
			Reason string `json:"reason"`
		} `json:"breaks"`
	}
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)

	providers := make([]string, 0, len(got.Breaks["app"]))
	for _, b := range got.Breaks["app"] {
		s.Equal("provider_resource_not_found", b.Reason)
		providers = append(providers, b.LeftResource.Provider)
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
	var got struct {
		Deployable bool `json:"deployable"`
		Breaks     map[string][]struct {
			LeftResource struct {
				Provider string `json:"consumed_provider"`
			} `json:"left_resource"`
			Reason   string `json:"reason"`
			Property string `json:"property"`
		} `json:"breaks"`
	}
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)

	s.Require().Len(got.Breaks["app"], 1)
	s.Equal("catalog", got.Breaks["app"][0].LeftResource.Provider)
	s.Equal("type_mismatch", got.Breaks["app"][0].Reason)
	s.Equal("root.id", got.Breaks["app"][0].Property)

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

	var got struct {
		Deployable bool `json:"deployable"`
		Breaks     map[string][]struct {
			LeftResource struct {
				Provider string `json:"consumed_provider"`
			} `json:"left_resource"`
			Reason               string   `json:"reason"`
			HumanReadable        string   `json:"human_readable"`
			DeployedEnvironments []string `json:"deployed_environments"`
		} `json:"breaks"`
	}
	s.Require().NoError(json.Unmarshal([]byte(body), &got))

	s.False(got.Deployable)
	s.Require().Len(got.Breaks["front"], 1)

	breakItem := got.Breaks["front"][0]
	s.Equal("api", breakItem.LeftResource.Provider)
	s.Equal("provider_resource_not_deployed_in_environment", breakItem.Reason)
	s.Equal([]string{"staging"}, breakItem.DeployedEnvironments)
	s.Contains(breakItem.HumanReadable, "staging")
	s.Contains(breakItem.HumanReadable, "isn't deployed")
}

// The provider exists but has never been deployed anywhere. The break still fires
// (not deployable) and reports an empty deployed-environments list.
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

	var got struct {
		Deployable bool `json:"deployable"`
		Breaks     map[string][]struct {
			Reason               string   `json:"reason"`
			HumanReadable        string   `json:"human_readable"`
			DeployedEnvironments []string `json:"deployed_environments"`
		} `json:"breaks"`
	}
	s.Require().NoError(json.Unmarshal([]byte(body), &got))

	s.False(got.Deployable)
	s.Require().Len(got.Breaks["front"], 1)

	breakItem := got.Breaks["front"][0]
	s.Equal("provider_resource_not_deployed_in_environment", breakItem.Reason)
	s.Empty(breakItem.DeployedEnvironments)
	s.Contains(breakItem.HumanReadable, "any environment yet")
}
