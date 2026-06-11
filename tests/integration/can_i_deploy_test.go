package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
)

type checkedResource struct {
	Direction  string `json:"direction"`
	Kind       string `json:"kind"`
	Provider   string `json:"consumed_provider"`
	Endpoint   string `json:"endpoint"`
	Method     string `json:"method"`
	StatusCode string `json:"response_status_code"`
	Version    string `json:"version"`
}

type breakItem struct {
	CheckedResource     checkedResource   `json:"checked_resource"`
	CounterpartResource *checkedResource  `json:"counterpart_resource,omitempty"`
	Reason              string            `json:"reason"`
	Details             map[string]string `json:"details,omitempty"`
}

type canIDeployResponse struct {
	Deployable bool                   `json:"deployable"`
	Breaks     map[string][]breakItem `json:"breaks"`
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
	s.JSONEq(`{"message":"Contract checked successfully","deployable":true}`, body)

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

	frontBreaks := got.Breaks["front"]
	s.Require().Len(frontBreaks, 2)

	byReason := map[string]breakItem{}
	for _, b := range frontBreaks {
		s.Equal("consumes", b.CheckedResource.Direction)
		s.Equal("rest_response", b.CheckedResource.Kind)
		s.Equal("api", b.CheckedResource.Provider)
		s.Equal("/things", b.CheckedResource.Endpoint)
		s.Equal("get", b.CheckedResource.Method)
		s.Equal("200", b.CheckedResource.StatusCode)

		s.Require().NotNil(b.CounterpartResource)
		s.Equal("provides", b.CounterpartResource.Direction)

		byReason[b.Reason] = b
	}

	typeMismatch, ok := byReason["type_mismatch"]
	s.Require().True(ok)
	s.Equal(map[string]string{
		"property":                  "$.id",
		"checked_property_type":     "integer",
		"counterpart_property_type": "string",
	}, typeMismatch.Details)

	missing, ok := byReason["missing_in_provider"]
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

	frontBreaks := got.Breaks["front"]
	s.Require().Len(frontBreaks, 1)

	b := frontBreaks[0]
	s.Equal("provides", b.CheckedResource.Direction)
	s.Equal("/things", b.CheckedResource.Endpoint)
	s.Equal("get", b.CheckedResource.Method)
	s.Equal("200", b.CheckedResource.StatusCode)

	s.Require().NotNil(b.CounterpartResource)
	s.Equal("consumes", b.CounterpartResource.Direction)
	s.Equal("api", b.CounterpartResource.Provider)

	s.Equal("type_mismatch", b.Reason)
	s.Equal(map[string]string{
		"property":                  "$.id",
		"checked_property_type":     "string",
		"counterpart_property_type": "integer",
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

	providers := make([]string, 0, len(got.Breaks["app"]))
	for _, b := range got.Breaks["app"] {
		s.Equal("provider_resource_not_found", b.Reason)
		s.Equal("consumes", b.CheckedResource.Direction)
		s.Nil(b.CounterpartResource)
		s.Empty(b.Details)
		providers = append(providers, b.CheckedResource.Provider)
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

	s.Require().Len(got.Breaks["app"], 1)
	b := got.Breaks["app"][0]
	s.Equal("catalog", b.CheckedResource.Provider)
	s.Equal("type_mismatch", b.Reason)
	s.Equal("$.id", b.Details["property"])

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
	s.Require().Len(got.Breaks["front"], 1)

	b := got.Breaks["front"][0]
	s.Equal("consumes", b.CheckedResource.Direction)
	s.Equal("api", b.CheckedResource.Provider)
	s.Equal("provider_resource_not_deployed_in_environment", b.Reason)
	s.Equal("staging", b.Details["deployed_environments"])
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
		s.Require().JSONEqf(`{"message":"Contract checked successfully","deployable":true}`, body,
			"can-i-deploy %s %s", participant, version)
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

	var got canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &got))
	s.False(got.Deployable)
	s.NotContains(body, `"consumed_provider":""`)
	s.NotContains(body, `"response_status_code":""`)
	s.NotContains(body, `"version":""`)
	s.Contains(body, `"consumed_provider":null`)
	s.Contains(body, `"response_status_code":null`)
	s.Contains(body, `"version":null`)

	s.Require().Len(got.Breaks, 2, "breaks must surface both sides: consumer (pets) and provider (app)")

	petsBreaks := got.Breaks["pets"]
	s.Require().Len(petsBreaks, 1)

	consumerSide := petsBreaks[0]
	s.Equal("consumes", consumerSide.CheckedResource.Direction)
	s.Equal("rest_response", consumerSide.CheckedResource.Kind)
	s.Equal("users", consumerSide.CheckedResource.Provider)
	s.Equal("/users/*", consumerSide.CheckedResource.Endpoint)
	s.Equal("get", consumerSide.CheckedResource.Method)
	s.Equal("200", consumerSide.CheckedResource.StatusCode)
	s.Require().NotNil(consumerSide.CounterpartResource)
	s.Equal("provides", consumerSide.CounterpartResource.Direction)
	s.Equal("type_mismatch", consumerSide.Reason)
	s.Equal(map[string]string{
		"property":                  "$.userId",
		"checked_property_type":     "string",
		"counterpart_property_type": "integer",
	}, consumerSide.Details)

	appBreaks := got.Breaks["app"]
	s.Require().Len(appBreaks, 2)

	byReason := map[string]breakItem{}
	for _, b := range appBreaks {
		s.Equal("provides", b.CheckedResource.Direction)
		s.Require().NotNil(b.CounterpartResource)
		s.Equal("consumes", b.CounterpartResource.Direction)
		s.Equal("pets", b.CounterpartResource.Provider)
		byReason[b.Reason] = b
	}

	missingInConsumer, ok := byReason["missing_in_consumer"]
	s.Require().True(ok)
	s.Equal("rest_request", missingInConsumer.CheckedResource.Kind)
	s.Equal("/pets", missingInConsumer.CheckedResource.Endpoint)
	s.Equal("post", missingInConsumer.CheckedResource.Method)
	s.Equal(map[string]string{"property": "$.breed"}, missingInConsumer.Details)

	missingInProvider, ok := byReason["missing_in_provider"]
	s.Require().True(ok)
	s.Equal("rest_response", missingInProvider.CheckedResource.Kind)
	s.Equal("/pets/*", missingInProvider.CheckedResource.Endpoint)
	s.Equal("get", missingInProvider.CheckedResource.Method)
	s.Equal("200", missingInProvider.CheckedResource.StatusCode)
	s.Equal(map[string]string{"property": "$.name"}, missingInProvider.Details)

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
	s.Require().Len(got.Breaks["front"], 1)

	b := got.Breaks["front"][0]
	s.Equal("provider_resource_not_deployed_in_environment", b.Reason)
	s.Empty(b.Details)
}
