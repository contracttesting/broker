package integration_test

import (
	"context"
	"fmt"
	"net/http"
)

const frontParticipantBody = `{"participant":"front_app"}`

const invoiceListModuleYAML = `consumes:
  payments:
    rest:
      /invoices:
        get:
          responses:
            200: InvoiceList
schemas:
  InvoiceList:
    type: object
    properties:
      id:
        type: string
      name:
        type: string
`

const invoiceDetailModuleYAML = `consumes:
  payments:
    rest:
      /invoices:
        get:
          responses:
            200: InvoiceDetail
schemas:
  InvoiceDetail:
    type: object
    properties:
      id:
        type: string
      created_at:
        type: string
`

const invoiceCreateModuleYAML = `consumes:
  payments:
    rest:
      /invoices:
        post:
          request: InvoiceCreate
schemas:
  InvoiceCreate:
    type: object
    properties:
      id:
        type: string
      name:
        type: string
`

const invoiceImportModuleYAML = `consumes:
  payments:
    rest:
      /invoices:
        post:
          request: InvoiceImport
schemas:
  InvoiceImport:
    type: object
    properties:
      id:
        type: string
      email:
        type: string
`

const invoiceBothSpellingsYAML = `consumes:
  payments:
    rest:
      /invoices:
        get:
          responses:
            200: InvoiceList
      /invoices/:
        get:
          responses:
            200: InvoiceDetail
`

const invoiceBothSpellingsSchemasYAML = `schemas:
  InvoiceList:
    type: object
    properties:
      id:
        type: string
  InvoiceDetail:
    type: object
    properties:
      created_at:
        type: string
`

const invoiceStringIDModuleYAML = `consumes:
  payments:
    rest:
      /invoices:
        get:
          responses:
            200: InvoiceString
schemas:
  InvoiceString:
    type: object
    properties:
      id:
        type: string
`

const invoiceIntegerIDModuleYAML = `consumes:
  payments:
    rest:
      /invoices:
        get:
          responses:
            200: InvoiceInteger
schemas:
  InvoiceInteger:
    type: object
    properties:
      id:
        type: integer
`

// mergedProperties reads back what the publish stored for the single resource of the
// contract, one line per path: "$.id string required".
func (s *IntegrationSuite) mergedProperties() []string {
	rows, err := s.Pool.Query(context.Background(),
		`SELECT properties.path, property_versions.type, property_versions.optional
		   FROM properties
		   JOIN property_versions ON property_versions.property_id = properties.id
		  ORDER BY properties.path`,
	)
	s.Require().NoError(err)
	defer rows.Close()

	var stored []string
	for rows.Next() {
		var path, propertyType string
		var optional bool
		s.Require().NoError(rows.Scan(&path, &propertyType, &optional))

		presence := "required"
		if optional {
			presence = "optional"
		}

		stored = append(stored, fmt.Sprintf("%s %s %s", path, propertyType, presence))
	}

	return stored
}

func (s *IntegrationSuite) TestPublishContract_ConsumerModulesReadTheSameResponse_MergesByUnion() {
	status, _ := s.post("/api/participants", frontParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("front_app", "1",
		contractFragment{"list.yaml", invoiceListModuleYAML},
		contractFragment{"detail.yaml", invoiceDetailModuleYAML},
	))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("resources"))
	s.Equal([]string{
		"$ object required",
		"$.created_at string required",
		"$.id string required",
		"$.name string required",
	}, s.mergedProperties())
}

func (s *IntegrationSuite) TestPublishContract_ConsumerModulesSendTheSameRequest_PartialFieldsTurnOptional() {
	status, _ := s.post("/api/participants", frontParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("front_app", "1",
		contractFragment{"create.yaml", invoiceCreateModuleYAML},
		contractFragment{"import.yaml", invoiceImportModuleYAML},
	))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("resources"))
	s.Equal([]string{
		"$ object required",
		"$.email string optional",
		"$.id string required",
		"$.name string optional",
	}, s.mergedProperties())
}

func (s *IntegrationSuite) TestPublishContract_BothConsumedSpellingsInOneFile_MergesIntoOneResource() {
	status, _ := s.post("/api/participants", frontParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("front_app", "1",
		contractFragment{"invoices.yaml", invoiceBothSpellingsYAML},
		contractFragment{"schemas.yaml", invoiceBothSpellingsSchemasYAML},
	))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("resources"))
	s.Equal([]string{
		"$ object required",
		"$.created_at string required",
		"$.id string required",
	}, s.mergedProperties())

	var endpoint string
	err := s.Pool.QueryRow(context.Background(), "SELECT endpoint FROM resources").Scan(&endpoint)
	s.Require().NoError(err)
	s.Equal("/invoices", endpoint)
}

func (s *IntegrationSuite) TestPublishContract_ConsumerModulesDisagreeOnPropertyType_Rejected() {
	status, _ := s.post("/api/participants", frontParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("front_app", "1",
		contractFragment{"a.yaml", invoiceStringIDModuleYAML},
		contractFragment{"b.yaml", invoiceIntegerIDModuleYAML},
	))
	s.Equal(http.StatusBadRequest, status)
	s.JSONEq(`{"message":"contract validation failed","violations":[`+
		`"conflicting property type for $.id at consumes payments GET /invoices 200: string (a.yaml) and integer (b.yaml)"`+
		`]}`, body)

	s.Equal(0, s.countRows("contracts"))
	s.Equal(0, s.countRows("resources"))
}

// the union is commutative, so the fragments can arrive in any order and still build
// the very same contract — the checksum proves it by aliasing the stored snapshot
func (s *IntegrationSuite) TestPublishContract_MergedFragmentsInAnyOrder_AliasTheSameSnapshot() {
	status, _ := s.post("/api/participants", frontParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	status, _ = s.post("/api/contracts", s.publishBody("front_app", "1",
		contractFragment{"list.yaml", invoiceListModuleYAML},
		contractFragment{"detail.yaml", invoiceDetailModuleYAML},
	))
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", s.publishBody("front_app", "2",
		contractFragment{"detail.yaml", invoiceDetailModuleYAML},
		contractFragment{"list.yaml", invoiceListModuleYAML},
	))
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("contracts"))
	s.Equal(2, s.countRows("contract_versions"))
	s.Equal(1, s.countRows("resources"))
}

func (s *IntegrationSuite) TestPublishContract_MergedContractRepublished_StaysOneVersion() {
	status, _ := s.post("/api/participants", frontParticipantBody)
	s.Require().Equal(http.StatusOK, status)

	published := s.publishBody("front_app", "1",
		contractFragment{"list.yaml", invoiceListModuleYAML},
		contractFragment{"detail.yaml", invoiceDetailModuleYAML},
	)

	status, _ = s.post("/api/contracts", published)
	s.Require().Equal(http.StatusOK, status)

	status, body := s.post("/api/contracts", published)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"message":"contract publish successful"}`, body)

	s.Equal(1, s.countRows("contracts"))
	s.Equal(1, s.countRows("contract_versions"))
	s.Equal(1, s.countRows("resources"))
}
