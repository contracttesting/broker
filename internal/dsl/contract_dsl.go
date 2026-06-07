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

func (c *Contract) HydrateContract(contract *model.Contract) {
	c.hydrateResources(contract, NewResourcePath(""), *c)
}

func (c *Contract) hydrateResources(
	contract *model.Contract,
	resourcePath ResourcePath,
	unknown any,
) {
	switch unknown := unknown.(type) {
	case Contract:
		dsl := unknown

		for serviceName, consumes := range dsl.ConsumesServices {
			consumerResourcePath := resourcePath.Append("consumes", serviceName)
			c.hydrateResources(
				contract,
				consumerResourcePath,
				consumes,
			)
		}

		c.hydrateResources(
			contract,
			resourcePath.Append("provides"),
			dsl.Provides,
		)

	case Consumes:
		consumes := unknown
		c.hydrateResources(
			contract,
			resourcePath,
			consumes.Rest,
		)

	case Provides:
		provides := unknown
		c.hydrateResources(
			contract,
			resourcePath,
			provides.Rest,
		)
		c.hydrateResources(
			contract,
			resourcePath,
			provides.Message,
		)

	case Rest:
		rest := unknown
		for endpoint, methods := range rest {
			if methods.Get.IsNonZero() {
				c.hydrateResources(
					contract,
					resourcePath.Append("rest", endpoint),
					methods.Get,
				)
			}

			if methods.Post.IsNonZero() {
				c.hydrateResources(
					contract,
					resourcePath.Append("rest", endpoint),
					methods.Post,
				)
			}

			if methods.Put.IsNonZero() {
				c.hydrateResources(
					contract,
					resourcePath.Append("rest", endpoint),
					methods.Put,
				)
			}

			if methods.Delete.IsNonZero() {
				c.hydrateResources(
					contract,
					resourcePath.Append("rest", endpoint),
					methods.Delete,
				)
			}
		}

	case GetMethod:
		getMethod := unknown
		c.hydrateResources(
			contract,
			resourcePath.Append("get", "responses"),
			getMethod.Responses,
		)

	case PostMethod:
		postMethod := unknown
		if postMethod.HasRequestBody() {
			requestResourcePath := resourcePath.Append("post", "request")

			properties := buildSchema(
				NewDepthCounter(postMethod.RequestBody),
				c.Schemas,
				make(map[string]model.Property),
				NewPropertyPath("$"),
				c.Schemas[postMethod.RequestBody],
			)

			contract.AddResource(requestResourcePath.ToResource(properties))
		}

		c.hydrateResources(
			contract,
			resourcePath.Append("post", "responses"),
			postMethod.Responses,
		)

	case PutMethod:
		putMethod := unknown
		if putMethod.HasRequestBody() {
			requestResourcePath := resourcePath.Append("put", "request")

			properties := buildSchema(
				NewDepthCounter(putMethod.RequestBody),
				c.Schemas,
				make(map[string]model.Property),
				NewPropertyPath("$"),
				c.Schemas[putMethod.RequestBody],
			)

			contract.AddResource(requestResourcePath.ToResource(properties))
		}

		c.hydrateResources(
			contract,
			resourcePath.Append("put", "responses"),
			putMethod.Responses,
		)

	case DeleteMethod:
		deleteMethod := unknown
		path := resourcePath.Append("delete", "responses")
		c.hydrateResources(
			contract,
			path,
			deleteMethod.Responses,
		)

	case Responses:
		responses := unknown
		for statusCode, schemaName := range responses {
			responseResourcePath := resourcePath.Append(strconv.Itoa(statusCode))
			properties := buildSchema(
				NewDepthCounter(schemaName),
				c.Schemas,
				make(map[string]model.Property),
				NewPropertyPath("$"),
				c.Schemas[schemaName],
			)

			contract.AddResource(responseResourcePath.ToResource(properties))
		}
	}
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

			return properties
		}

		return properties
	case *Schema:
		depthCounter.Enter()
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
