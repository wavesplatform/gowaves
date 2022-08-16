package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/itests/node_client"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ItestSuite struct {
	suite.Suite

	mainCtx context.Context
	cancel  context.CancelFunc
	cfg     config.TestConfig
	docker  *d.Docker
	conns   net.NodeConnections
	clients *node_client.NodesClients
}

func (suite *ItestSuite) SetupSuite() {
	suite.mainCtx, suite.cancel = context.WithCancel(context.Background())

	paths, cfg, err := config.CreateFileConfigs()
	suite.NoError(err, "couldn't create config")
	suite.cfg = cfg

	docker, err := d.NewDocker()
	suite.NoError(err, "couldn't create docker pool")
	suite.docker = docker
	err = docker.RunContainers(suite.mainCtx, paths)
	if err != nil {
		docker.Finish(suite.cancel)
		suite.NoError(err, "couldn't run docker containers")
	}
	suite.conns = net.NewNodeConnections(suite.T())
	suite.clients = &node_client.NodesClients{
		GoClients:    node_client.NewNodeClient(suite.T(), d.ScalaNodeRESTApiPort, d.ScalaNodeGrpsApiPort),
		ScalaClients: node_client.NewNodeClient(suite.T(), d.GoNodeRESTApiPort, d.GoNodeGrpsApiPort),
	}
}

func (suite *ItestSuite) TearDownSuite() {
	lastHeight := suite.clients.ScalaClients.HttpClient.GetHeight(suite.T())
	newHeight := suite.clients.WaitForNewHeight(suite.T(), *lastHeight)
	suite.clients.StateHashCmp(suite.T(), newHeight)

	suite.docker.Finish(suite.cancel)
	suite.conns.Close()
}

func (suite *ItestSuite) SetupTest() {
	suite.clients.SendStartMessage(suite.T())
}

func (suite *ItestSuite) TearDownTest() {
	suite.clients.SendEndMessage(suite.T())
}

func (suite *ItestSuite) Test_SendTransaction() {
	a := proto.NewOptionalAssetWaves()
	ts := uint64(time.Now().UnixNano() / 1000000)
	tx := proto.NewUnsignedTransferWithSig(suite.cfg.Accounts[0].PublicKey, a, a, ts, 1000000000, 10000000,
		proto.NewRecipientFromAddress(suite.cfg.Accounts[1].Address), proto.Attachment{})
	err := tx.Sign('L', suite.cfg.Accounts[0].SecretKey)
	suite.NoError(err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.conns.SendToEachNode(suite.T(), &txMsg)

	suite.clients.WaitForTransaction(suite.T(), tx.ID, 1*time.Minute)
	b := suite.clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), suite.cfg.Accounts[1].Address)
	suite.Equal(suite.cfg.Accounts[1].Amount+1000000000, uint64(b.GetAvailable()))
}

func TestItestSuite(t *testing.T) {
	suite.Run(t, new(ItestSuite))
}
