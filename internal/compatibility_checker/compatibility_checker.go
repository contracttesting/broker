package compatibility_checker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
)

type BrokenResource struct {
	Kind               model.ResourceKind
	ConsumedProvider   string
	Endpoint           string
	Method             string
	ResponseStatusCode string
}

func NewBrokenResource(resource model.Resource) BrokenResource {
	return BrokenResource{
		Kind:               resource.Kind,
		ConsumedProvider:   resource.ConsumedProvider,
		Endpoint:           resource.Endpoint,
		Method:             resource.Method,
		ResponseStatusCode: resource.ResponseStatusCode,
	}
}

type BreakingReason string

const (
	ReasonProviderNotFound                         BreakingReason = "provider_not_found"
	ReasonProviderResourceNotFound                 BreakingReason = "provider_resource_not_found"
	ReasonMissingInProvider                        BreakingReason = "missing_in_provider"
	ReasonMissingInConsumer                        BreakingReason = "missing_in_consumer"
	ReasonTypeMismatch                             BreakingReason = "type_mismatch"
	ReasonOptionalInProviderRequiredInConsumer     BreakingReason = "optional_in_provider_required_in_consumer"
	ReasonOptionalInConsumerRequiredInProvider     BreakingReason = "optional_in_consumer_required_in_provider"
	ReasonProviderResourceNotDeployedInEnvironment BreakingReason = "provider_resource_not_deployed_in_environment"
)

type BreakingChange struct {
	LeftResource         *model.Resource `json:"left_resource"`
	RightResource        *model.Resource `json:"right_resource"`
	Reason               BreakingReason  `json:"reason"`
	Property             string          `json:"property"`
	HumanReadable        string          `json:"human_readable"`
	DeployedEnvironments []string        `json:"deployed_environments,omitempty"`
	ConsumerName         string          `json:"consumer_name,omitempty"`
	ProviderName         string          `json:"provider_name,omitempty"`
	ConsumerType         string          `json:"consumer_type,omitempty"`
	ProviderType         string          `json:"provider_type,omitempty"`
}

type CompatibilityResult struct {
	CounterpartParticipantID int64
	CounterpartVersion       string
	Deployable               bool
}

type CompatibilityReport struct {
	Results []CompatibilityResult       `json:"results"`
	Breaks  map[string][]BreakingChange `json:"breaks"`
}

func NewCompatibilityReport() *CompatibilityReport {
	return &CompatibilityReport{
		Results: make([]CompatibilityResult, 0),
		Breaks:  make(map[string][]BreakingChange),
	}
}

func (r *CompatibilityReport) Append(b BreakingChange) {
	if r.Breaks == nil {
		r.Breaks = make(map[string][]BreakingChange)
	}

	r.Breaks[b.LeftResource.ParticipantName()] = append(r.Breaks[b.LeftResource.ParticipantName()], b)
}

func NewBreakingChange(
	leftResource *model.Resource,
	rightResource *model.Resource,
	reason BreakingReason,
	property string,
) BreakingChange {
	breakingChange := BreakingChange{
		LeftResource:  leftResource,
		RightResource: rightResource,
		Reason:        reason,
		Property:      property,
		ConsumerName:  leftResource.ParticipantName(),
		ProviderName:  leftResource.ConsumedProvider,
	}

	if reason == ReasonTypeMismatch {
		breakingChange.ConsumerType = leftResource.Properties[property].Type
		breakingChange.ProviderType = rightResource.Properties[property].Type
	}

	breakingChange.humanReadable()

	return breakingChange
}

func NewProviderNotDeployedBreakingChange(consumer *model.Resource, deployedEnvironments []string) BreakingChange {
	breakingChange := BreakingChange{
		LeftResource:         consumer,
		Reason:               ReasonProviderResourceNotDeployedInEnvironment,
		DeployedEnvironments: deployedEnvironments,
		ConsumerName:         consumer.ParticipantName(),
		ProviderName:         consumer.ConsumedProvider,
	}

	breakingChange.humanReadable()

	return breakingChange
}

func (b *BreakingChange) humanReadable() {
	switch b.Reason {
	case ReasonProviderNotFound:
		b.HumanReadable = fmt.Sprintf(
			"Provider %s not found",
			b.LeftResource.ParticipantName(),
		)

	case ReasonProviderResourceNotFound:
		b.HumanReadable = fmt.Sprintf(
			"No %s was found",
			b.LeftResource.Operation(),
		)

	case ReasonMissingInProvider:
		b.HumanReadable = fmt.Sprintf(
			"Property %s is missing in provider %s on %s",
			b.Property,
			b.RightResource.ParticipantName(),
			b.LeftResource.Operation(),
		)

	case ReasonMissingInConsumer:
		b.HumanReadable = fmt.Sprintf(
			"Property %s is missing in consumer %s on %s",
			b.Property,
			b.LeftResource.ParticipantName(),
			b.LeftResource.Operation(),
		)

	case ReasonTypeMismatch:
		consumerType := b.LeftResource.Properties[b.Property].Type
		providerType := b.RightResource.Properties[b.Property].Type

		b.HumanReadable = fmt.Sprintf(
			"Property %s type mismatch on %s, provider %s expects %s but consumer %s expects %s",
			b.Property,
			b.LeftResource.Operation(),
			b.RightResource.ParticipantName(),
			providerType,
			b.LeftResource.ParticipantName(),
			consumerType,
		)

	case ReasonOptionalInProviderRequiredInConsumer:
		b.HumanReadable = fmt.Sprintf(
			"Property %s is optional in provider %s but required in consumer %s on %s",
			b.Property,
			b.RightResource.ParticipantName(),
			b.LeftResource.ParticipantName(),
			b.LeftResource.Operation(),
		)

	case ReasonOptionalInConsumerRequiredInProvider:
		b.HumanReadable = fmt.Sprintf(
			"Property %s is optional in consumer %s but required in provider %s on %s",
			b.Property,
			b.LeftResource.ParticipantName(),
			b.RightResource.ParticipantName(),
			b.LeftResource.Operation(),
		)

	case ReasonProviderResourceNotDeployedInEnvironment:
		if len(b.DeployedEnvironments) > 0 {
			b.HumanReadable = fmt.Sprintf(
				"%s exists but isn't deployed in this environment yet (deployed in: %s)",
				b.LeftResource.Operation(),
				strings.Join(b.DeployedEnvironments, ", "),
			)
		} else {
			b.HumanReadable = fmt.Sprintf(
				"%s exists but isn't deployed in any environment yet",
				b.LeftResource.Operation(),
			)
		}

	default:
		b.HumanReadable = "Unknown reason"
	}
}

type CompatibilityChecker struct {
	repository *repository.ContractRepository
}

func NewCompatibilityChecker(repository *repository.ContractRepository) *CompatibilityChecker {
	return &CompatibilityChecker{
		repository: repository,
	}
}

func (c *CompatibilityChecker) Check(
	ctx context.Context,
	uploaded *model.Contract,
	environment *model.Environment,
) *CompatibilityReport {
	report := NewCompatibilityReport()

	for _, resource := range uploaded.Resources {
		switch resource.Direction {
		case model.Consumes:
			c.checkConsumer(ctx, resource, environment, report)
		case model.Provides:
			c.checkProvider(ctx, resource, environment, report)
		}
	}

	return report
}

func (c *CompatibilityChecker) checkConsumer(
	ctx context.Context,
	consumer model.Resource,
	environment *model.Environment,
	report *CompatibilityReport,
) {
	provider, err := c.repository.LoadProviderResourceOfConsumer(ctx, consumer)

	if errors.Is(err, repository.ErrProviderResourceNotFound) {
		report.Append(NewBreakingChange(
			&consumer,
			nil,
			ReasonProviderResourceNotFound,
			"",
		))

		report.Results = append(report.Results, CompatibilityResult{
			Deployable: false,
		})

		return
	}

	version, deployed := provider.DeployedVersionIn(environment.Name)
	if !deployed {
		report.Append(NewProviderNotDeployedBreakingChange(
			&consumer,
			provider.DeployedEnvironments(),
		))

		report.Results = append(report.Results, CompatibilityResult{
			Deployable: false,
		})

		return
	}

	provider.Version = version

	breaks := checkResources(&consumer, &provider)
	for _, breakingChange := range breaks {
		report.Append(breakingChange)
	}

	report.Results = append(report.Results, CompatibilityResult{
		CounterpartParticipantID: provider.ParticipantID(),
		CounterpartVersion:       provider.Version,
		Deployable:               len(breaks) == 0,
	})
}

func (c *CompatibilityChecker) checkProvider(
	ctx context.Context,
	provider model.Resource,
	environment *model.Environment,
	report *CompatibilityReport,
) {
	consumers := c.repository.FindConsumersOfProviderAndEnvironment(ctx, provider, environment)

	for _, consumer := range consumers {
		consumerBreaks := checkResources(&consumer, &provider)
		for _, breakingChange := range consumerBreaks {
			report.Append(breakingChange)
		}

		report.Results = append(report.Results, CompatibilityResult{
			CounterpartParticipantID: consumer.ParticipantID(),
			CounterpartVersion:       consumer.Version,
			Deployable:               len(consumerBreaks) == 0,
		})
	}
}

func checkResources(leftResource *model.Resource, rightResource *model.Resource) []BreakingChange {
	switch leftResource.Kind {
	case model.RestRequest:
		return checkRequestResource(leftResource, rightResource)
	default:
		return checkResponseResource(leftResource, rightResource)
	}
}

func checkResponseResource(leftResource *model.Resource, rightResource *model.Resource) []BreakingChange {
	var breaks []BreakingChange

	for consumerPropertyPath, consumerProperty := range leftResource.Properties {
		providerProperty, propertyExists := rightResource.Properties[consumerPropertyPath]

		// If the property is not present in the provider and is required in the consumer, it is a breaking change.
		if !propertyExists && !consumerProperty.Optional {
			breaks = append(breaks, NewBreakingChange(
				leftResource,
				rightResource,
				ReasonMissingInProvider,
				consumerPropertyPath,
			))

			continue
		}

		// If the property is present in the provider and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, NewBreakingChange(
				leftResource,
				rightResource,
				ReasonTypeMismatch,
				consumerPropertyPath,
			))

			continue
		}

		// If the property is required in the consumer and is optional in the provider, it is a breaking change.
		if !consumerProperty.Optional && providerProperty.Optional {
			breaks = append(breaks, NewBreakingChange(
				leftResource,
				rightResource,
				ReasonOptionalInProviderRequiredInConsumer,
				consumerPropertyPath,
			))
		}
	}

	return breaks
}

func checkRequestResource(leftResource *model.Resource, rightResource *model.Resource) []BreakingChange {
	var breaks []BreakingChange

	for providerPropertyPath, providerProperty := range rightResource.Properties {
		consumerProperty, propertyExists := leftResource.Properties[providerPropertyPath]
		// If the property is not present in the consumer and is not optional, it is a breaking change.
		if !propertyExists && !providerProperty.Optional {
			breaks = append(breaks, NewBreakingChange(
				leftResource,
				rightResource,
				ReasonMissingInConsumer,
				providerPropertyPath,
			))

			continue
		}

		// If the property is present in the consumer and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, NewBreakingChange(
				leftResource,
				rightResource,
				ReasonTypeMismatch,
				providerPropertyPath,
			))

			continue
		}

		// If the property is required in the provider and is optional in the consumer, it is a breaking change.
		if !providerProperty.Optional && consumerProperty.Optional {
			breaks = append(breaks, NewBreakingChange(
				leftResource,
				rightResource,
				ReasonOptionalInConsumerRequiredInProvider,
				providerPropertyPath,
			))
		}
	}

	return breaks
}
