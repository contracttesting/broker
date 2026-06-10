package can_i_deploy

import (
	"fmt"
	"os"
)

type CanIDeployErrorView struct {
	Participant  string
	Environment  string
	Counterparts []NotCompatibleCounterpart
}

type NotCompatibleCounterpart struct {
	Name      string
	Resources []NotCompatibleResource
}

type NotCompatibleResource struct {
	Method, Path, Location string
	Groups                 []SpecificBreakGroup
}

type SpecificBreakGroup struct {
	Label  string
	Breaks []string
}

func (r CanIDeployResponseBody) CanIDeployErrorView(participant, environment string) CanIDeployErrorView {
	fmt.Println("r", r)
	os.Exit(1)
	return CanIDeployErrorView{}
}
