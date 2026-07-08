package integration_test

import (
	"encoding/json"
	"net/http"
)

// JSON renditions of the contracts under examples/anyof: a provider whose
// /pets response items are anyOf [Pet, Dog], a consumer reading a subset of
// every variant, and a consumer matching no variant.
const anyOfPetsProviderContract = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": { "200": "Pets" }
        }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": {
        "uuid": { "type": "string" },
        "name": { "type": "string" },
        "age":  { "type": "integer" }
      }
    },
    "Dog": {
      "type": "object",
      "properties": {
        "uuid": { "type": "string" },
        "name": { "type": "string" }
      }
    },
    "Pets": {
      "type": "array",
      "items": {
        "anyOf": [ { "$ref": "Pet" }, { "$ref": "Dog" } ]
      }
    }
  }
}`

const anyOfCompatibleConsumerContract = `{
  "consumes": {
    "pets-service": {
      "rest": {
        "/pets": {
          "get": {
            "responses": { "200": "Pets" }
          }
        }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": {
        "uuid": { "type": "string" },
        "name": { "type": "string" }
      }
    },
    "Pets": {
      "type": "array",
      "items": { "$ref": "Pet" }
    }
  }
}`

const anyOfBreakingConsumerContract = `{
  "consumes": {
    "pets-service": {
      "rest": {
        "/pets": {
          "get": {
            "responses": { "200": "Pets" }
          }
        }
      }
    }
  },
  "schemas": {
    "Pet": {
      "type": "object",
      "properties": {
        "age": { "type": "string" }
      }
    },
    "Pets": {
      "type": "array",
      "items": { "$ref": "Pet" }
    }
  }
}`

func (s *IntegrationSuite) TestCanIDeploy_AnyOfUnion() {
	mustPost := func(path, body string) {
		status, _ := s.post(path, body)
		s.Require().Equalf(http.StatusOK, status, "POST %s", path)
	}

	mustPost("/api/participants", `{"participant":"pets-service"}`)
	mustPost("/api/contracts", `{"participant":"pets-service","version":"v1","contract":`+anyOfPetsProviderContract+`}`)
	mustPost("/api/environments", `{"participant":"production"}`)
	mustPost("/api/can-i-deploy", `{"participant":"pets-service","version":"v1","environment":"production"}`)
	mustPost("/api/deployments", `{"participant":"pets-service","version":"v1","environment":"production"}`)
	mustPost("/api/participants", `{"participant":"app"}`)
	mustPost("/api/contracts", `{"participant":"app","version":"v1","contract":`+anyOfCompatibleConsumerContract+`}`)

	status, body := s.post("/api/can-i-deploy", `{"participant":"app","version":"v1","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var compatible canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &compatible))
	s.True(compatible.Deployable)

	s.Require().Len(compatible.Results, 1)
	provider := compatible.Results["pets-service"]
	s.True(provider.Deployable)
	s.Require().NotNil(provider.Endpoints)
	s.Empty(provider.Endpoints)

	mustPost("/api/contracts", `{"participant":"app","version":"v2","contract":`+anyOfBreakingConsumerContract+`}`)

	status, body = s.post("/api/can-i-deploy", `{"participant":"app","version":"v2","environment":"production"}`)
	s.Equal(http.StatusOK, status)

	var breaking canIDeployResponse
	s.Require().NoError(json.Unmarshal([]byte(body), &breaking))
	s.False(breaking.Deployable)

	s.Require().Len(breaking.Results, 1)
	provider = breaking.Results["pets-service"]
	s.False(provider.Deployable)

	// the consumer accepts neither Pet nor Dog, so each provider variant
	// the response may carry is reported as unaccepted
	breaks := provider.Endpoints["/pets"]["get"]["200"]
	s.Require().Len(breaks, 2)

	properties := []string{}
	for _, contractBreak := range breaks {
		s.Equal("property_not_matching_any_variant", contractBreak.Reason)
		s.Equal("object", contractBreak.Details["consumerPropertyType"])
		s.Equal("object", contractBreak.Details["providerPropertyType"])
		properties = append(properties, contractBreak.Details["property"])
	}
	s.ElementsMatch([]string{"$[]#0", "$[]#1"}, properties)
}
