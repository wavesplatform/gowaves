package utilities

import (
	"github.com/wavesplatform/gowaves/itests/config"
	integration "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"time"
)

func GetCurrentTimestampInMs() uint64 {
	return uint64(time.Now().UnixNano() / 1000000)
}

func GetAccount(suite *integration.BaseSuite, i int) config.AccountInfo {
	return suite.Cfg.Accounts[i]
}

func GetAvalibleBalanceInWaves(suite *integration.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAssetBalance(suite *integration.BaseSuite, address proto.WavesAddress, id []byte) int64 {
	return suite.Clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, id).GetAmount()
}

func GetInvalidTxIdsInBlockchain(suite *integration.BaseSuite, ids []*crypto.Digest, timeout time.Duration) []*crypto.Digest {
	time.Sleep(timeout)
	for _, id := range ids {
		_, _, errGo := suite.Clients.GoClients.HttpClient.TransactionInfoRaw(*id)
		_, _, errScala := suite.Clients.ScalaClients.HttpClient.TransactionInfoRaw(*id)
		if (errGo != nil) && (errScala != nil) {
			ids[0] = nil
			if len(ids) > 1 {
				copy(ids[0:], ids[1:])
				ids = ids[:len(ids)-1]
			} else if len(ids) == 1 {
				ids = nil
			}
		}
	}
	return ids
}
