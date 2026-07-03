package contract_differ

import "github.com/contracttesting/broker/internal/model"

type ChangeKind string

const (
	ChangeAdded    ChangeKind = "added"
	ChangeModified ChangeKind = "modified"
	ChangeRemoved  ChangeKind = "removed"
)

type ContractDiff struct {
	Resources map[string]ResourceChange
}

type ResourceChange struct {
	Kind       ChangeKind
	Properties map[string]PropertyChange
}

type PropertyChange struct {
	Kind   ChangeKind
	Before model.Property
	After  model.Property
}

type ResourceProperties = map[string]model.Property

func DiffResourceProperties(current, next map[string]ResourceProperties) ContractDiff {
	resourcesChanges := map[string]ResourceChange{}

	for resourceKey, oldProperties := range current {
		nextProperties, exists := next[resourceKey]
		// If the resource is not present in the new contract, it was removed
		if !exists {
			resourcesChanges[resourceKey] = removedResourceChange(oldProperties)
			continue
		}

		// If the resource is present in the new contract and is different, it was modified
		if resourceChanges, resourceWasChanged := modifiedResourceChange(oldProperties, nextProperties); resourceWasChanged {
			resourcesChanges[resourceKey] = resourceChanges
		}
	}

	for newResourceKey, newProperties := range next {
		// If the resource is not present in the previous contract, it was added
		if _, exists := current[newResourceKey]; !exists {
			resourcesChanges[newResourceKey] = addedResourceChange(newProperties)
		}
	}

	return ContractDiff{
		Resources: resourcesChanges,
	}
}

func addedResourceChange(properties ResourceProperties) ResourceChange {
	propertiesChanged := make(map[string]PropertyChange, len(properties))

	for propertyPath, property := range properties {
		propertiesChanged[propertyPath] = PropertyChange{Kind: ChangeAdded, After: property}
	}

	return ResourceChange{Kind: ChangeAdded, Properties: propertiesChanged}
}

func removedResourceChange(properties ResourceProperties) ResourceChange {
	propertiesChanged := make(map[string]PropertyChange, len(properties))

	for propertyPath, property := range properties {
		propertiesChanged[propertyPath] = PropertyChange{Kind: ChangeRemoved, Before: property}
	}

	return ResourceChange{Kind: ChangeRemoved, Properties: propertiesChanged}
}

func modifiedResourceChange(prev, next ResourceProperties) (ResourceChange, bool) {
	propertiesChanged := map[string]PropertyChange{}

	for propertyPath, property := range prev {
		nextProperty, exists := next[propertyPath]
		// If the property is not present in the next resource, it was removed
		if !exists {
			propertiesChanged[propertyPath] = PropertyChange{Kind: ChangeRemoved, Before: property}
			continue
		}

		// If the property is present in the next resource and is different, it was modified
		if !property.IsSame(&nextProperty) {
			propertiesChanged[propertyPath] = PropertyChange{Kind: ChangeModified, Before: property, After: nextProperty}
		}
	}

	for path, nextProperty := range next {
		// If the property is not present in the previous resource, it was added
		if _, exists := prev[path]; !exists {
			propertiesChanged[path] = PropertyChange{Kind: ChangeAdded, After: nextProperty}
		}
	}

	// If no properties were changed, the resource was not modified
	if len(propertiesChanged) == 0 {
		return ResourceChange{}, false
	}

	// If properties were changed, the resource was modified
	return ResourceChange{
		Kind:       ChangeModified,
		Properties: propertiesChanged,
	}, true
}
