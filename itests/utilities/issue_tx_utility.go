package utl

import (
	"github.com/wavesplatform/gowaves/itests/config"
	integration "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"time"
)

func GetAccount(suite *integration.BaseSuite, i int) config.AccountInfo {
	return suite.Cfg.Accounts[i]
}

func GetAvalibleBalanceInWaves(suite *integration.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAssetBalance(suite *integration.BaseSuite, address proto.WavesAddress, id []byte) int64 {
	return suite.Clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, id).GetAmount()
}

func GetHeightGo(suite *integration.BaseSuite) *client.BlocksHeight {
	return suite.Clients.GoClients.GrpcClient.GetHeight(suite.T())
}

func GetHeightScala(suite *integration.BaseSuite) *client.BlocksHeight {
	return suite.Clients.ScalaClients.GrpcClient.GetHeight(suite.T())
}

func GetInvalidTxIdsInBlockchain(suite *integration.BaseSuite, ids map[string]*crypto.Digest, timeout time.Duration) map[string]string {
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
