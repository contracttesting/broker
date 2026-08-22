package validator

const (
	SegmentServiceName       = "service_name"
	SegmentEndpoint          = "endpoint"
	SegmentResource          = "resource"
	SegmentSchemaName        = "schema_name"
	SegmentSchemaDeclaration = "schema_declaration"
	SegmentSchema            = "schema"
	SegmentStatusCode        = "status_code"
)

type Rule interface {
	Code() string
	Validate(value any, validationContext *ValidationContext)
}

type StatefulRule interface {
	Rule
	Fresh() StatefulRule
}
