package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
)

type resourceJSON struct {
	Direction          string  `json:"direction"`
	Interaction        string  `json:"interaction"`
	ConsumedProvider   *string `json:"consumedProvider"`
	Endpoint           string  `json:"endpoint"`
	Method             string  `json:"method"`
	ResponseStatusCode *string `json:"responseStatusCode"`
	Version            *string `json:"version"`
}

type breakJSON struct {
	CheckedResource     resourceJSON      `json:"checkedResource"`
	CounterpartResource *resourceJSON     `json:"counterpartResource"`
	Reason              string            `json:"reason"`
	Details             map[string]string `json:"details"`
}

type counterpartJSON struct {
	ParticipantName    string  `json:"participantName"`
	ParticipantVersion *string `json:"participantVersion"`
}

type resultJSON struct {
	Deployable              bool            `json:"deployable"`
	IncompatibleCounterpart counterpartJSON `json:"incompatibleCounterpart"`
	Breaks                  []breakJSON     `json:"breaks"`
}

type canIDeployResponse struct {
	Message     string                `json:"message"`
	Participant string                `json:"participant"`
	Version     string                `json:"version"`
	Environment string                `json:"environment"`
	Deployable  bool                  `json:"deployable"`
	Results     map[string]resultJSON `json:"results"`
}

// breaksByReason indexes a result's breaks by their reason (the scenarios below have at
// most one break per reason within a single counterpart result).
func breaksByReason(breaks []breakJSON) map[string]breakJSON {
	out := make(map[string]breakJSON, len(breaks))
	for _, b := range breaks {
		out[b.Reason] = b
	}
	return out
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

	s.Require().Len(v1Got.Results, 1)
	v1Api := v1Got.Results["api"]
	s.True(v1Api.Deployable)
	s.Equal("api", v1Api.IncompatibleCounterpart.ParticipantName)
	s.Require().NotNil(v1Api.IncompatibleCounterpart.ParticipantVersion)
	s.Equal("v1", *v1Api.IncompatibleCounterpart.ParticipantVersion)
	// compatible counterparts render "breaks":[], never null
	s.Require().NotNil(v1Api.Breaks)
	s.Empty(v1Api.Breaks)

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

	s.Require().Len(got.Results, 1)
	api := got.Results["api"]
	s.False(api.Deployable)
	s.Equal("api", api.IncompatibleCounterpart.ParticipantName)
	s.Require().NotNil(api.IncompatibleCounterpart.ParticipantVersion)
	s.Equal("v1", *api.IncompatibleCounterpart.ParticipantVersion)
	s.Require().Len(api.Breaks, 2)
	for _, b := range api.Breaks {
		s.Equal("consumes", b.CheckedResource.Direction)
		s.Equal("rest_response", b.CheckedResource.Interaction)
		s.Equal("get", b.CheckedResource.Method)
		s.Equal("/things", b.CheckedResource.Endpoint)
		s.Require().NotNil(b.CheckedResource.ResponseStatusCode)
		s.Equal("200", *b.CheckedResource.ResponseStatusCode)
		s.Require().NotNil(b.CounterpartResource)
		s.Equal("provides", b.CounterpartResource.Direction)
		s.Require().NotNil(b.CounterpartResource.Version)
		s.Equal("v1", *b.CounterpartResource.Version)
	}

	byReason := breaksByReason(api.Breaks)

	typeMismatch, ok := byReason["property_type_mismatch"]
	s.Require().True(ok)
	s.Equal(map[string]string{
		"property":                "$.id",
		"checkedPropertyType":     "integer",
		"counterpartPropertyType": "string",
	}, typeMismatch.Details)

	missing, ok := byReason["property_missing_in_provider"]
	s.Require().True(ok)
	s.Equal(map[string]string{"property": "$.name"}, missing.Details)

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

	s.Require().Len(got.Results, 1)
	front := got.Results["front"]
	s.False(front.Deployable)
	s.Equal("front", front.IncompatibleCounterpart.ParticipantName)
	s.Require().NotNil(front.IncompatibleCounterpart.ParticipantVersion)
	s.Equal("v1", *front.IncompatibleCounterpart.ParticipantVersion)
	s.Require().Len(front.Breaks, 1)

	b := front.Breaks[0]
	s.Equal("property_type_mismatch", b.Reason)
	s.Equal("provides", b.CheckedResource.Direction)
	s.Equal("rest_response", b.CheckedResource.Interaction)
	s.Equal("/things", b.CheckedResource.Endpoint)
	s.Equal("get", b.CheckedResource.Method)
	s.Require().NotNil(b.CheckedResource.ResponseStatusCode)
	s.Equal("200", *b.CheckedResource.ResponseStatusCode)
	s.Require().NotNil(b.CounterpartResource)
	s.Equal("consumes", b.CounterpartResource.Direction)
	s.Require().NotNil(b.CounterpartResource.Version)
	s.Equal("v1", *b.CounterpartResource.Version)
	s.Equal(map[string]string{
		"property":                "$.id",
		"checkedPropertyType":     "string",
		"counterpartPropertyType": "integer",
	}, b.Details)
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

	s.Require().Len(got.Results, 3)
	for _, provider := range []string{"users", "auth", "catalog"} {
		result, ok := got.Results[provider]
		s.Require().Truef(ok, "missing result for %s", provider)
		s.False(result.Deployable)
		s.Equal(provider, result.IncompatibleCounterpart.ParticipantName)
		s.Nil(result.IncompatibleCounterpart.ParticipantVersion)
		s.Require().Len(result.Breaks, 1)
		b := result.Breaks[0]
		s.Equal("provider_resource_not_found", b.Reason)
		s.Equal("consumes", b.CheckedResource.Direction)
		s.Require().NotNil(b.CheckedResource.ConsumedProvider)
		s.Equal(provider, *b.CheckedResource.ConsumedProvider)
		s.Nil(b.CounterpartResource)
		s.Empty(b.Details)
	}

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

	s.Require().Len(got.Results, 3)
	for _, provider := range []string{"users", "auth"} {
		result := got.Results[provider]
		s.Truef(result.Deployable, "expected %s to be deployable", provider)
		s.Equal(provider, result.IncompatibleCounterpart.ParticipantName)
		s.Require().NotNil(result.IncompatibleCounterpart.ParticipantVersion)
		s.Equal("v1", *result.IncompatibleCounterpart.ParticipantVersion)
		s.Require().NotNilf(result.Breaks, "expected %s breaks to render as []", provider)
		s.Empty(result.Breaks)
	}

	catalog := got.Results["catalog"]
	s.False(catalog.Deployable)
	s.Equal("catalog", catalog.IncompatibleCounterpart.ParticipantName)
	s.Require().NotNil(catalog.IncompatibleCounterpart.ParticipantVersion)
	s.Equal("v1", *catalog.IncompatibleCounterpart.ParticipantVersion)
	s.Require().Len(catalog.Breaks, 1)
	b := catalog.Breaks[0]
	s.Equal("property_type_mismatch", b.Reason)
	s.Equal(map[string]string{
		"property":                "$.id",
		"checkedPropertyType":     "integer",
		"counterpartPropertyType": "string",
	}, b.Details)

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

	s.Require().Len(got.Results, 1)
	api := got.Results["api"]
	s.False(api.Deployable)
	s.Equal("api", api.IncompatibleCounterpart.ParticipantName)
	s.Nil(api.IncompatibleCounterpart.ParticipantVersion)
	s.Require().Len(api.Breaks, 1)

	b := api.Breaks[0]
	s.Equal("provider_resource_not_deployed_in_environment", b.Reason)
	s.Nil(b.CounterpartResource)
	s.Equal(map[string]string{"deployedEnvironments": "staging"}, b.Details)

	// the provider participant is known even though it is not deployed in the target
	// environment: the matrix row keeps its identity with a NULL counterpart version
	var counterpartName string
	var counterpartVersion *string
	s.Require().NoError(s.Pool.QueryRow(context.Background(),
		`SELECT p.name, cm.counterpart_version
		   FROM compatibility_matrix cm
		   JOIN participants p ON p.id = cm.counterpart_participant_id`).
		Scan(&counterpartName, &counterpartVersion))
	s.Equal("api", counterpartName)
	s.Nil(counterpartVersion)
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
		for counterpart, result := range deployGot.Results {
			s.Truef(result.Deployable, "can-i-deploy %s %s vs %s", participant, version, counterpart)
			s.Emptyf(result.Breaks, "can-i-deploy %s %s vs %s", participant, version, counterpart)
		}
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

	// empty versions render as JSON null, never empty strings
	s.NotContains(body, `"version":""`)
	s.Contains(body, `"version":null`)

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)
	s.Equal("pets", got.Participant)
	s.Equal("v2", got.Version)
	s.Equal("production", got.Environment)

	s.Require().Len(got.Results, 2)

	// pets v2 acts as a provider (checked against the deployed app consumer) and
	// as a consumer of users (checked against the deployed users provider).
	appResult, ok := got.Results["app"]
	s.Require().True(ok)
	s.False(appResult.Deployable)
	s.Equal("app", appResult.IncompatibleCounterpart.ParticipantName)
	s.Require().NotNil(appResult.IncompatibleCounterpart.ParticipantVersion)
	s.Equal("v1", *appResult.IncompatibleCounterpart.ParticipantVersion)
	s.Require().Len(appResult.Breaks, 2)

	appByReason := breaksByReason(appResult.Breaks)

	missingInConsumer, ok := appByReason["property_missing_in_consumer"]
	s.Require().True(ok)
	s.Equal("provides", missingInConsumer.CheckedResource.Direction)
	s.Equal("rest_request", missingInConsumer.CheckedResource.Interaction)
	s.Equal("post", missingInConsumer.CheckedResource.Method)
	s.Equal("/pets", missingInConsumer.CheckedResource.Endpoint)
	s.Nil(missingInConsumer.CheckedResource.ResponseStatusCode)
	s.Equal(map[string]string{"property": "$.breed"}, missingInConsumer.Details)
	s.Require().NotNil(missingInConsumer.CounterpartResource)
	s.Equal("consumes", missingInConsumer.CounterpartResource.Direction)
	s.Require().NotNil(missingInConsumer.CounterpartResource.Version)
	s.Equal("v1", *missingInConsumer.CounterpartResource.Version)

	missingInProvider, ok := appByReason["property_missing_in_provider"]
	s.Require().True(ok)
	s.Equal("provides", missingInProvider.CheckedResource.Direction)
	s.Equal("rest_response", missingInProvider.CheckedResource.Interaction)
	s.Equal("get", missingInProvider.CheckedResource.Method)
	s.Equal("/pets/*", missingInProvider.CheckedResource.Endpoint)
	s.Require().NotNil(missingInProvider.CheckedResource.ResponseStatusCode)
	s.Equal("200", *missingInProvider.CheckedResource.ResponseStatusCode)
	s.Equal(map[string]string{"property": "$.name"}, missingInProvider.Details)

	usersResult, ok := got.Results["users"]
	s.Require().True(ok)
	s.False(usersResult.Deployable)
	s.Equal("users", usersResult.IncompatibleCounterpart.ParticipantName)
	s.Require().NotNil(usersResult.IncompatibleCounterpart.ParticipantVersion)
	s.Equal("v1", *usersResult.IncompatibleCounterpart.ParticipantVersion)
	s.Require().Len(usersResult.Breaks, 1)

	consumerBreak := usersResult.Breaks[0]
	s.Equal("property_type_mismatch", consumerBreak.Reason)
	s.Equal("consumes", consumerBreak.CheckedResource.Direction)
	s.Equal("rest_response", consumerBreak.CheckedResource.Interaction)
	s.Equal("get", consumerBreak.CheckedResource.Method)
	s.Equal("/users/*", consumerBreak.CheckedResource.Endpoint)
	s.Equal(map[string]string{
		"property":                "$.userId",
		"checkedPropertyType":     "string",
		"counterpartPropertyType": "integer",
	}, consumerBreak.Details)

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

	s.Require().Len(got.Results, 1)
	api := got.Results["api"]
	s.False(api.Deployable)
	s.Equal("api", api.IncompatibleCounterpart.ParticipantName)
	s.Nil(api.IncompatibleCounterpart.ParticipantVersion)
	s.Require().Len(api.Breaks, 1)

	b := api.Breaks[0]
	s.Equal("provider_resource_not_deployed_in_environment", b.Reason)
	s.Nil(b.CounterpartResource)
	s.Empty(b.Details)
}
