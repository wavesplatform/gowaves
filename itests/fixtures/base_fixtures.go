package fixtures

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/stoewer/go-strcase"

	"github.com/wavesplatform/gowaves/itests/clients"
	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
)

type BaseSuite struct {
	suite.Suite

	MainCtx     context.Context
	Cancel      context.CancelFunc
	Cfg         config.TestConfig
	Docker      *d.Docker
	Clients     *clients.NodesClients
	SendToNodes []clients.Implementation
}

func (suite *BaseSuite) BaseSetup(options ...config.BlockchainOption) {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", options...)
}

func (suite *BaseSuite) BaseSetupWithImages(goRepository, goTag, scalaRepository, scalaTag string,
	options ...config.BlockchainOption) {
	suite.MainCtx, suite.Cancel = context.WithCancel(context.Background())
	suiteName := strcase.KebabCase(suite.T().Name())
	cfg, err := config.NewBlockchainConfig(options...)
	suite.Require().NoError(err, "couldn't create blockchain config")
	suite.Cfg = cfg.TestConfig()

	goConfigurator, err := config.NewGoConfigurator(suiteName, cfg)
	suite.Require().NoError(err, "couldn't create Go configurator")
	goConfigurator.WithImageRepository(goRepository).WithImageTag(goTag)

	scalaConfigurator, err := config.NewScalaConfigurator(suiteName, cfg)
	suite.Require().NoError(err, "couldn't create Scala configurator")
	scalaConfigurator.WithGoNode("go-node").WithImageRepository(scalaRepository).WithImageTag(scalaTag)

	docker, err := d.NewDocker(suiteName)
	suite.Require().NoError(err, "couldn't create Docker pool")
	suite.Docker = docker

	if sErr := docker.StartNodes(suite.MainCtx, goConfigurator, scalaConfigurator); sErr != nil {
		docker.Finish(suite.Cancel)
		suite.Require().NoError(sErr, "couldn't start nodes")
	}

	suite.Clients = clients.NewNodesClients(suite.MainCtx, suite.T(), docker.GoNode().IP(), docker.ScalaNode().IP(),
		docker.GoNode().Ports(), docker.ScalaNode().Ports())
	suite.Clients.Handshake()
	suite.SendToNodes = []clients.Implementation{clients.NodeGo}
}

func (suite *BaseSuite) WithGoImage(repository, tag string) *BaseSuite {
	suite.BaseSetupWithImages(repository, tag, "wavesplatform/wavesnode", "latest")
	return suite
}

func (suite *BaseSuite) WithScalaImage(repository, tag string) *BaseSuite {
	suite.BaseSetupWithImages("go-node", "latest", repository, tag)
	return suite
}

func (suite *BaseSuite) WithImages(goRepository, goTag, scalaRepository, scalaTag string) *BaseSuite {
	suite.BaseSetupWithImages(goRepository, goTag, scalaRepository, scalaTag)
	return suite
}

func (suite *BaseSuite) SetupSuite() {
	suite.BaseSetup()
}

func (suite *BaseSuite) TearDownSuite() {
	suite.Clients.WaitForStateHashEquality(suite.T())
	slog.Info("Closing clients")
	suite.Clients.Close(suite.T())
	slog.Info("Closing Docker")
	suite.Docker.Finish(suite.Cancel)
}

func (suite *BaseSuite) SetupTest() {
	const waitForConnectedPeersTimeout = 5 * time.Second
	err := suite.Clients.WaitForConnectedPeers(suite.MainCtx, waitForConnectedPeersTimeout)
	suite.Require().NoError(err, "no connected peers or an unexpected error occurred")
	fmt.Println(suite.Clients.GoClient.HTTPClient.GetHeight(suite.T()))
	fmt.Println(suite.Clients.ScalaClient.HTTPClient.GetHeight(suite.T()))
	// Miners Balances
	minerGoBalanceFromGo := suite.Clients.GoClient.HTTPClient.WavesBalance(suite.T(), suite.Cfg.Accounts[0].Address)
	minerScalaBalanceFromGo := suite.Clients.GoClient.HTTPClient.WavesBalance(suite.T(), suite.Cfg.Accounts[1].Address)
	minerGoBalanceFromScala := suite.Clients.ScalaClient.HTTPClient.WavesBalance(suite.T(), suite.Cfg.Accounts[0].Address)
	minerScalaBalanceFromScala := suite.Clients.ScalaClient.HTTPClient.WavesBalance(suite.T(), suite.Cfg.Accounts[1].Address)
	fmt.Println(fmt.Sprintf("Go miner balance at Height = %d: FromGoNode: %d, FromScala: %d",
		suite.Clients.GoClient.HTTPClient.GetHeight(suite.T()), minerGoBalanceFromGo, minerGoBalanceFromScala))
	fmt.Println(fmt.Sprintf("Scala miner balance at Height = %d: FromGoNode: %d, FromScalaNode: %d",
		suite.Clients.ScalaClient.HTTPClient.GetHeight(suite.T()), minerScalaBalanceFromGo, minerScalaBalanceFromScala))

	suite.Clients.WaitForHeight(suite.T(), 2) // Wait for nodes to start mining
	suite.Clients.SendStartMessage(suite.T())
}

func (suite *BaseSuite) TearDownTest() {
	suite.Clients.SendEndMessage(suite.T())
}

type BaseNegativeSuite struct {
	BaseSuite
}

func (suite *BaseNegativeSuite) SetupSuite() {
	suite.BaseSetup()
	suite.SendToNodes = append(suite.SendToNodes, clients.NodeScala)
}
