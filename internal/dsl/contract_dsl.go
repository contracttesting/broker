package dsl

import (
	"fmt"
	"strconv"

	"github.com/contracttesting/broker/internal/model"
)

type Contract struct {
	Provides         Provides            `json:"provides,omitzero"`
	ConsumesServices ConsumesServicesMap `json:"consumes,omitzero"`
	Schemas          SchemasMap          `json:"schemas,omitzero"`
}

func (c *Contract) HydrateContract(contract *model.UploadedContract) error {
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
	properties := buildSchema(
		NewDepthCounter(schemaName),
		c.Schemas,
		make(map[string]model.Property),
		NewPropertyPath("$"),
		c.Schemas[schemaName],
	)

	return contract.AddResource(resourcePath.ToResource(properties))
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
