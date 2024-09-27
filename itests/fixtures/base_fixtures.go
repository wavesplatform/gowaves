package fixtures

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/stoewer/go-strcase"

	"github.com/wavesplatform/gowaves/itests/clients"
	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
)

type BaseSuite struct {
	suite.Suite

	MainCtx context.Context
	Cancel  context.CancelFunc
	Cfg     config.TestConfig
	Docker  *d.Docker
	Clients *clients.NodesClients
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

	suite.Clients = clients.NewNodesClients(suite.T(), docker.GoNode().Ports(), docker.ScalaNode().Ports())
}

func (suite *BaseSuite) SetupSuite() {
	suite.BaseSetup(config.WithScalaMining())
}

func (suite *BaseSuite) TearDownSuite() {
	suite.Clients.WaitForStateHashEquality(suite.T())
	suite.Docker.Finish(suite.Cancel)
}

func (suite *BaseSuite) SetupTest() {
	const waitForConnectedPeersTimeout = 5 * time.Second
	err := suite.Clients.WaitForConnectedPeers(suite.MainCtx, waitForConnectedPeersTimeout)
	suite.Require().NoError(err, "no connected peers or an unexpected error occurred")
	suite.Clients.WaitForHeight(suite.T(), 2) // Wait for nodes to start mining
	suite.Clients.SendStartMessage(suite.T())
}

func (suite *BaseSuite) TearDownTest() {
	suite.Clients.SendEndMessage(suite.T())
}
