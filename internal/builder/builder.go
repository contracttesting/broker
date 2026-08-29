package builder

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/contracttesting/broker/internal/common"
	"github.com/contracttesting/broker/internal/dsl"
	"github.com/contracttesting/broker/internal/mapper/schemamapper"
	"github.com/contracttesting/broker/internal/model"
)

func Hydrate(fragments []dsl.Fragment, contract *model.UploadedContract) error {
	hydrator := newHydrator(fragments)

	for _, fragment := range fragments {
		if err := hydrator.hydrateResources(
			fragment.Source,
			dsl.NewResourcePath(""),
			*fragment.Contract,
		); err != nil {
			return err
		}
	}

	return hydrator.materialize(contract)
}

type hydrator struct {
	schemas   dsl.SchemasMap
	collected []*collectedResource
	merging   map[string]*collectedResource
}

func newHydrator(fragments []dsl.Fragment) *hydrator {
	h := &hydrator{
		schemas: make(dsl.SchemasMap),
		merging: make(map[string]*collectedResource),
	}

	for _, fragment := range fragments {
		for name, schema := range fragment.Contract.Schemas {
			h.schemas[name] = schema
		}
	}

	return h
}

func (h *hydrator) hydrateResources(
	source string,
	resourcePath dsl.ResourcePath,
	unknown any,
) error {
	switch unknown := unknown.(type) {
	case dsl.Contract:
		for _, serviceName := range slices.Sorted(maps.Keys(unknown.ConsumesServices)) {
			consumerResourcePath := resourcePath.Append("consumes", serviceName)
			if err := h.hydrateResources(source, consumerResourcePath, unknown.ConsumesServices[serviceName]); err != nil {
				return err
			}
		}

		return h.hydrateResources(
			source,
			resourcePath.Append("provides"),
			unknown.Provides,
		)

	case dsl.Consumes:
		return h.hydrateResources(source, resourcePath, unknown.Rest)

	case dsl.Provides:
		return h.hydrateResources(source, resourcePath, unknown.Rest)

	case dsl.Rest:
		for _, endpoint := range slices.Sorted(maps.Keys(unknown)) {
			methods := unknown[endpoint]

			// both spellings of an endpoint reach here alive, and normalizing the key
			// is what lands them on the same resource
			endpointPath := resourcePath.Append("rest", common.NormalizeEndpoint(endpoint))

			if methods.Get.IsNonZero() {
				if err := h.hydrateResources(source, endpointPath, methods.Get); err != nil {
					return err
				}
			}

			if methods.Post.IsNonZero() {
				if err := h.hydrateResources(source, endpointPath, methods.Post); err != nil {
					return err
				}
			}

			if methods.Put.IsNonZero() {
				if err := h.hydrateResources(source, endpointPath, methods.Put); err != nil {
					return err
				}
			}

			if methods.Delete.IsNonZero() {
				if err := h.hydrateResources(source, endpointPath, methods.Delete); err != nil {
					return err
				}
			}
		}

	case dsl.GetMethod:
		return h.hydrateResources(
			source,
			resourcePath.Append("get", "responses"),
			unknown.Responses,
		)

	case dsl.PostMethod:
		if unknown.HasRequestBody() {
			requestResourcePath := resourcePath.Append("post", "request")
			if err := h.collectResource(source, requestResourcePath, unknown.RequestBody); err != nil {
				return err
			}
		}

		return h.hydrateResources(
			source,
			resourcePath.Append("post", "responses"),
			unknown.Responses,
		)

	case dsl.PutMethod:
		if unknown.HasRequestBody() {
			requestResourcePath := resourcePath.Append("put", "request")
			if err := h.collectResource(source, requestResourcePath, unknown.RequestBody); err != nil {
				return err
			}
		}

		return h.hydrateResources(
			source,
			resourcePath.Append("put", "responses"),
			unknown.Responses,
		)

	case dsl.DeleteMethod:
		return h.hydrateResources(
			source,
			resourcePath.Append("delete", "responses"),
			unknown.Responses,
		)

	case dsl.Responses:
		for _, statusCode := range slices.Sorted(maps.Keys(unknown)) {
			responseResourcePath := resourcePath.Append(strconv.Itoa(statusCode))
			if err := h.collectResource(source, responseResourcePath, unknown[statusCode]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *hydrator) collectResource(
	source string,
	resourcePath dsl.ResourcePath,
	schemaName string,
) error {
	flattened, err := flattenResourceSchema(h.schemas, schemaName)
	if err != nil {
		return err
	}

	h.entryFor(resourcePath, source).declare(flattened)

	return nil
}

func (h *hydrator) entryFor(resourcePath dsl.ResourcePath, source string) *collectedResource {
	if entry, merging := h.merging[resourcePath.String()]; merging {
		return entry
	}

	entry := newCollectedResource(resourcePath, source)
	h.collected = append(h.collected, entry)

	if resourcePath.IsConsumer() {
		h.merging[resourcePath.String()] = entry
	}

	return entry
}

func (h *hydrator) materialize(contract *model.UploadedContract) error {
	for _, collected := range h.collected {
		resource := collected.resourcePath.ToResource(collected.properties())

		if err := contract.AddResource(resource, collected.source); err != nil {
			return err
		}
	}

	return nil
}

// collectedResource is one resource under construction: every declaration of it counted,
// and every property path with the declarations that carried it.
type collectedResource struct {
	resourcePath dsl.ResourcePath
	source       string
	declarations int
	declared     map[string]*declaredProperty
}

type declaredProperty struct {
	propertyType string
	declaredIn   int
	optionalIn   int
}

func newCollectedResource(resourcePath dsl.ResourcePath, source string) *collectedResource {
	return &collectedResource{
		resourcePath: resourcePath,
		source:       source,
		declared:     map[string]*declaredProperty{},
	}
}

func (c *collectedResource) declare(flattened map[string]model.Property) {
	c.declarations++

	for path, flat := range flattened {
		property, known := c.declared[path]
		if !known {
			property = &declaredProperty{propertyType: flat.Type}
			c.declared[path] = property
		}

		property.declaredIn++
		if flat.Optional {
			property.optionalIn++
		}
	}
}

func (c *collectedResource) properties() map[string]model.Property {
	properties := make(map[string]model.Property, len(c.declared))

	for path, property := range c.declared {
		properties[path] = model.NewProperty(path, property.propertyType, c.isOptional(property))
	}

	return properties
}

// isOptional reads the union the way the compatibility check will. On a request the app
// only sends what every module sends, so a path missing from — or optional in — one
// declaration is optional for all; on a response the strongest reader wins, and a module
// that never mentions the path has no say in it.
func (c *collectedResource) isOptional(property *declaredProperty) bool {
	if c.isRequest() {
		return property.declaredIn < c.declarations || property.optionalIn > 0
	}

	return property.optionalIn == property.declaredIn
}

func (c *collectedResource) isRequest() bool {
	return strings.HasSuffix(c.resourcePath.String(), ";request")
}

// flattenResourceSchema resolves the paths a schema declares. The descent itself lives
// in schemamapper.ToPropertyModels; what is left here is the rejection of a type
// validation would already have caught.
func flattenResourceSchema(schemas dsl.SchemasMap, schemaName string) (map[string]model.Property, error) {
	flattened := schemamapper.ToPropertyModels(schemas, schemas[schemaName])

	for _, path := range slices.Sorted(maps.Keys(flattened)) {
		if !dsl.IsSupportedType(flattened[path].Type) {
			return nil, fmt.Errorf("unknown schema type %q at %s", flattened[path].Type, path)
		}
	}

	return flattened, nil
}
