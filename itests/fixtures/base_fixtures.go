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
}

func (suite *BaseSuite) BaseSetup(options ...config.BlockchainOption) {
	suite.MainCtx, suite.Cancel = context.WithCancel(context.Background())
	suiteName := strcase.KebabCase(suite.T().Name())
	cfg, err := config.NewBlockchainConfig(options...)
	suite.Require().NoError(err, "couldn't create blockchain config")
	suite.Cfg = cfg.TestConfig()

	goConfigurator, err := config.NewGoConfigurator(suiteName, cfg)
	suite.Require().NoError(err, "couldn't create Go configurator")
	scalaConfigurator, err := config.NewScalaConfigurator(suiteName, cfg)
	suite.Require().NoError(err, "couldn't create Scala configurator")

	docker, err := d.NewDocker(suiteName)
	suite.Require().NoError(err, "couldn't create Docker pool")
	suite.Docker = docker

	if gsErr := docker.StartGoNode(suite.MainCtx, goConfigurator); gsErr != nil {
		docker.Finish(suite.Cancel)
		suite.Require().NoError(gsErr, "couldn't start Go node container")
	}
	scalaConfigurator.WithGoNode(docker.GoNode().ContainerNetworkIP())
	if ssErr := docker.StartScalaNode(suite.MainCtx, scalaConfigurator); ssErr != nil {
		docker.Finish(suite.Cancel)
		suite.Require().NoError(ssErr, "couldn't start Scala node container")
	}

	suite.Clients = node_client.NewNodesClients(suite.T(), docker.GoNode().Ports(), docker.ScalaNode().Ports())
}

func (suite *BaseSuite) SetupSuite() {
	suite.BaseSetup(config.WithScalaMining())
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
