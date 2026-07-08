package compatibility_checker

import (
	"strconv"
	"strings"

	"github.com/contracttesting/broker/internal/model"
)

// propertyBreak locates a break by path only. Paths are absolute at the
// resource root; inside anyOf variant matching they are relative to the
// variant root, where breaks only signal (in)compatibility and never surface.
type propertyBreak struct {
	reason BreakingReason
	path   string
}

func compareResponseProperties(consumer, provider map[string]model.Property) []propertyBreak {
	var breaks []propertyBreak

	for path, consumerProperty := range consumer {
		if insideAnyOf(path, consumer, provider) {
			continue
		}

		providerProperty, propertyExists := provider[path]

		// If the property is not present in the provider and is required in the consumer, it is a breaking change.
		if !propertyExists {
			if !consumerProperty.Optional {
				breaks = append(breaks, propertyBreak{ReasonPropertyMissingInProvider, path})
			}

			continue
		}

		// A union on either side switches this subtree to variant matching:
		// the provider produces the response, so every provider variant must
		// be accepted by at least one consumer variant.
		if consumerProperty.Type == "anyOf" || providerProperty.Type == "anyOf" {
			if !consumerProperty.Optional && providerProperty.Optional {
				breaks = append(breaks, propertyBreak{ReasonPropertyOptionalInProviderRequiredInConsumer, path})
			}

			breaks = append(breaks, unmatchedVariants(provider, consumer, path,
				func(providerVariant, consumerVariant map[string]model.Property) bool {
					return len(compareResponseProperties(consumerVariant, providerVariant)) == 0
				})...)

			continue
		}

		// If the property is present in the provider and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, propertyBreak{ReasonPropertyTypeMismatch, path})

			continue
		}

		// If the property is required in the consumer and is optional in the provider, it is a breaking change.
		if !consumerProperty.Optional && providerProperty.Optional {
			breaks = append(breaks, propertyBreak{ReasonPropertyOptionalInProviderRequiredInConsumer, path})
		}
	}

	return breaks
}

func compareRequestProperties(consumer, provider map[string]model.Property) []propertyBreak {
	var breaks []propertyBreak

	for path, providerProperty := range provider {
		if insideAnyOf(path, consumer, provider) {
			continue
		}

		consumerProperty, propertyExists := consumer[path]

		// If the property is not present in the consumer and is not optional, it is a breaking change.
		if !propertyExists {
			if !providerProperty.Optional {
				breaks = append(breaks, propertyBreak{ReasonPropertyMissingInConsumer, path})
			}

			continue
		}

		// A union on either side switches this subtree to variant matching:
		// the consumer produces the request, so every consumer variant must
		// be accepted by at least one provider variant.
		if consumerProperty.Type == "anyOf" || providerProperty.Type == "anyOf" {
			if !providerProperty.Optional && consumerProperty.Optional {
				breaks = append(breaks, propertyBreak{ReasonPropertyOptionalInConsumerRequiredInProvider, path})
			}

			breaks = append(breaks, unmatchedVariants(consumer, provider, path,
				func(consumerVariant, providerVariant map[string]model.Property) bool {
					return len(compareRequestProperties(consumerVariant, providerVariant)) == 0
				})...)

			continue
		}

		// If the property is present in the consumer and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, propertyBreak{ReasonPropertyTypeMismatch, path})

			continue
		}

		// If the property is required in the provider and is optional in the consumer, it is a breaking change.
		if !providerProperty.Optional && consumerProperty.Optional {
			breaks = append(breaks, propertyBreak{ReasonPropertyOptionalInConsumerRequiredInProvider, path})
		}
	}

	return breaks
}

// insideAnyOf reports whether the path lives under an anyOf node on either
// side; those subtrees are handled by variant matching at the node itself.
func insideAnyOf(path string, consumer, provider map[string]model.Property) bool {
	for at, char := range path {
		if char != '.' && char != '[' && char != '#' {
			continue
		}

		ancestor := path[:at]
		if consumer[ancestor].Type == "anyOf" || provider[ancestor].Type == "anyOf" {
			return true
		}
	}

	return false
}

// unmatchedVariants applies the union rule at an anyOf node: every variant
// the sender may produce must be accepted by at least one receiver variant.
// The sender is the provider for responses and the consumer for requests; a
// plain schema on either side is the one-variant case. Each unmatched sender
// variant yields one break at its own path.
func unmatchedVariants(
	sender, receiver map[string]model.Property,
	path string,
	accepts func(senderVariant, receiverVariant map[string]model.Property) bool,
) []propertyBreak {
	receiverVariants := variantSubtrees(receiver, path)

	var breaks []propertyBreak
	for _, senderVariant := range variantSubtrees(sender, path) {
		matched := false
		for _, receiverVariant := range receiverVariants {
			if accepts(senderVariant.properties, receiverVariant.properties) {
				matched = true
				break
			}
		}

		if !matched {
			breaks = append(breaks, propertyBreak{ReasonPropertyNotMatchingAnyVariant, senderVariant.path})
		}
	}

	return breaks
}

type variantSubtree struct {
	path       string
	properties map[string]model.Property
}

func variantSubtrees(properties map[string]model.Property, path string) []variantSubtree {
	if properties[path].Type != "anyOf" {
		return []variantSubtree{{path: path, properties: rebaseSubtree(properties, path)}}
	}

	var variants []variantSubtree
	for index := 0; ; index++ {
		variantPath := path + "#" + strconv.Itoa(index)
		if _, exists := properties[variantPath]; !exists {
			break
		}

		variants = append(variants, variantSubtree{path: variantPath, properties: rebaseSubtree(properties, variantPath)})
	}

	return variants
}

// rebaseSubtree extracts the properties rooted at root, rewritten relative to
// "$", so variant subtrees compare positionally regardless of where they live.
func rebaseSubtree(properties map[string]model.Property, root string) map[string]model.Property {
	subtree := make(map[string]model.Property)

	for path, property := range properties {
		if path != root && !extendsRoot(root, path) {
			continue
		}

		rebased := "$" + path[len(root):]
		property.Path = rebased
		subtree[rebased] = property
	}

	return subtree
}

// extendsRoot is a token-boundary prefix check: $.pet extends into $.pet.id,
// $.pet[] and $.pet#0, but not into $.pets.
func extendsRoot(root, path string) bool {
	if len(path) <= len(root) || !strings.HasPrefix(path, root) {
		return false
	}

	next := path[len(root)]

	return next == '.' || next == '[' || next == '#'
}
