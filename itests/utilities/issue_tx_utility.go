package utl

import (
	"github.com/wavesplatform/gowaves/itests/config"
	i "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"time"
)

func GetAccount(suite *i.BaseSuite, i int) config.AccountInfo {
	return suite.Cfg.Accounts[i]
}

func GetAvalibleBalanceInWavesGo(suite *i.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAssetBalanceGo(suite *i.BaseSuite, address proto.WavesAddress, id []byte) int64 {
	return suite.Clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, id).GetAmount()
}

func GetTxIdsInBlockchain(suite *i.BaseSuite, ids map[string]*crypto.Digest, timeout time.Duration) map[string]string {
	time.Sleep(timeout)
	txIds := make(map[string]string)
	for name, id := range ids {
		_, _, errGo := suite.Clients.GoClients.HttpClient.TransactionInfoRaw(*id)
		_, _, errScala := suite.Clients.ScalaClients.HttpClient.TransactionInfoRaw(*id)
		if errGo == nil {
			txIds["Go "+name] = id.String()
		}
		if errScala == nil {
			txIds["Scala "+name] = id.String()
		}
	}
	return txIds
}

func SendAndWaitTransaction(suite *i.BaseSuite, tx *proto.IssueWithSig, timeout time.Duration) (error, error) {
	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.Conns = net.Reconnect(suite.T(), suite.Conns, suite.Ports)
	suite.Conns.SendToEachNode(suite.T(), &txMsg)

	errGo, errScala := suite.Clients.WaitForTransaction(tx.ID, timeout)
	return errGo, errScala
}
