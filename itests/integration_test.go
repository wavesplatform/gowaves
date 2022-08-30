package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SendTxSuite struct {
	BaseSuite
}

func (suite *SendTxSuite) Test_SendTransaction() {
	a := proto.NewOptionalAssetWaves()
	ts := uint64(time.Now().UnixNano() / 1000000)
	tx := proto.NewUnsignedTransferWithSig(suite.Cfg.Accounts[2].PublicKey, a, a, ts, 1000000000, 10000000,
		proto.NewRecipientFromAddress(suite.Cfg.Accounts[3].Address), proto.Attachment{})
	err := tx.Sign('L', suite.Cfg.Accounts[2].SecretKey)
	suite.NoError(err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.Conns.SendToEachNode(suite.T(), &txMsg)

	suite.Clients.WaitForTransaction(suite.T(), tx.ID, 1*time.Minute)
	b := suite.Clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), suite.Cfg.Accounts[3].Address)
	suite.Equal(suite.Cfg.Accounts[3].Amount+1000000000, uint64(b.GetAvailable()))
}

func TestItest1Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SendTxSuite))
}

func TestItest2Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SendTxSuite))
}

func TestItest3Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SendTxSuite))
}
