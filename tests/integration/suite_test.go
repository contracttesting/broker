package integration_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/h2non/gock.v1"

	upload_contract "github.com/contracttesting/cli/internal/features/upload_contract"
)

const BrokerURL = "http://broker.test"

type Suite struct {
	suite.Suite
	Client *upload_contract.Client
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) SetupTest() {
	s.Client = upload_contract.NewClient(BrokerURL)
	gock.InterceptClient(s.Client.HTTPClient())
}

func (s *Suite) TearDownTest() {
	gock.RestoreClient(s.Client.HTTPClient())
	gock.Off()
}
