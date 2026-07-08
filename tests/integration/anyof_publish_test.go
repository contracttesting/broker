package integration_test

import (
	"context"
	"net/http"

	"github.com/contracttesting/broker/internal/repository"
)

const anyOfProviderContract = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": {
            "200": "Pets"
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

const anyOfSingleVariantContract = `{
  "provides": {
    "rest": {
      "/pets": {
        "get": {
          "responses": {
            "200": "Pets"
          }
        }
      }
    }
  },
  "schemas": {
    "Pets": {
      "anyOf": [ { "type": "string" } ]
    }
  }
}`

func (s *IntegrationSuite) TestPublishContract_AnyOf_PersistsAndReloadsFlatVariantRows() {
	status, _ := s.post("/api/participants", `{"participant":"pets-service"}`)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", `{"participant":"pets-service","version":"1","contract":`+anyOfProviderContract+`}`)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	wantTypes := map[string]string{
		"$":          "array",
		"$[]":        "anyOf",
		"$[]#0":      "object",
		"$[]#0.uuid": "string",
		"$[]#0.name": "string",
		"$[]#0.age":  "integer",
		"$[]#1":      "object",
		"$[]#1.uuid": "string",
		"$[]#1.name": "string",
	}

	rows, err := s.Pool.Query(context.Background(),
		`SELECT p.path, pv.type FROM properties p JOIN property_versions pv ON pv.property_id = p.id`)
	s.Require().NoError(err)
	defer rows.Close()

	storedTypes := map[string]string{}
	for rows.Next() {
		var path, propertyType string
		s.Require().NoError(rows.Scan(&path, &propertyType))
		storedTypes[path] = propertyType
	}
	s.Equal(wantTypes, storedTypes)

	contract, found := repository.NewContractRepository(s.Pool).GetLatestContractByName(context.Background(), "pets-service")
	s.Require().True(found)
	s.Require().Len(contract.Resources, 1)

	for _, resource := range contract.Resources {
		reloadedTypes := map[string]string{}
		for path, property := range resource.Properties {
			reloadedTypes[path] = property.Type
		}
		s.Equal(wantTypes, reloadedTypes)
	}
}

func (s *IntegrationSuite) TestPublishContract_MalformedAnyOf_RejectedNothingStored() {
	status, _ := s.post("/api/participants", `{"participant":"pets-service"}`)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", `{"participant":"pets-service","version":"1","contract":`+anyOfSingleVariantContract+`}`)
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"invalid schema \"Pets\": anyOf with a single variant is just that schema"}`, body)

	s.Equal(0, s.countRows("contracts"))
}
