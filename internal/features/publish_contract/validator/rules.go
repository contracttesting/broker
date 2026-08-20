package validator

const (
	SegmentRest       = "rest"
	SegmentEndpoint   = "endpoint"
	SegmentResource   = "resource"
	SegmentSchemaName = "schema_name"
	SegmentSchemas    = "schemas"
	SegmentSchema     = "schema"
	SegmentResponses  = "responses"
)

type Rule interface {
	Code() string
	Validate(segment any, validationContext *ValidationContext)
}

type StatefulRule interface {
	Rule
	Fresh() StatefulRule
}
