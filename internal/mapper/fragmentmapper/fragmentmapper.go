package fragmentmapper

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/contracttesting/broker/internal/common"
	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/mapper/resourcepathmapper"
	"github.com/contracttesting/broker/internal/mapper/schemamapper"
	"github.com/contracttesting/broker/internal/model"
)

func ToResourceModels(fragments []dsl.Fragment) ([]model.UploadedResource, error) {
	schemas := schemasFromFragments(fragments)

	var declarations []resourceDeclaration
	for _, fragment := range fragments {
		fragmentDeclarations, err := declarationsFromFragment(fragment, schemas)
		if err != nil {
			return nil, err
		}

		declarations = append(declarations, fragmentDeclarations...)
	}

	merged, err := mergeDeclarationsByResourcePath(declarations)
	if err != nil {
		return nil, err
	}

	resources := make([]model.UploadedResource, 0, len(merged))
	for _, declaration := range merged {
		resources = append(resources, declaration.resource)
	}

	return resources, nil
}

func schemasFromFragments(fragments []dsl.Fragment) dsl.SchemasMap {
	schemas := make(dsl.SchemasMap)

	for _, fragment := range fragments {
		for name, schema := range fragment.Contract.Schemas {
			if _, declared := schemas[name]; !declared {
				schemas[name] = schema
			}
		}
	}

	return schemas
}

func declarationsFromFragment(fragment dsl.Fragment, schemas dsl.SchemasMap) ([]resourceDeclaration, error) {
	root := dsl.NewResourcePath("")

	var declarations []resourceDeclaration
	for _, serviceName := range slices.Sorted(maps.Keys(fragment.Contract.ConsumesServices)) {
		consumed, err := declarationsFromRest(
			fragment.Source,
			fragment.Contract.ConsumesServices[serviceName].Rest,
			root.Append("consumes", serviceName),
			schemas,
		)
		if err != nil {
			return nil, err
		}

		declarations = append(declarations, consumed...)
	}

	provided, err := declarationsFromRest(
		fragment.Source,
		fragment.Contract.Provides.Rest,
		root.Append("provides"),
		schemas,
	)
	if err != nil {
		return nil, err
	}

	return append(declarations, provided...), nil
}

func declarationsFromRest(source string, rest dsl.Rest, resourcePath dsl.ResourcePath, schemas dsl.SchemasMap) ([]resourceDeclaration, error) {
	var declarations []resourceDeclaration
	for _, endpoint := range slices.Sorted(maps.Keys(rest)) {
		endpointPath := resourcePath.Append("rest", common.NormalizeEndpoint(endpoint))

		fromMethods, err := declarationsFromMethods(source, rest[endpoint], endpointPath, schemas)
		if err != nil {
			return nil, err
		}

		declarations = append(declarations, fromMethods...)
	}

	return declarations, nil
}

func declarationsFromMethods(source string, methods dsl.HttpMethods, resourcePath dsl.ResourcePath, schemas dsl.SchemasMap) ([]resourceDeclaration, error) {
	var declarations []resourceDeclaration

	if methods.Get.IsNonZero() {
		responses, err := declarationsFromResponses(source, methods.Get.Responses, resourcePath.Append("get", "responses"), schemas)
		if err != nil {
			return nil, err
		}

		declarations = append(declarations, responses...)
	}

	if methods.Post.IsNonZero() {
		if methods.Post.HasRequestBody() {
			request, err := declarationFromRequestBody(source, methods.Post.RequestBody, resourcePath.Append("post", "request"), schemas)
			if err != nil {
				return nil, err
			}

			declarations = append(declarations, request)
		}

		responses, err := declarationsFromResponses(source, methods.Post.Responses, resourcePath.Append("post", "responses"), schemas)
		if err != nil {
			return nil, err
		}

		declarations = append(declarations, responses...)
	}

	if methods.Put.IsNonZero() {
		if methods.Put.HasRequestBody() {
			request, err := declarationFromRequestBody(source, methods.Put.RequestBody, resourcePath.Append("put", "request"), schemas)
			if err != nil {
				return nil, err
			}

			declarations = append(declarations, request)
		}

		responses, err := declarationsFromResponses(source, methods.Put.Responses, resourcePath.Append("put", "responses"), schemas)
		if err != nil {
			return nil, err
		}

		declarations = append(declarations, responses...)
	}

	if methods.Delete.IsNonZero() {
		responses, err := declarationsFromResponses(source, methods.Delete.Responses, resourcePath.Append("delete", "responses"), schemas)
		if err != nil {
			return nil, err
		}

		declarations = append(declarations, responses...)
	}

	return declarations, nil
}

func declarationFromRequestBody(source, schemaName string, resourcePath dsl.ResourcePath, schemas dsl.SchemasMap) (resourceDeclaration, error) {
	return declarationFromSchema(source, schemaName, resourcePath, schemas)
}

func declarationsFromResponses(source string, responses dsl.Responses, resourcePath dsl.ResourcePath, schemas dsl.SchemasMap) ([]resourceDeclaration, error) {
	var declarations []resourceDeclaration
	for _, statusCode := range slices.Sorted(maps.Keys(responses)) {
		declaration, err := declarationFromSchema(source, responses[statusCode], resourcePath.Append(strconv.Itoa(statusCode)), schemas)
		if err != nil {
			return nil, err
		}

		declarations = append(declarations, declaration)
	}

	return declarations, nil
}

func declarationFromSchema(source, schemaName string, resourcePath dsl.ResourcePath, schemas dsl.SchemasMap) (resourceDeclaration, error) {
	properties := schemamapper.ToPropertyModels(schemas, schemas[schemaName])

	for _, path := range slices.Sorted(maps.Keys(properties)) {
		if !dsl.IsSupportedType(properties[path].Type) {
			return resourceDeclaration{}, fmt.Errorf("unknown schema type %q at %s", properties[path].Type, path)
		}
	}

	return resourceDeclaration{
		source:       source,
		resourcePath: resourcePath,
		resource:     resourcepathmapper.ToResourceModel(resourcePath, properties),
	}, nil
}
