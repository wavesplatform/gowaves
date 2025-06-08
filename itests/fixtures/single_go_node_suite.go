package fixtures

import (
	"context"
	"net"

	"github.com/stoewer/go-strcase"
	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/clients"
	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SingleGoNodeSuite struct {
	suite.Suite

	MainCtx context.Context
	Cancel  context.CancelFunc
	Cfg     config.TestConfig
	Docker  *d.Docker
	Client  *clients.NodeUniversalClient
}

func (suite *SingleGoNodeSuite) BaseSetup(options ...config.BlockchainOption) {
	suite.MainCtx, suite.Cancel = context.WithCancel(context.Background())
	suiteName := strcase.KebabCase(suite.T().Name())
	cfg, err := config.NewBlockchainConfig(options...)
	suite.Require().NoError(err, "couldn't create blockchain config")
	suite.Cfg = cfg.TestConfig()

	goConfigurator, err := config.NewGoConfigurator(suiteName, cfg)
	suite.Require().NoError(err, "couldn't create Go configurator")

	docker, err := d.NewDocker(suiteName)
	suite.Require().NoError(err, "couldn't create Docker pool")
	suite.Docker = docker

	if sErr := docker.StartGoNode(suite.MainCtx, goConfigurator); sErr != nil {
		docker.Finish(suite.Cancel)
		suite.Require().NoError(sErr, "couldn't start Go node container")
	}

	gp, err := proto.NewPeerInfoFromString(docker.GoNode().IP() + ":" + config.BindPort)
	suite.Require().NoError(err, "failed to create Go peer info")
	peers := []proto.PeerInfo{gp}
	addr := net.JoinHostPort(config.DefaultIP, docker.GoNode().Ports().BindPort)
	suite.Client = clients.NewNodeUniversalClient(suite.MainCtx, suite.T(), clients.NodeGo,
		docker.GoNode().Ports().RESTAPIPort, docker.GoNode().Ports().GRPCPort, addr, peers)
	suite.Client.Handshake()
}

func (suite *SingleGoNodeSuite) SetupSuite() {
	suite.BaseSetup()
}

func (suite *SingleGoNodeSuite) TearDownSuite() {
	suite.Client.Close(suite.T())
	suite.Docker.Finish(suite.Cancel)
}

func (suite *SingleGoNodeSuite) SetupTest() {
	suite.Client.SendStartMessage(suite.T())
}

func (suite *SingleGoNodeSuite) TearDownTest() {
	suite.Client.SendEndMessage(suite.T())
}
