package dsl

// Rule is the identity of one validation check in the catalog. Where a rule applies is
// declared by implementing the optional hook interfaces below: the walker, at each
// node, calls every rule of the execution that implements the hook for that node.
type Rule interface{ Code() string }

type EndpointRule interface {
	Rule
	CheckEndpoint(endpoint string, vctx ValidationContext)
}

type RestRule interface {
	Rule
	CheckRest(r Rest, vctx ValidationContext)
}

type SchemasRule interface {
	Rule
	CheckSchemas(s SchemasMap, vctx ValidationContext)
}

// ResourceRule sees each leaf of the walk with the full resource key (direction,
// service when consumes, normalized endpoint, method, request|status).
type ResourceRule interface {
	Rule
	CheckResource(resource string, vctx ValidationContext)
}

// SchemaNameRule sees every schema name a request body or a response references.
type SchemaNameRule interface {
	Rule
	CheckSchemaName(name string, vctx ValidationContext)
}

// SchemaRule sees each node of the schema descent.
type SchemaRule interface {
	Rule
	CheckSchema(s Schema, vctx ValidationContext)
}

type ResponsesRule interface {
	Rule
	CheckResponses(r Responses, vctx ValidationContext)
}

// StatefulRule marks a rule whose checks accumulate state across hook calls. Every
// Validate starts from Fresh instances, so concurrent publishes never share state.
type StatefulRule interface {
	Rule
	Fresh() Rule
}
