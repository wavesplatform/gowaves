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
	"github.com/wavesplatform/gowaves/itests/utils"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ItestSuite struct {
	suite.Suite

	mainCtx context.Context
	cancel  context.CancelFunc
	cfg     config.TestConfig
	docker  *d.Docker

	ctx context.Context
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
}

func (suite *ItestSuite) TearDownSuite() {
	suite.docker.Finish(suite.cancel)
}

func (suite *ItestSuite) SetupTest() {
	suite.ctx = context.Background()
	utils.SendStartMessage(suite.T(), suite.ctx)
}

func (suite *ItestSuite) TearDownTest() {
	utils.SendEndMessage(suite.T(), suite.ctx)
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

	heightBefore := node_client.GoNodeClient(suite.T()).GetHeight(suite.T(), suite.ctx)

	connections := net.NewNodeConnections(suite.T())
	defer connections.Close()
	connections.SendToEachNode(suite.T(), &txMsg)

	newHeight := utils.WaitForNewHeight(suite.T(), suite.ctx, *heightBefore)
	utils.StateHashCmp(suite.T(), suite.ctx, newHeight)
}

func TestItestSuite(t *testing.T) {
	suite.Run(t, new(ItestSuite))
}
