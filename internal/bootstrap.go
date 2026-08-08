package internal

import (
	"github.com/contracttesting/broker/internal/components"
	"github.com/contracttesting/broker/internal/features/can_i_deploy"
	"github.com/contracttesting/broker/internal/features/create_environment"
	"github.com/contracttesting/broker/internal/features/create_participant"
	"github.com/contracttesting/broker/internal/features/publish_contract"
	"github.com/contracttesting/broker/internal/features/record_deployment"
	"github.com/contracttesting/broker/internal/features/rename_participant"
)

func Run() *components.Components {
	components := components.New()

	create_participant.Register(components)
	create_environment.Register(components)
	publish_contract.Register(components)
	can_i_deploy.Register(components)
	record_deployment.Register(components)
	rename_participant.Register(components)

	return components
}
