package descriptor

import (
	"fmt"
	"strconv"

	"github.com/contracttesting/broker/internal/features/publish_contract/dsl"
	"github.com/contracttesting/broker/internal/validations"
)

var (
	Text      = String()
	Flag      = Bool()
	SchemaRef = String()

	SchemaType = Enum{
		Code:    "schema.invalid_type",
		Allowed: []string{"object", "array", "string", "integer", "float", "boolean"},
	}

	Endpoint     = Key{Code: "endpoint.syntax", Check: endpointSyntax}
	ServiceName  = Key{Code: "service.name_syntax", Check: validations.ParticipantName}
	StatusCode   = Key{Code: "status.out_of_range", Check: statusCodeInRange}
	SchemaName   = Key{}
	PropertyName = Key{}
)

func endpointSyntax(key string) error {
	return validations.Endpoint(dsl.NormalizeEndpoint(key))
}

const (
	minStatusCode = 100
	maxStatusCode = 599
)

func statusCodeInRange(key string) error {
	status, err := strconv.Atoi(key)
	if err != nil || status < minStatusCode || status > maxStatusCode {
		return fmt.Errorf("must be between %d and %d", minStatusCode, maxStatusCode)
	}

	return nil
}
