package fixtures

import (
	"context"
	"fmt"
	"net"

	"github.com/stoewer/go-strcase"
	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/clients"
	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/itests/docker"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type MultiGoNodesSuite struct {
	suite.Suite
	MainCtx context.Context
	Cancel  context.CancelFunc
	Cfg     config.TestConfig
	Docker  *docker.Docker
	Clients []*clients.NodeUniversalClient
}

func (suite *MultiGoNodesSuite) BaseSetup(nodeCount int, options ...config.BlockchainOption) {
	suite.MainCtx, suite.Cancel = context.WithCancel(context.Background())
	suiteName := strcase.KebabCase(suite.T().Name())

	cfg, err := config.NewBlockchainConfig(options...)
	suite.Require().NoError(err, "couldn't create blockchain config")
	suite.Cfg = cfg.TestConfig()

	goConfigurators := make([]*config.GoConfigurator, 0, nodeCount)
	for i := 0; i < nodeCount; i++ {
		goCfg, err := config.NewGoConfigurator(fmt.Sprintf("%s-go-%d", suiteName, i), cfg)
		suite.Require().NoError(err, "couldn't create Go configurator for node %d", i)
		goConfigurators = append(goConfigurators, goCfg)
	}

	docker, err := d.NewDocker(suiteName)
	suite.Require().NoError(err, "couldn't create Docker pool")
	suite.Docker = docker

	dockerConfigurators := make([]config.DockerConfigurator, len(goConfigurators))
	for i, gc := range goConfigurators {
		dockerConfigurators[i] = gc
	}

	if sErr := docker.StartMultipleGoNodes(suite.MainCtx, dockerConfigurators...); sErr != nil {
		docker.Finish(suite.Cancel)
		suite.Require().NoError(sErr, "couldn't start Go nodes containers")
	}

	suite.Clients = make([]*clients.NodeUniversalClient, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node := docker.GoNodes()[i]

		peers := make([]proto.PeerInfo, 0, nodeCount-1)
		for j := 0; j < nodeCount; j++ {
			if j == i {
				continue
			}
			peerNode := docker.GoNodes()[j]
			peer, err := proto.NewPeerInfoFromString(peerNode.IP() + ":" + config.BindPort)
			suite.Require().NoError(err)
			peers = append(peers, peer)
		}

		addr := net.JoinHostPort(config.DefaultIP, node.Ports().BindPort)
		suite.Clients[i] = clients.NewNodeUniversalClient(
			suite.MainCtx,
			suite.T(),
			clients.NodeGo,
			node.Ports().RESTAPIPort,
			node.Ports().GRPCPort,
			addr,
			peers,
		)
	}

	for _, client := range suite.Clients {
		client.Handshake()
	}
}

func (suite *MultiGoNodesSuite) SetupSuite() {
	suite.BaseSetup(3)
}

func (suite *MultiGoNodesSuite) TearDownSuite() {
	clientsCount := len(suite.Clients)
	for i := 0; i < clientsCount; i++ {
		suite.Clients[i].Close(suite.T())
	}
	suite.Docker.FinishAllGoNodes(suite.Cancel, clientsCount)
}

func (suite *MultiGoNodesSuite) SetupTest() {
	for _, client := range suite.Clients {
		client.SendStartMessage(suite.T())
	}
}

func (suite *MultiGoNodesSuite) TearDownTest() {
	for _, client := range suite.Clients {
		client.SendEndMessage(suite.T())
	}
}
