package fixtures

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/stoewer/go-strcase"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/itests/node_client"
)

type BaseSuite struct {
	suite.Suite

	MainCtx context.Context
	Cancel  context.CancelFunc
	Cfg     config.TestConfig
	Docker  *d.Docker
	Clients *node_client.NodesClients
	Ports   *d.Ports
}

func (suite *BaseSuite) SetupSuite() {
	const enableScalaMining = true

	suiteName := strcase.KebabCase(suite.T().Name())

	suite.MainCtx, suite.Cancel = context.WithCancel(context.Background())
	paths, cfg, err := config.CreateFileConfigs(suiteName, enableScalaMining)
	suite.Require().NoError(err, "couldn't create config")
	suite.Cfg = cfg

	docker, err := d.NewDocker(suiteName)
	suite.Require().NoError(err, "couldn't create Docker pool")
	suite.Docker = docker

	ports, err := docker.RunContainers(suite.MainCtx, paths, suiteName)
	if err != nil {
		docker.Finish(suite.Cancel)
		suite.Require().NoError(err, "couldn't run Docker containers")
	}
	suite.Ports = ports
	suite.Clients = node_client.NewNodesClients(suite.T(), ports)
}

func (suite *BaseSuite) TearDownSuite() {
	suite.Clients.WaitForStateHashEquality(suite.T())
	suite.Docker.Finish(suite.Cancel)
}

func (suite *BaseSuite) SetupTest() {
	errGo, errScala := suite.Clients.WaitForConnectedPeers(5 * time.Second)
	suite.Require().NoError(errGo, "Go: no connected peers")
	suite.Require().NoError(errScala, "Scala: no connected peers")
	suite.Clients.WaitForHeight(suite.T(), 2) // Wait for nodes to start mining
	suite.Clients.SendStartMessage(suite.T())
}

func (suite *BaseSuite) TearDownTest() {
	suite.Clients.SendEndMessage(suite.T())
}
