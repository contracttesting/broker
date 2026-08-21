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

func (c Contract) Validate(vctx ValidationContext) {
	c.Provides.Validate(vctx.At("provides"))
	c.ConsumesServices.Validate(vctx)
	c.Schemas.Validate(vctx)
}

// Fragment is one uploaded file: the contract parsed out of it plus the path it came
// from, which every publish error quotes.
type Fragment struct {
	Source   string
	Contract *Contract
}

// HydrateFragments merges the fragments into a single contract: the schemas of every
// fragment form one namespace, and each fragment is hydrated against that namespace,
// so refs cross files and resources accumulate into one contract.
//
// It transforms, it does not validate: the fragments have already been through
// Normalize and Validate, so an error here is a broken invariant, not bad input.
func HydrateFragments(fragments []Fragment, contract *model.UploadedContract) error {
	hydrator := newHydrator(fragments)

	for _, fragment := range fragments {
		if err := hydrator.hydrateResources(
			contract,
			fragment.Source,
			NewResourcePath(""),
			*fragment.Contract,
		); err != nil {
			return err
		}
	}

	return nil
}

type hydrator struct {
	schemas SchemasMap
}

func newHydrator(fragments []Fragment) *hydrator {
	h := &hydrator{schemas: make(SchemasMap)}

	for _, fragment := range fragments {
		for name, schema := range fragment.Contract.Schemas {
			h.schemas[name] = schema
		}
	}

	return h
}

func (h *hydrator) hydrateResources(
	contract *model.UploadedContract,
	source string,
	resourcePath ResourcePath,
	unknown any,
) error {
	switch unknown := unknown.(type) {
	case Contract:
		for serviceName, consumes := range unknown.ConsumesServices {
			consumerResourcePath := resourcePath.Append("consumes", serviceName)
			if err := h.hydrateResources(contract, source, consumerResourcePath, consumes); err != nil {
				return err
			}
		}

		return h.hydrateResources(
			contract,
			source,
			resourcePath.Append("provides"),
			unknown.Provides,
		)

	case Consumes:
		return h.hydrateResources(contract, source, resourcePath, unknown.Rest)

	case Provides:
		return h.hydrateResources(contract, source, resourcePath, unknown.Rest)

	case Rest:
		for endpoint, methods := range unknown {
			endpointPath := resourcePath.Append("rest", endpoint)

			if methods.Get.IsNonZero() {
				if err := h.hydrateResources(contract, source, endpointPath, methods.Get); err != nil {
					return err
				}
			}

			if methods.Post.IsNonZero() {
				if err := h.hydrateResources(contract, source, endpointPath, methods.Post); err != nil {
					return err
				}
			}

			if methods.Put.IsNonZero() {
				if err := h.hydrateResources(contract, source, endpointPath, methods.Put); err != nil {
					return err
				}
			}

			if methods.Delete.IsNonZero() {
				if err := h.hydrateResources(contract, source, endpointPath, methods.Delete); err != nil {
					return err
				}
			}
		}

	case GetMethod:
		return h.hydrateResources(
			contract,
			source,
			resourcePath.Append("get", "responses"),
			unknown.Responses,
		)

	case PostMethod:
		if unknown.HasRequestBody() {
			requestResourcePath := resourcePath.Append("post", "request")
			if err := h.addResource(contract, source, requestResourcePath, unknown.RequestBody); err != nil {
				return err
			}
		}

		return h.hydrateResources(
			contract,
			source,
			resourcePath.Append("post", "responses"),
			unknown.Responses,
		)

	case PutMethod:
		if unknown.HasRequestBody() {
			requestResourcePath := resourcePath.Append("put", "request")
			if err := h.addResource(contract, source, requestResourcePath, unknown.RequestBody); err != nil {
				return err
			}
		}

		return h.hydrateResources(
			contract,
			source,
			resourcePath.Append("put", "responses"),
			unknown.Responses,
		)

	case DeleteMethod:
		return h.hydrateResources(
			contract,
			source,
			resourcePath.Append("delete", "responses"),
			unknown.Responses,
		)

	case Responses:
		for statusCode, schemaName := range unknown {
			responseResourcePath := resourcePath.Append(strconv.Itoa(statusCode))
			if err := h.addResource(contract, source, responseResourcePath, schemaName); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *hydrator) addResource(
	contract *model.UploadedContract,
	source string,
	resourcePath ResourcePath,
	schemaName string,
) error {
	properties := make(map[string]model.Property)
	resource := resourcePath.ToResource(properties)

	if err := buildSchema(
		h.schemas,
		properties,
		NewPropertyPath("$"),
		h.schemas[schemaName],
	); err != nil {
		return err
	}

	return contract.AddResource(resource, source)
}

// buildSchema fills properties by walking the schema, following refs through the
// namespace. Validation has already ruled out the cycles and the missing pieces that
// would keep this walk from terminating.
func buildSchema(
	schemas SchemasMap,
	properties map[string]model.Property,
	propertyPath PropertyPath,
	unknown any,
) error {
	switch unknown := unknown.(type) {
	case Schema:
		if unknown.IsObject() {
			properties[propertyPath.String()] = model.NewProperty(
				propertyPath.String(),
				"object",
				unknown.Optional,
			)

			for name, schemaProperties := range unknown.Properties {
				if err := buildSchema(
					schemas,
					properties,
					propertyPath.Append(name),
					schemaProperties,
				); err != nil {
					return err
				}
			}

			return nil
		}

		if unknown.IsArray() {
			properties[propertyPath.String()] = model.NewProperty(
				propertyPath.String(),
				"array",
				unknown.Optional,
			)

			return buildSchema(
				schemas,
				properties,
				propertyPath.AppendArray(),
				unknown.Items,
			)
		}

		if unknown.IsPrimitive() {
			properties[propertyPath.String()] = model.NewProperty(
				propertyPath.String(),
				unknown.Type,
				unknown.Optional,
			)

			return nil
		}

		if unknown.IsRef() {
			return buildSchema(
				schemas,
				properties,
				propertyPath,
				schemas[unknown.Ref],
			)
		}

		return fmt.Errorf("unknown schema type %q at %s", unknown.Type, propertyPath)
	case *Schema:
		return buildSchema(
			schemas,
			properties,
			propertyPath,
			*unknown,
		)
	default:
		return fmt.Errorf("unknown schema type %T", unknown)
	}
}
