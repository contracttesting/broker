package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
)

type partyJSON struct {
	Participant string  `json:"participant"`
	Version     *string `json:"version"`
	Role        string  `json:"role"`
	Checked     bool    `json:"checked"`
}

type breakJSON struct {
	Interaction string            `json:"interaction"`
	Method      string            `json:"method"`
	Endpoint    string            `json:"endpoint"`
	StatusCode  *string           `json:"statusCode"`
	Property    *string           `json:"property"`
	Reason      string            `json:"reason"`
	Details     map[string]string `json:"details"`
}

type incompatibilityJSON struct {
	Consumer      partyJSON   `json:"consumer"`
	Provider      partyJSON   `json:"provider"`
	BreakingCount int         `json:"breakingCount"`
	Breaks        []breakJSON `json:"breaks"`
}

type canIDeployResponse struct {
	Message           string                `json:"message"`
	Participant       string                `json:"participant"`
	Version           string                `json:"version"`
	Environment       string                `json:"environment"`
	Deployable        bool                  `json:"deployable"`
	Incompatibilities []incompatibilityJSON `json:"incompatibilities"`
}

// breaksByReason indexes an edge's breaks by their reason (the scenarios below have at
// most one break per reason within a single incompatibility).
func breaksByReason(breaks []breakJSON) map[string]breakJSON {
	out := make(map[string]breakJSON, len(breaks))
	for _, b := range breaks {
		out[b.Reason] = b
	}
	return out
}

// assertCheckedSide verifies the directed-edge flags: both parties carry their role, and
// only the subject side (the participant being deployed) is flagged checked.
func assertCheckedSide(s *IntegrationSuite, inc incompatibilityJSON, checkedRole string) {
	s.Equal("consumer", inc.Consumer.Role)
	s.Equal("provider", inc.Provider.Role)
	s.Equal(checkedRole == "consumer", inc.Consumer.Checked)
	s.Equal(checkedRole == "provider", inc.Provider.Checked)
}

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
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+apiV1ProviderContract+`}`)
	mustPost("/api/environments", `{"participant":"production"}`)
	mustPost("/api/deployments", `{"participant":"api","version":"v1","environment":"production"}`)
	mustPost("/api/participants", `{"participant":"front"}`)
	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+frontV1ConsumerContract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"front","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var v1Got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &v1Got))
	s.True(v1Got.Deployable)
	s.Equal("front", v1Got.Participant)
	s.Equal("v1", v1Got.Version)
	s.Equal("production", v1Got.Environment)
	s.Empty(v1Got.Incompatibilities)

	s.Equal(1, s.countRows("compatibility_matrix"))
	var v1Deployable bool
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT deployable FROM compatibility_matrix WHERE version = 'v1'`).Scan(&v1Deployable))
	s.True(v1Deployable)

	mustPost("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)
	mustPost("/api/contracts", `{"participant":"front","version":"v2","contract":`+frontV2ConsumerContract+`}`)

	status, body = s.post("/api/can-i-deploy", `{"participant":"front","version":"v2","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)
	s.Equal("front", got.Participant)
	s.Equal("v2", got.Version)
	s.Equal("production", got.Environment)

	s.Require().Len(got.Incompatibilities, 1)
	inc := got.Incompatibilities[0]
	assertCheckedSide(s, inc, "consumer")
	s.Equal("front", inc.Consumer.Participant)
	s.Require().NotNil(inc.Consumer.Version)
	s.Equal("v2", *inc.Consumer.Version)
	s.Equal("api", inc.Provider.Participant)
	s.Require().NotNil(inc.Provider.Version)
	s.Equal("v1", *inc.Provider.Version)
	s.Equal(2, inc.BreakingCount)
	s.Require().Len(inc.Breaks, 2)
	for _, b := range inc.Breaks {
		s.Equal("response", b.Interaction)
		s.Equal("get", b.Method)
		s.Equal("/things", b.Endpoint)
		s.Require().NotNil(b.StatusCode)
		s.Equal("200", *b.StatusCode)
	}

	byReason := breaksByReason(inc.Breaks)

	typeMismatch, ok := byReason["property_type_mismatch"]
	s.Require().True(ok)
	s.Require().NotNil(typeMismatch.Property)
	s.Equal("$.id", *typeMismatch.Property)
	s.Equal("integer", typeMismatch.Details["checkedPropertyType"])
	s.Equal("string", typeMismatch.Details["counterpartPropertyType"])

	missing, ok := byReason["property_missing_in_provider"]
	s.Require().True(ok)
	s.Require().NotNil(missing.Property)
	s.Equal("$.name", *missing.Property)
	s.Empty(missing.Details)

	s.Equal(2, s.countRows("compatibility_matrix"))
	var v2Deployable bool
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT deployable FROM compatibility_matrix WHERE version = 'v2'`).Scan(&v2Deployable))
	s.False(v2Deployable)
}

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

func (s *IntegrationSuite) TestCanIDeploy_ProviderCheckedAgainstDeployedConsumer() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	mustPost("/api/participants", `{"participant":"api"}`)
	mustPost("/api/participants", `{"participant":"front"}`)
	mustPost("/api/environments", `{"participant":"production"}`)
	mustPost("/api/contracts", `{"participant":"front","version":"v1","contract":`+providerCheckedConsumerContract+`}`)
	mustPost("/api/deployments", `{"participant":"front","version":"v1","environment":"production"}`)
	mustPost("/api/contracts", `{"participant":"api","version":"v1","contract":`+apiV1ProviderContract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"api","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)
	s.Equal("api", got.Participant)
	s.Equal("v1", got.Version)

	s.Require().Len(got.Incompatibilities, 1)
	inc := got.Incompatibilities[0]
	assertCheckedSide(s, inc, "provider")
	s.Equal("api", inc.Provider.Participant)
	s.Require().NotNil(inc.Provider.Version)
	s.Equal("v1", *inc.Provider.Version)
	s.Equal("front", inc.Consumer.Participant)
	s.Require().NotNil(inc.Consumer.Version)
	s.Equal("v1", *inc.Consumer.Version)
	s.Require().Len(inc.Breaks, 1)

	b := inc.Breaks[0]
	s.Equal("response", b.Interaction)
	s.Equal("/things", b.Endpoint)
	s.Equal("get", b.Method)
	s.Require().NotNil(b.StatusCode)
	s.Equal("200", *b.StatusCode)
	s.Equal("property_type_mismatch", b.Reason)
	s.Require().NotNil(b.Property)
	s.Equal("$.id", *b.Property)
	s.Equal("string", b.Details["checkedPropertyType"])
	s.Equal("integer", b.Details["counterpartPropertyType"])
}

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

	status, _ = s.post("/api/contracts",
		`{"participant":"app","version":"v1","contract":`+appV1ThreeDependenciesContract+`}`)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/can-i-deploy",
		`{"participant":"app","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)
	s.Equal("app", got.Participant)
	s.Equal("v1", got.Version)

	s.Require().Len(got.Incompatibilities, 3)
	providers := make([]string, 0, len(got.Incompatibilities))
	for _, inc := range got.Incompatibilities {
		assertCheckedSide(s, inc, "consumer")
		s.Equal("app", inc.Consumer.Participant)
		s.Require().NotNil(inc.Consumer.Version)
		s.Equal("v1", *inc.Consumer.Version)
		s.Nil(inc.Provider.Version)
		s.Require().Len(inc.Breaks, 1)
		b := inc.Breaks[0]
		s.Equal("provider_resource_not_found", b.Reason)
		s.Nil(b.Property)
		s.Empty(b.Details)
		providers = append(providers, inc.Provider.Participant)
	}
	s.ElementsMatch([]string{"users", "auth", "catalog"}, providers)

	s.Equal(3, s.countRows("compatibility_matrix"))
	var nonDeployable int
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM compatibility_matrix WHERE version = 'v1' AND NOT deployable`).
		Scan(&nonDeployable))
	s.Equal(3, nonDeployable)
}

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

	mustPost("/api/contracts", `{"participant":"users","version":"v1","contract":`+usersV1ProviderContract+`}`)
	mustPost("/api/deployments", `{"participant":"users","version":"v1","environment":"production"}`)
	mustPost("/api/contracts", `{"participant":"auth","version":"v1","contract":`+authV1ProviderContract+`}`)
	mustPost("/api/deployments", `{"participant":"auth","version":"v1","environment":"production"}`)
	mustPost("/api/contracts", `{"participant":"catalog","version":"v1","contract":`+catalogV1ProviderContract+`}`)
	mustPost("/api/deployments", `{"participant":"catalog","version":"v1","environment":"production"}`)

	mustPost("/api/contracts", `{"participant":"app","version":"v1","contract":`+appV1MixedDependenciesContract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"app","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)
	s.Equal("app", got.Participant)
	s.Equal("v1", got.Version)

	s.Require().Len(got.Incompatibilities, 1)
	inc := got.Incompatibilities[0]
	assertCheckedSide(s, inc, "consumer")
	s.Equal("app", inc.Consumer.Participant)
	s.Equal("catalog", inc.Provider.Participant)
	s.Require().NotNil(inc.Provider.Version)
	s.Equal("v1", *inc.Provider.Version)
	s.Require().Len(inc.Breaks, 1)
	b := inc.Breaks[0]
	s.Equal("property_type_mismatch", b.Reason)
	s.Require().NotNil(b.Property)
	s.Equal("$.id", *b.Property)
	s.Equal("integer", b.Details["checkedPropertyType"])
	s.Equal("string", b.Details["counterpartPropertyType"])

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
	s.Equal(map[string]string{"users": "v1", "auth": "v1", "catalog": "v1"}, versionByProvider)
}

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
	s.Equal("front", got.Participant)
	s.Equal("v1", got.Version)

	s.Require().Len(got.Incompatibilities, 1)
	inc := got.Incompatibilities[0]
	assertCheckedSide(s, inc, "consumer")
	s.Equal("front", inc.Consumer.Participant)
	s.Equal("api", inc.Provider.Participant)
	s.Nil(inc.Provider.Version)
	s.Require().Len(inc.Breaks, 1)

	b := inc.Breaks[0]
	s.Equal("provider_resource_not_deployed_in_environment", b.Reason)
	s.Nil(b.Property)
	s.Equal("staging", b.Details["deployedEnvironments"])
}

const dualRoleUsersV1Contract = `
{
  "provides": {
    "rest": {
      "/users": {
        "post": {
          "request": "CreateUserRequest",
          "responses": { "200": "CreateUserResponse" }
        }
      },
      "/users/*": {
        "get": { "responses": { "200": "User" } }
      }
    }
  },
  "schemas": {
    "CreateUserRequest": {
      "type": "object",
      "properties": {
        "email":    { "type": "string" },
        "password": { "type": "string" }
      }
    },
    "CreateUserResponse": {
      "type": "object",
      "properties": {
        "userId": { "type": "integer" }
      }
    },
    "User": {
      "type": "object",
      "properties": {
        "userId": { "type": "integer" },
        "status": { "type": "string" }
      }
    }
  }
}`

const dualRolePetsV1Contract = `
{
  "consumes": {
    "users": {
      "rest": {
        "/users/*": {
          "get": { "responses": { "200": "User" } }
        }
      }
    }
  },
  "provides": {
    "rest": {
      "/pets": {
        "post": {
          "request": "CreatePetRequest",
          "responses": { "200": "Pet" }
        }
      },
      "/pets/*": {
        "get": { "responses": { "200": "Pet" } }
      }
    }
  },
  "schemas": {
    "User": {
      "type": "object",
      "properties": {
        "userId": { "type": "integer" }
      }
    },
    "CreatePetRequest": {
      "type": "object",
      "properties": {
        "name":   { "type": "string" },
        "userId": { "type": "integer" }
      }
    },
    "Pet": {
      "type": "object",
      "properties": {
        "petId":  { "type": "integer" },
        "userId": { "type": "integer" },
        "name":   { "type": "string" }
      }
    }
  }
}`

const dualRolePetsV2Contract = `
{
  "consumes": {
    "users": {
      "rest": {
        "/users/*": {
          "get": { "responses": { "200": "User" } }
        }
      }
    }
  },
  "provides": {
    "rest": {
      "/pets": {
        "post": {
          "request": "CreatePetRequest",
          "responses": { "200": "Pet" }
        }
      },
      "/pets/*": {
        "get": { "responses": { "200": "PetSummary" } }
      }
    }
  },
  "schemas": {
    "User": {
      "type": "object",
      "properties": {
        "userId": { "type": "string" }
      }
    },
    "CreatePetRequest": {
      "type": "object",
      "properties": {
        "name":   { "type": "string" },
        "userId": { "type": "integer" },
        "breed":  { "type": "string" }
      }
    },
    "Pet": {
      "type": "object",
      "properties": {
        "petId":  { "type": "integer" },
        "userId": { "type": "integer" },
        "name":   { "type": "string" }
      }
    },
    "PetSummary": {
      "type": "object",
      "properties": {
        "petId":  { "type": "integer" },
        "userId": { "type": "integer" }
      }
    }
  }
}`

const dualRoleAppV1Contract = `
{
  "consumes": {
    "users": {
      "rest": {
        "/users": {
          "post": {
            "request": "CreateUserRequest",
            "responses": { "200": "CreateUserResponse" }
          }
        },
        "/users/*": {
          "get": { "responses": { "200": "User" } }
        }
      }
    },
    "pets": {
      "rest": {
        "/pets": {
          "post": {
            "request": "CreatePetRequest",
            "responses": { "200": "Pet" }
          }
        },
        "/pets/*": {
          "get": { "responses": { "200": "Pet" } }
        }
      }
    }
  },
  "schemas": {
    "CreateUserRequest": {
      "type": "object",
      "properties": {
        "email":    { "type": "string" },
        "password": { "type": "string" }
      }
    },
    "CreateUserResponse": {
      "type": "object",
      "properties": {
        "userId": { "type": "integer" }
      }
    },
    "User": {
      "type": "object",
      "properties": {
        "userId": { "type": "integer" },
        "status": { "type": "string" }
      }
    },
    "CreatePetRequest": {
      "type": "object",
      "properties": {
        "name":   { "type": "string" },
        "userId": { "type": "integer" }
      }
    },
    "Pet": {
      "type": "object",
      "properties": {
        "petId":  { "type": "integer" },
        "userId": { "type": "integer" },
        "name":   { "type": "string" }
      }
    }
  }
}`

func (s *IntegrationSuite) TestCanIDeploy_ConsumerAndProviderSameContract() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	checkDeployableAndDeploy := func(participant, version string) {
		status, body := s.post("/api/can-i-deploy",
			`{"participant":"`+participant+`","version":"`+version+`","environment":"production"}`)
		s.Require().Equalf(http.StatusOK, status, "can-i-deploy %s %s", participant, version)
		var deployGot canIDeployResponse
		s.Require().NoErrorf(json.Unmarshal([]byte(body), &deployGot), "can-i-deploy %s %s", participant, version)
		s.Truef(deployGot.Deployable, "can-i-deploy %s %s", participant, version)
		s.Emptyf(deployGot.Incompatibilities, "can-i-deploy %s %s", participant, version)
		mustPost("/api/deployments",
			`{"participant":"`+participant+`","version":"`+version+`","environment":"production"}`)
	}

	for _, name := range []string{"users", "pets", "app"} {
		mustPost("/api/participants", `{"participant":"`+name+`"}`)
	}
	mustPost("/api/environments", `{"participant":"production"}`)

	mustPost("/api/contracts", `{"participant":"users","version":"v1","contract":`+dualRoleUsersV1Contract+`}`)
	checkDeployableAndDeploy("users", "v1")

	mustPost("/api/contracts", `{"participant":"pets","version":"v1","contract":`+dualRolePetsV1Contract+`}`)
	checkDeployableAndDeploy("pets", "v1")

	mustPost("/api/contracts", `{"participant":"app","version":"v1","contract":`+dualRoleAppV1Contract+`}`)
	checkDeployableAndDeploy("app", "v1")

	mustPost("/api/contracts", `{"participant":"pets","version":"v2","contract":`+dualRolePetsV2Contract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"pets","version":"v2","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	// versions and the request break's statusCode render as JSON null, never empty strings
	s.NotContains(body, `"statusCode":""`)
	s.NotContains(body, `"version":""`)
	s.Contains(body, `"statusCode":null`)

	// both parties always carry their role and an explicit checked flag:
	// the subject side is checked:true, the counterpart checked:false
	s.Contains(body, `"role":"provider"`)
	s.Contains(body, `"role":"consumer"`)
	s.Contains(body, `"checked":true`)
	s.Contains(body, `"checked":false`)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)
	s.Equal("pets", got.Participant)
	s.Equal("v2", got.Version)
	s.Equal("production", got.Environment)

	s.Require().Len(got.Incompatibilities, 2)

	// pets v2 acts as a provider (checked against the deployed app consumer) and
	// as a consumer of users (checked against the deployed users provider).
	appEdge := got.Incompatibilities[0]
	assertCheckedSide(s, appEdge, "provider")
	s.Equal("app", appEdge.Consumer.Participant)
	s.Require().NotNil(appEdge.Consumer.Version)
	s.Equal("v1", *appEdge.Consumer.Version)
	s.Equal("pets", appEdge.Provider.Participant)
	s.Require().NotNil(appEdge.Provider.Version)
	s.Equal("v2", *appEdge.Provider.Version)
	s.Equal(2, appEdge.BreakingCount)
	s.Require().Len(appEdge.Breaks, 2)

	appByReason := breaksByReason(appEdge.Breaks)

	missingInConsumer, ok := appByReason["property_missing_in_consumer"]
	s.Require().True(ok)
	s.Equal("request", missingInConsumer.Interaction)
	s.Equal("post", missingInConsumer.Method)
	s.Equal("/pets", missingInConsumer.Endpoint)
	s.Nil(missingInConsumer.StatusCode)
	s.Require().NotNil(missingInConsumer.Property)
	s.Equal("$.breed", *missingInConsumer.Property)

	missingInProvider, ok := appByReason["property_missing_in_provider"]
	s.Require().True(ok)
	s.Equal("response", missingInProvider.Interaction)
	s.Equal("get", missingInProvider.Method)
	s.Equal("/pets/*", missingInProvider.Endpoint)
	s.Require().NotNil(missingInProvider.StatusCode)
	s.Equal("200", *missingInProvider.StatusCode)
	s.Require().NotNil(missingInProvider.Property)
	s.Equal("$.name", *missingInProvider.Property)

	usersEdge := got.Incompatibilities[1]
	assertCheckedSide(s, usersEdge, "consumer")
	s.Equal("pets", usersEdge.Consumer.Participant)
	s.Require().NotNil(usersEdge.Consumer.Version)
	s.Equal("v2", *usersEdge.Consumer.Version)
	s.Equal("users", usersEdge.Provider.Participant)
	s.Require().NotNil(usersEdge.Provider.Version)
	s.Equal("v1", *usersEdge.Provider.Version)
	s.Require().Len(usersEdge.Breaks, 1)

	consumerBreak := usersEdge.Breaks[0]
	s.Equal("property_type_mismatch", consumerBreak.Reason)
	s.Equal("response", consumerBreak.Interaction)
	s.Equal("get", consumerBreak.Method)
	s.Equal("/users/*", consumerBreak.Endpoint)
	s.Require().NotNil(consumerBreak.Property)
	s.Equal("$.userId", *consumerBreak.Property)
	s.Equal("string", consumerBreak.Details["checkedPropertyType"])
	s.Equal("integer", consumerBreak.Details["counterpartPropertyType"])

	type matrixRow struct {
		Counterpart string
		Version     string
		Deployable  bool
	}

	rows, err := s.Pool.Query(context.Background(),
		`SELECT counterpart.name, cm.counterpart_version, cm.deployable
		   FROM compatibility_matrix cm
		   JOIN participants checked ON checked.id = cm.participant_id
		   JOIN participants counterpart ON counterpart.id = cm.counterpart_participant_id
		  WHERE checked.name = 'pets' AND cm.version = 'v2'`)
	s.Require().NoError(err)
	defer rows.Close()

	var matrixRows []matrixRow
	for rows.Next() {
		var row matrixRow
		s.Require().NoError(rows.Scan(&row.Counterpart, &row.Version, &row.Deployable))
		matrixRows = append(matrixRows, row)
	}
	s.Require().NoError(rows.Err())

	s.ElementsMatch([]matrixRow{
		{Counterpart: "users", Version: "v1", Deployable: false},
		{Counterpart: "app", Version: "v1", Deployable: false},
	}, matrixRows)
}

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
	s.Equal("front", got.Participant)
	s.Equal("v1", got.Version)

	s.Require().Len(got.Incompatibilities, 1)
	inc := got.Incompatibilities[0]
	assertCheckedSide(s, inc, "consumer")
	s.Equal("front", inc.Consumer.Participant)
	s.Equal("api", inc.Provider.Participant)
	s.Nil(inc.Provider.Version)
	s.Require().Len(inc.Breaks, 1)

	b := inc.Breaks[0]
	s.Equal("provider_resource_not_deployed_in_environment", b.Reason)
	s.Nil(b.Property)
	s.Empty(b.Details)
}
