package fixtures

import (
	"context"
	"strings"
	"time"

	"github.com/stretchr/testify/suite"

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

	suite.MainCtx, suite.Cancel = context.WithCancel(context.Background())
	paths, cfg, err := config.CreateFileConfigs(enableScalaMining)
	suite.Require().NoError(err, "couldn't create config")
	suite.Cfg = cfg

	suiteName := strings.ToLower(suite.T().Name())
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
	height := suite.Clients.WaitForNewHeight(suite.T())
	suite.Clients.StateHashCmp(suite.T(), height)

	suite.Docker.Finish(suite.Cancel)
}

func (suite *BaseSuite) SetupTest() {
	errGo, errScala := suite.Clients.WaitForConnectedPeers(suite.T(), 5*time.Second)
	suite.Require().NoError(errGo, "Go: no connected peers")
	suite.Require().NoError(errScala, "Scala: no connected peers")
	suite.Clients.WaitForHeight(suite.T(), 2) // Wait for nodes to start mining
	suite.Clients.SendStartMessage(suite.T())
}

func (suite *BaseSuite) TearDownTest() {
	suite.Clients.SendEndMessage(suite.T())
}
