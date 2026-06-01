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
	Resource   model.Resource
	Properties map[string]PropertyChange
}

type PropertyChange struct {
	Kind   ChangeKind
	Before model.Property
	After  model.Property
}

func Diff(oldContract *model.Contract, newContract *model.Contract) ContractDiff {
	var upcoming map[string]model.Resource

	if newContract != nil {
		upcoming = newContract.Resources
	}

	resoucesChanges := map[string]ResourceChange{}

	for resourceKey, oldResource := range oldContract.Resources {
		newResource, exists := upcoming[resourceKey]
		// If the resource is not present in the new contract, it was removed
		if !exists {
			resoucesChanges[resourceKey] = removedResourceChange(oldResource)
			continue
		}

		// If the resource is present in the new contract and is different, it was modified
		if resourceChanges, resourceWasChanged := modifiedResourceChange(oldResource, newResource); resourceWasChanged {
			resoucesChanges[resourceKey] = resourceChanges
		}
	}

	for newResourceKey, newResource := range upcoming {
		// If the resource is not present in the previous contract, it was added
		if _, exists := oldContract.Resources[newResourceKey]; !exists {
			resoucesChanges[newResourceKey] = addedResourceChange(newResource)
		}
	}

	return ContractDiff{Resources: resoucesChanges}
}

func addedResourceChange(r model.Resource) ResourceChange {
	propertiesChanged := make(map[string]PropertyChange, len(r.Properties))

	for propertyPath, property := range r.Properties {
		// If the property is present in the previous resource, it was added
		propertiesChanged[propertyPath] = PropertyChange{Kind: ChangeAdded, After: property}
	}

	return ResourceChange{Kind: ChangeAdded, Resource: r, Properties: propertiesChanged}
}

func removedResourceChange(r model.Resource) ResourceChange {
	propertiesChanged := make(map[string]PropertyChange, len(r.Properties))

	for propertyPath, property := range r.Properties {
		// If the property is present in the previous resource, it was removed
		propertiesChanged[propertyPath] = PropertyChange{Kind: ChangeRemoved, Before: property}
	}

	return ResourceChange{Kind: ChangeRemoved, Resource: r, Properties: propertiesChanged}
}

func modifiedResourceChange(prev, next model.Resource) (ResourceChange, bool) {
	propertiesChanged := map[string]PropertyChange{}

	for propertyPath, property := range prev.Properties {
		nextProperty, exists := next.Properties[propertyPath]
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

	for path, nextProperty := range next.Properties {
		// If the property is not present in the previous resource, it was added
		if _, exists := prev.Properties[path]; !exists {
			propertiesChanged[path] = PropertyChange{Kind: ChangeAdded, After: nextProperty}
		}
	}

	// If no properties were changed, the resource was not modified
	if len(propertiesChanged) == 0 {
		return ResourceChange{}, false
	}

	// If properties were changed, the resource was modified
	return ResourceChange{Kind: ChangeModified, Resource: next, Properties: propertiesChanged}, true
}
