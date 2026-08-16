package dsl

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/contracttesting/broker/internal/model"
)

type Contract struct {
	Provides         Provides            `json:"provides,omitzero"`
	ConsumesServices ConsumesServicesMap `json:"consumes,omitzero"`
	Schemas          SchemasMap          `json:"schemas,omitzero"`
}

func (c *Contract) HydrateContract(contract *model.UploadedContract) error {
	if err := c.validateSchemaNames(); err != nil {
		return err
	}

	for endpoint := range c.Provides.Rest {
		if err := validateEndpoint(endpoint); err != nil {
			return err
		}
	}

	for _, consumes := range c.ConsumesServices {
		for endpoint := range consumes.Rest {
			if err := validateEndpoint(endpoint); err != nil {
				return err
			}
		}
	}

	return c.hydrateResources(contract, NewResourcePath(""), *c)
}

// validateSchemaNames checks every ref of the contract, including refs inside schemas
// no endpoint reaches.
func (c *Contract) validateSchemaNames() error {
	for _, name := range slices.Sorted(maps.Keys(c.Schemas)) {
		if err := c.validateRefs(c.Schemas[name], NewPropertyPath(name)); err != nil {
			return err
		}
	}

	return nil
}

// validateRefs walks a schema structurally without following its refs: every schema of
// the contract is walked on its own, so a cycle is never entered here.
func (c *Contract) validateRefs(schema Schema, propertyPath PropertyPath) error {
	switch {
	case schema.IsRef():
		if _, resolved := c.Schemas[schema.Ref]; !resolved {
			return unresolvedSchemaName(schema.Ref, propertyPath.String())
		}

	case schema.IsArray():
		if schema.Items != nil {
			return c.validateRefs(*schema.Items, propertyPath.AppendArray())
		}

	case schema.IsObject():
		for _, name := range slices.Sorted(maps.Keys(schema.Properties)) {
			if err := c.validateRefs(schema.Properties[name], propertyPath.Append(name)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Contract) hydrateResources(
	contract *model.UploadedContract,
	resourcePath ResourcePath,
	unknown any,
) error {
	switch unknown := unknown.(type) {
	case Contract:
		for serviceName, consumes := range unknown.ConsumesServices {
			consumerResourcePath := resourcePath.Append("consumes", serviceName)
			if err := c.hydrateResources(contract, consumerResourcePath, consumes); err != nil {
				return err
			}
		}

		return c.hydrateResources(
			contract,
			resourcePath.Append("provides"),
			unknown.Provides,
		)

	case Consumes:
		return c.hydrateResources(contract, resourcePath, unknown.Rest)

	case Provides:
		return c.hydrateResources(contract, resourcePath, unknown.Rest)

	case Rest:
		for endpoint, methods := range unknown {
			endpointPath := resourcePath.Append("rest", normalizeEndpoint(endpoint))

			if methods.Get.IsNonZero() {
				if err := c.hydrateResources(contract, endpointPath, methods.Get); err != nil {
					return err
				}
			}

			if methods.Post.IsNonZero() {
				if err := c.hydrateResources(contract, endpointPath, methods.Post); err != nil {
					return err
				}
			}

			if methods.Put.IsNonZero() {
				if err := c.hydrateResources(contract, endpointPath, methods.Put); err != nil {
					return err
				}
			}

			if methods.Delete.IsNonZero() {
				if err := c.hydrateResources(contract, endpointPath, methods.Delete); err != nil {
					return err
				}
			}
		}

	case GetMethod:
		return c.hydrateResources(
			contract,
			resourcePath.Append("get", "responses"),
			unknown.Responses,
		)

	case PostMethod:
		if unknown.HasRequestBody() {
			requestResourcePath := resourcePath.Append("post", "request")
			if err := c.addResource(contract, requestResourcePath, unknown.RequestBody); err != nil {
				return err
			}
		}

		return c.hydrateResources(
			contract,
			resourcePath.Append("post", "responses"),
			unknown.Responses,
		)

	case PutMethod:
		if unknown.HasRequestBody() {
			requestResourcePath := resourcePath.Append("put", "request")
			if err := c.addResource(contract, requestResourcePath, unknown.RequestBody); err != nil {
				return err
			}
		}

		return c.hydrateResources(
			contract,
			resourcePath.Append("put", "responses"),
			unknown.Responses,
		)

	case DeleteMethod:
		return c.hydrateResources(
			contract,
			resourcePath.Append("delete", "responses"),
			unknown.Responses,
		)

	case Responses:
		for statusCode, schemaName := range unknown {
			responseResourcePath := resourcePath.Append(strconv.Itoa(statusCode))
			if err := c.addResource(contract, responseResourcePath, schemaName); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Contract) addResource(
	contract *model.UploadedContract,
	resourcePath ResourcePath,
	schemaName string,
) error {
	properties := make(map[string]model.Property)
	resource := resourcePath.ToResource(properties)

	schema, resolved := c.Schemas[schemaName]
	if !resolved {
		return unresolvedSchemaName(schemaName, resource.Describe())
	}

	buildSchema(
		NewDepthCounter(schemaName),
		c.Schemas,
		properties,
		NewPropertyPath("$"),
		schema,
	)

	return contract.AddResource(resource)
}

func unresolvedSchemaName(name, position string) error {
	return fmt.Errorf("unresolved schema name: %s referenced at %s", name, position)
}

func buildSchema(
	depthCounter *DepthCounter,
	schemas SchemasMap,
	properties map[string]model.Property,
	propertyPath PropertyPath,
	unknown any,
) map[string]model.Property {
	switch unknown := unknown.(type) {
	case Schema:
		if unknown.IsObject() {
			properties[propertyPath.String()] = model.NewProperty(
				propertyPath.String(),
				"object",
				unknown.Optional,
			)

			for name, schemaProperties := range unknown.Properties {
				depthCounter.Enter()
				properties = buildSchema(
					depthCounter,
					schemas,
					properties,
					propertyPath.Append(name),
					schemaProperties,
				)
				depthCounter.Exit()
			}

			return properties
		}

		if unknown.IsArray() {
			properties[propertyPath.String()] = model.NewProperty(
				propertyPath.String(),
				"array",
				unknown.Optional,
			)

			depthCounter.Enter()
			properties = buildSchema(
				depthCounter,
				schemas,
				properties,
				propertyPath.AppendArray(),
				unknown.Items,
			)
			depthCounter.Exit()

			return properties
		}

		if unknown.IsPrimitive() {
			properties[propertyPath.String()] = model.NewProperty(
				propertyPath.String(),
				unknown.Type,
				unknown.Optional,
			)

			return properties
		}

		if unknown.IsRef() {
			depthCounter.Enter()
			properties = buildSchema(
				depthCounter,
				schemas,
				properties,
				propertyPath,
				schemas[unknown.Ref],
			)
			depthCounter.Exit()

			return properties
		}

		return properties
	case *Schema:
		return buildSchema(
			depthCounter,
			schemas,
			properties,
			propertyPath,
			*unknown,
		)
	default:
		panic(fmt.Sprintf("unknown schema type %T", unknown))
	}
}
