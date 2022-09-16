package utilities

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func GetAccount(suite *f.BaseSuite, i int) config.AccountInfo {
	return suite.Cfg.Accounts[i]
}

func GetAvalibleBalanceInWavesGo(suite *f.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAssetBalanceGo(suite *f.BaseSuite, address proto.WavesAddress, id []byte) int64 {
	return suite.Clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, id).GetAmount()
}

func GetTxIdsInBlockchain(suite *f.BaseSuite, ids map[string]*crypto.Digest, timeout time.Duration) map[string]string {
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

func SendAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction, scheme proto.Scheme, timeout time.Duration) (error, error) {
	bts, err := tx.MarshalBinary()
	require.NoError(suite.T(), err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}
	idBytes, err := tx.GetID(scheme)
	if err != nil {
		panic(fmt.Sprintf("failed to get txID: %v", err))
	}
	id, err := crypto.NewDigestFromBytes(idBytes)
	if err != nil {
		panic(fmt.Sprintf("failed to create new digest from bytes: %v", err))
	}

	suite.Conns.Reconnect(suite.T(), suite.Ports)
	suite.Conns.SendToEachNode(suite.T(), &txMsg)

	errGo, errScala := suite.Clients.WaitForTransaction(&id, timeout)
	return errGo, errScala
}
