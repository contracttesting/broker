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

// Fragment is one uploaded file: the contract parsed out of it plus the path it came
// from, which every publish error quotes.
type Fragment struct {
	Source   string
	Contract *Contract
}

// HydrateFragments merges the fragments into a single contract: the schemas of every
// fragment form one namespace, and each fragment is hydrated against that namespace,
// so refs cross files and resources accumulate into one contract.
func HydrateFragments(fragments []Fragment, contract *model.UploadedContract) error {
	hydrator, err := newHydrator(fragments)
	if err != nil {
		return err
	}

	if err := hydrator.validateSchemaNames(); err != nil {
		return err
	}

	for _, fragment := range fragments {
		if err := hydrator.hydrateFragment(fragment, contract); err != nil {
			return err
		}
	}

	return nil
}

type hydrator struct {
	schemas       SchemasMap
	schemaSources map[string]string
}

func newHydrator(fragments []Fragment) (*hydrator, error) {
	h := &hydrator{
		schemas:       make(SchemasMap),
		schemaSources: make(map[string]string),
	}

	for _, fragment := range fragments {
		for _, name := range slices.Sorted(maps.Keys(fragment.Contract.Schemas)) {
			if declaredIn, taken := h.schemaSources[name]; taken {
				return nil, fmt.Errorf(
					"duplicate schema: %s declared in %s and %s",
					name,
					declaredIn,
					fragment.Source,
				)
			}

			h.schemas[name] = fragment.Contract.Schemas[name]
			h.schemaSources[name] = fragment.Source
		}
	}

	return h, nil
}

// validateSchemaNames checks every ref of the namespace, including refs inside schemas
// no endpoint reaches.
func (h *hydrator) validateSchemaNames() error {
	for _, name := range slices.Sorted(maps.Keys(h.schemas)) {
		if err := h.validateRefs(
			h.schemas[name],
			NewPropertyPath(name),
			h.schemaSources[name],
		); err != nil {
			return err
		}
	}

	return nil
}

// validateRefs walks a schema structurally without following its refs: every schema of
// the namespace is walked on its own, so a cycle is never entered here.
func (h *hydrator) validateRefs(schema Schema, propertyPath PropertyPath, source string) error {
	switch {
	case schema.IsRef():
		if _, resolved := h.schemas[schema.Ref]; !resolved {
			return unresolvedSchemaName(schema.Ref, propertyPath.String(), source)
		}

	case schema.IsArray():
		if schema.Items != nil {
			return h.validateRefs(*schema.Items, propertyPath.AppendArray(), source)
		}

	case schema.IsObject():
		for _, name := range slices.Sorted(maps.Keys(schema.Properties)) {
			if err := h.validateRefs(
				schema.Properties[name],
				propertyPath.Append(name),
				source,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *hydrator) hydrateFragment(fragment Fragment, contract *model.UploadedContract) error {
	for endpoint := range fragment.Contract.Provides.Rest {
		if err := validateEndpoint(endpoint); err != nil {
			return fmt.Errorf("%w (%s)", err, fragment.Source)
		}
	}

	for _, consumes := range fragment.Contract.ConsumesServices {
		for endpoint := range consumes.Rest {
			if err := validateEndpoint(endpoint); err != nil {
				return fmt.Errorf("%w (%s)", err, fragment.Source)
			}
		}
	}

	return h.hydrateResources(contract, fragment.Source, NewResourcePath(""), *fragment.Contract)
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
			endpointPath := resourcePath.Append("rest", normalizeEndpoint(endpoint))

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

	schema, resolved := h.schemas[schemaName]
	if !resolved {
		return unresolvedSchemaName(schemaName, resource.Describe(), source)
	}

	if err := buildSchema(
		NewDepthCounter(schemaName),
		h.schemas,
		properties,
		NewPropertyPath("$"),
		schema,
	); err != nil {
		return fmt.Errorf("%w (%s)", err, h.schemaSources[schemaName])
	}

	return contract.AddResource(resource, source)
}

func unresolvedSchemaName(name, position, source string) error {
	return fmt.Errorf("unresolved schema name: %s referenced at %s (%s)", name, position, source)
}

// buildSchema fills properties by walking the schema, following refs through the
// namespace. It returns SchemaTooDeep once the walk passes MAX_DEPTH levels, which is
// also how a cyclic chain of refs terminates.
func buildSchema(
	depthCounter *DepthCounter,
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
				if err := descend(
					depthCounter,
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

			return descend(
				depthCounter,
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
			return descend(
				depthCounter,
				schemas,
				properties,
				propertyPath,
				schemas[unknown.Ref],
			)
		}

		return nil
	case *Schema:
		return buildSchema(
			depthCounter,
			schemas,
			properties,
			propertyPath,
			*unknown,
		)
	default:
		return fmt.Errorf("unknown schema type %T", unknown)
	}
}

// descend takes buildSchema one level down, counting the level in and back out.
func descend(
	depthCounter *DepthCounter,
	schemas SchemasMap,
	properties map[string]model.Property,
	propertyPath PropertyPath,
	unknown any,
) error {
	if err := depthCounter.Enter(); err != nil {
		return err
	}

	if err := buildSchema(depthCounter, schemas, properties, propertyPath, unknown); err != nil {
		return err
	}

	depthCounter.Exit()

	return nil
}
