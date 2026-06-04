package compatibility_checker

import (
	"context"
	"errors"
	"strings"

	"github.com/contracttesting/broker/internal/model"
	"github.com/contracttesting/broker/internal/repository"
)

type BreakingReason string

const (
	ReasonProviderResourceNotFound                 BreakingReason = "provider_resource_not_found"
	ReasonMissingInProvider                        BreakingReason = "missing_in_provider"
	ReasonMissingInConsumer                        BreakingReason = "missing_in_consumer"
	ReasonTypeMismatch                             BreakingReason = "type_mismatch"
	ReasonOptionalInProviderRequiredInConsumer     BreakingReason = "optional_in_provider_required_in_consumer"
	ReasonOptionalInConsumerRequiredInProvider     BreakingReason = "optional_in_consumer_required_in_provider"
	ReasonProviderResourceNotDeployedInEnvironment BreakingReason = "provider_resource_not_deployed_in_environment"
)

const (
	detailKeyProperty             = "property"
	detailKeyConsumerType         = "consumer_type"
	detailKeyProviderType         = "provider_type"
	detailKeyDeployedEnvironments = "deployed_environments"
)

type BreakingChange struct {
	CheckedResource     *model.Resource   `json:"checked_resource"`
	CounterpartResource *model.Resource   `json:"counterpart_resource,omitempty"`
	Reason              BreakingReason    `json:"reason"`
	Details             map[string]string `json:"details,omitempty"`
}

func (b *BreakingChange) consumerResource() *model.Resource {
	if b.CheckedResource.IsConsumer() {
		return b.CheckedResource
	}

	return b.CounterpartResource
}

func (b *BreakingChange) ConsumerName() string {
	return b.consumerResource().ParticipantName()
}

func (b *BreakingChange) ProviderName() string {
	return b.consumerResource().ConsumedProvider
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

	r.Breaks[b.ConsumerName()] = append(r.Breaks[b.ConsumerName()], b)
}

func NewBreakingChange(
	checked *model.Resource,
	counterpart *model.Resource,
	reason BreakingReason,
	property string,
) BreakingChange {
	consumer, provider := checked, counterpart
	if !checked.IsConsumer() {
		consumer, provider = counterpart, checked
	}

	var details map[string]string
	if property != "" {
		details = map[string]string{detailKeyProperty: property}

		if reason == ReasonTypeMismatch {
			details[detailKeyConsumerType] = consumer.Properties[property].Type
			details[detailKeyProviderType] = provider.Properties[property].Type
		}
	}

	return BreakingChange{
		CheckedResource:     checked,
		CounterpartResource: counterpart,
		Reason:              reason,
		Details:             details,
	}
}

func NewProviderNotDeployedBreakingChange(consumer *model.Resource, deployedEnvironments []string) BreakingChange {
	var details map[string]string
	if len(deployedEnvironments) > 0 {
		details = map[string]string{
			detailKeyDeployedEnvironments: strings.Join(deployedEnvironments, ", "),
		}
	}

	return BreakingChange{
		CheckedResource: consumer,
		Reason:          ReasonProviderResourceNotDeployedInEnvironment,
		Details:         details,
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
		consumerBreaks := checkResources(&provider, &consumer)
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

func checkResources(checked *model.Resource, counterpart *model.Resource) []BreakingChange {
	consumer, provider := checked, counterpart
	if !checked.IsConsumer() {
		consumer, provider = counterpart, checked
	}

	switch consumer.Kind {
	case model.RestRequest:
		return checkRequestResource(checked, counterpart, consumer, provider)
	default:
		return checkResponseResource(checked, counterpart, consumer, provider)
	}
}

func checkResponseResource(
	checked *model.Resource,
	counterpart *model.Resource,
	consumer *model.Resource,
	provider *model.Resource,
) []BreakingChange {
	var breaks []BreakingChange

	for consumerPropertyPath, consumerProperty := range consumer.Properties {
		providerProperty, propertyExists := provider.Properties[consumerPropertyPath]

		// If the property is not present in the provider and is required in the consumer, it is a breaking change.
		if !propertyExists && !consumerProperty.Optional {
			breaks = append(breaks, NewBreakingChange(
				checked,
				counterpart,
				ReasonMissingInProvider,
				consumerPropertyPath,
			))

			continue
		}

		// If the property is present in the provider and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, NewBreakingChange(
				checked,
				counterpart,
				ReasonTypeMismatch,
				consumerPropertyPath,
			))

			continue
		}

		// If the property is required in the consumer and is optional in the provider, it is a breaking change.
		if !consumerProperty.Optional && providerProperty.Optional {
			breaks = append(breaks, NewBreakingChange(
				checked,
				counterpart,
				ReasonOptionalInProviderRequiredInConsumer,
				consumerPropertyPath,
			))
		}
	}

	return breaks
}

func checkRequestResource(
	checked *model.Resource,
	counterpart *model.Resource,
	consumer *model.Resource,
	provider *model.Resource,
) []BreakingChange {
	var breaks []BreakingChange

	for providerPropertyPath, providerProperty := range provider.Properties {
		consumerProperty, propertyExists := consumer.Properties[providerPropertyPath]
		// If the property is not present in the consumer and is not optional, it is a breaking change.
		if !propertyExists && !providerProperty.Optional {
			breaks = append(breaks, NewBreakingChange(
				checked,
				counterpart,
				ReasonMissingInConsumer,
				providerPropertyPath,
			))

			continue
		}

		// If the property is present in the consumer and the type is different, it is a breaking change.
		if consumerProperty.Type != providerProperty.Type {
			breaks = append(breaks, NewBreakingChange(
				checked,
				counterpart,
				ReasonTypeMismatch,
				providerPropertyPath,
			))

			continue
		}

		// If the property is required in the provider and is optional in the consumer, it is a breaking change.
		if !providerProperty.Optional && consumerProperty.Optional {
			breaks = append(breaks, NewBreakingChange(
				checked,
				counterpart,
				ReasonOptionalInConsumerRequiredInProvider,
				providerPropertyPath,
			))
		}
	}

	return breaks
}
