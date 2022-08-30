package integration_test

import (
	"context"
	"strings"
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
	enableScalaMining := true
	paths, cfg, err := config.CreateFileConfigs(enableScalaMining)
	suite.NoError(err, "couldn't create config")
	suite.cfg = cfg

	suiteName := strings.ToLower(suite.T().Name())
	docker, err := d.NewDocker(suiteName)
	suite.NoError(err, "couldn't create docker pool")
	suite.docker = docker

	ports, err := docker.RunContainers(suite.mainCtx, paths, suiteName)
	if err != nil {
		docker.Finish(suite.cancel)
		suite.NoError(err, "couldn't run docker containers")
	}

	suite.conns = net.NewNodeConnections(suite.T(), ports)
	suite.clients = node_client.NewNodesClients(suite.T(), ports)
}

func (suite *ItestSuite) TearDownSuite() {
	height := suite.clients.WaitForNewHeight(suite.T())
	suite.clients.StateHashCmp(suite.T(), height)

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
	tx := proto.NewUnsignedTransferWithSig(suite.cfg.Accounts[2].PublicKey, a, a, ts, 1000000000, 10000000,
		proto.NewRecipientFromAddress(suite.cfg.Accounts[3].Address), proto.Attachment{})
	err := tx.Sign('L', suite.cfg.Accounts[2].SecretKey)
	suite.NoError(err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.conns.SendToEachNode(suite.T(), &txMsg)

	suite.clients.WaitForTransaction(suite.T(), tx.ID, 1*time.Minute)
	b := suite.clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), suite.cfg.Accounts[3].Address)
	suite.Equal(suite.cfg.Accounts[3].Amount+1000000000, uint64(b.GetAvailable()))
}

func TestItest1Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ItestSuite))
}

func TestItest2Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ItestSuite))
}

func TestItest3Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ItestSuite))
}
