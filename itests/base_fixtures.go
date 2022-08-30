package integration

import (
	"context"
	"strings"

	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/itests/node_client"
)

type BaseSuite struct {
	suite.Suite

	MainCtx context.Context
	Cancel  context.CancelFunc
	Cfg     config.TestConfig
	Docker  *d.Docker
	Conns   net.NodeConnections
	Clients *node_client.NodesClients
}

func (suite *BaseSuite) SetupSuite() {
	suite.MainCtx, suite.Cancel = context.WithCancel(context.Background())
	enableScalaMining := true
	paths, cfg, err := config.CreateFileConfigs(enableScalaMining)
	suite.NoError(err, "couldn't create config")
	suite.Cfg = cfg

	suiteName := strings.ToLower(suite.T().Name())
	docker, err := d.NewDocker(suiteName)
	suite.NoError(err, "couldn't create Docker pool")
	suite.Docker = docker

	ports, err := docker.RunContainers(suite.MainCtx, paths, suiteName)
	if err != nil {
		docker.Finish(suite.Cancel)
		suite.NoError(err, "couldn't run Docker containers")
	}

	suite.Conns = net.NewNodeConnections(suite.T(), ports)
	suite.Clients = node_client.NewNodesClients(suite.T(), ports)
}

func (suite *BaseSuite) TearDownSuite() {
	height := suite.Clients.WaitForNewHeight(suite.T())
	suite.Clients.StateHashCmp(suite.T(), height)

	suite.Docker.Finish(suite.Cancel)
	suite.Conns.Close()
}

func (suite *BaseSuite) SetupTest() {
	suite.Clients.SendStartMessage(suite.T())
}

func (suite *BaseSuite) TearDownTest() {
	suite.Clients.SendEndMessage(suite.T())
}
