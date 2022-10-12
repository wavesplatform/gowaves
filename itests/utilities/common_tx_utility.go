package utilities

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789~!|#$%^&*()_+=\\\";:/?><|][{}"
)

type BroadcastedTransaction struct {
	TxID             crypto.Digest
	ResponseGo       *client.Response
	ErrorBrdCstGo    error
	ResponseScala    *client.Response
	ErrorBrdCstScala error
}

func NewBroadcastedTransaction(txId crypto.Digest, responseGo *client.Response, errBrdCstGo error,
	responseScala *client.Response, errBrdCstScala error) *BroadcastedTransaction {
	return &BroadcastedTransaction{
		TxID:             txId,
		ResponseGo:       responseGo,
		ErrorBrdCstGo:    errBrdCstGo,
		ResponseScala:    responseScala,
		ErrorBrdCstScala: errBrdCstScala,
	}
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for j := range b {
		b[j] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func GetCurrentTimestampInMs() uint64 {
	return uint64(time.Now().UnixMilli())
}

func GetAccount(suite *f.BaseSuite, i int) config.AccountInfo {
	return suite.Cfg.Accounts[i]
}

func GetAvailableBalanceInWavesGo(suite *f.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAvailableBalanceInWavesScala(suite *f.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.ScalaClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAvailableBalanceInWaves(suite *f.BaseSuite, address proto.WavesAddress) (int64, int64) {
	wavesBalanceGo := GetAvailableBalanceInWavesGo(suite, address)
	wavesBalanceScala := GetAvailableBalanceInWavesScala(suite, address)
	return wavesBalanceGo, wavesBalanceScala
}

func GetAssetBalanceGo(suite *f.BaseSuite, address proto.WavesAddress, id []byte) int64 {
	return suite.Clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, id).GetAmount()
}

func GetAssetBalanceScala(suite *f.BaseSuite, address proto.WavesAddress, id []byte) int64 {
	return suite.Clients.ScalaClients.GrpcClient.GetAssetBalance(suite.T(), address, id).GetAmount()
}

func GetAssetBalance(suite *f.BaseSuite, address proto.WavesAddress, id []byte) (int64, int64) {
	assetBalanceGo := GetAssetBalanceGo(suite, address, id)
	assetBalanceScala := GetAssetBalanceScala(suite, address, id)
	return assetBalanceGo, assetBalanceScala
}

func GetTxIdsInBlockchain(suite *f.BaseSuite, ids map[string]*crypto.Digest,
	timeout, tick time.Duration) map[string]string {
	var (
		ticker      = time.NewTicker(tick)
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		txIDs       = make(map[string]string, 2*len(ids))
	)
	defer func() {
		ticker.Stop()
		cancel()
	}()
	for {
		if len(txIDs) == 2*len(ids) { // fast path
			return txIDs
		}
		select {
		case <-ctx.Done():
			return txIDs
		case <-ticker.C:
			for name, id := range ids {
				goTxID := "Go " + name
				if _, ok := txIDs[goTxID]; !ok {
					_, _, errGo := suite.Clients.GoClients.HttpClient.TransactionInfoRaw(*id)
					if errGo == nil {
						txIDs[goTxID] = id.String()
					}
				}
				scalaTxID := "Scala " + name
				if _, ok := txIDs[scalaTxID]; !ok {
					_, _, errScala := suite.Clients.ScalaClients.HttpClient.TransactionInfoRaw(*id)
					if errScala == nil {
						txIDs[scalaTxID] = id.String()
					}
				}
			}
		}
	}
}

func extractTxID(t *testing.T, tx proto.Transaction, scheme proto.Scheme) crypto.Digest {
	idBytes, err := tx.GetID(scheme)
	require.NoError(t, err, "failed to get txID")
	id, err := crypto.NewDigestFromBytes(idBytes)
	require.NoError(t, err, "failed to create new digest from bytes")
	return id
}

func marshalTransaction(t *testing.T, tx proto.Transaction) []byte {
	bts, err := tx.MarshalBinary()
	require.NoError(t, err, "failed to marshal tx")
	return bts
}

func SendAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction, scheme proto.Scheme,
	timeout time.Duration) (error, error) {
	bts := marshalTransaction(suite.T(), tx)
	id := extractTxID(suite.T(), tx, scheme)
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.Conns.Reconnect(suite.T(), suite.Ports)
	suite.Conns.SendToEachNode(suite.T(), &txMsg)

	errGo, errScala := suite.Clients.WaitForTransaction(id, timeout)
	return errGo, errScala
}

func BroadcastAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction, scheme proto.Scheme, timeout time.Duration) (
	BroadcastedTransaction, error, error) {
	id := extractTxID(suite.T(), tx, scheme)

	respGo, errBrdCstGo := suite.Clients.GoClients.HttpClient.TransactionBroadcast(tx)
	respScala, errBrdCstScala := suite.Clients.ScalaClients.HttpClient.TransactionBroadcast(tx)
	errWtGo, errWtScala := suite.Clients.WaitForTransaction(id, timeout)

	return *NewBroadcastedTransaction(id, respGo, errBrdCstGo, respScala, errBrdCstScala), errWtGo, errWtScala
}
