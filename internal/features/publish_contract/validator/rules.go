package validator

const (
	SegmentServiceName       = "service_name"
	SegmentEndpoint          = "endpoint"
	SegmentResource          = "resource"
	SegmentResourceSchema    = "resource_schema"
	SegmentSchemaName        = "schema_name"
	SegmentSchemaDeclaration = "schema_declaration"
	SegmentSchema            = "schema"
	SegmentStatusCode        = "status_code"
)

// ResourceSchema pairs a resource with the schema it answers with, the two halves a
// rule needs to compare one declaration of a resource against another.
type ResourceSchema struct {
	Path       string
	SchemaName string
}

type Rule interface {
	Code() string
	Validate(value any, contextualValidator *ContextualValidator)
}
