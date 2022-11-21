package utilities

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	CommonSymbolSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789~!|#$%^&*()_+=\\\";:/?><|][{}"
)

type BroadcastedTransaction struct {
	TxID             crypto.Digest
	ResponseGo       *client.Response
	ErrorBrdCstGo    error
	ResponseScala    *client.Response
	ErrorBrdCstScala error
}

type Response struct {
	ResponseGo    *client.Response
	ResponseScala *client.Response
}

type BalanceInWaves struct {
	BalanceInWavesGo    int64
	BalanceInWavesScala int64
}

type BalanceInAsset struct {
	BalanceInAssetGo    int64
	BalanceInAssetScala int64
}

type WaitingError struct {
	ErrWtGo    error
	ErrWtScala error
}

type BroadcastingError struct {
	ErrorBrdCstGo    error
	ErrorBrdCstScala error
}

type ConsideredTransaction struct {
	TxID      crypto.Digest
	WtErr     WaitingError
	Resp      Response
	BrdCstErr BroadcastingError
}

func NewBalanceInWaves(balanceGo, balanceScala int64) *BalanceInWaves {
	return &BalanceInWaves{
		BalanceInWavesGo:    balanceGo,
		BalanceInWavesScala: balanceScala,
	}
}

func NewConsideredTransaction(txId crypto.Digest, respGo, respScala *client.Response,
	errWtGo, errWtScala, errBrdCstGo, errBrdCstScala error) *ConsideredTransaction {
	return &ConsideredTransaction{
		TxID: txId,
		Resp: Response{
			ResponseGo:    respGo,
			ResponseScala: respScala,
		},
		WtErr: WaitingError{
			ErrWtGo:    errWtGo,
			ErrWtScala: errWtScala,
		},
		BrdCstErr: BroadcastingError{
			ErrorBrdCstGo:    errBrdCstGo,
			ErrorBrdCstScala: errBrdCstScala,
		},
	}
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

func RandStringBytes(n int, symbolSet string) string {
	b := make([]byte, n)
	for j := range b {
		b[j] = symbolSet[rand.Intn(len(symbolSet))]
	}
	return string(b)
}

func GetTransactionJsonOrErrMsg(tx proto.Transaction) string {
	var result string
	jsonStr, err := json.Marshal(tx)
	if err != nil {
		result = fmt.Sprintf("Failed to create tx JSON: %s", err)
	} else {
		result = string(jsonStr)
	}
	return result
}

func GetCurrentTimestampInMs() uint64 {
	return uint64(time.Now().UnixMilli())
}

func GetAccount(suite *f.BaseSuite, i int) config.AccountInfo {
	return suite.Cfg.Accounts[i]
}

func GetAddressByAliasGo(suite *f.BaseSuite, alias string) []byte {
	return suite.Clients.GoClients.GrpcClient.GetAddressByAlias(suite.T(), alias)
}

func GetAddressByAliasScala(suite *f.BaseSuite, alias string) []byte {
	return suite.Clients.ScalaClients.GrpcClient.GetAddressByAlias(suite.T(), alias)
}

func GetAddressesByAlias(suite *f.BaseSuite, alias string) ([]byte, []byte) {
	return GetAddressByAliasGo(suite, alias), GetAddressByAliasScala(suite, alias)
}

func GetAvailableBalanceInWavesGo(suite *f.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAvailableBalanceInWavesScala(suite *f.BaseSuite, address proto.WavesAddress) int64 {
	//TODO(ipereiaslavskaia) return suite.Clients.ScalaClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable() after fixing grpc interface
	wavesBalance := suite.Clients.ScalaClients.HttpClient.WavesBalance(suite.T(), address)
	return int64(wavesBalance.Balance)
}

func GetAvailableBalanceInWaves(suite *f.BaseSuite, address proto.WavesAddress) (int64, int64) {
	return GetAvailableBalanceInWavesGo(suite, address), GetAvailableBalanceInWavesScala(suite, address)
}

func GetAssetBalanceGo(suite *f.BaseSuite, address proto.WavesAddress, assetId crypto.Digest) int64 {
	return suite.Clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, assetId.Bytes()).GetAmount()
}

func GetAssetBalanceScala(suite *f.BaseSuite, address proto.WavesAddress, assetId crypto.Digest) int64 {
	//TODO (ipereiaslavskaia) return suite.Clients.ScalaClients.GrpcClient.GetAssetBalance(suite.T(), address, assetId.Bytes()).GetAmount() after fixing grpc interface
	assetBalance := suite.Clients.ScalaClients.HttpClient.AssetBalance(suite.T(), address, assetId)
	return int64(assetBalance.Balance)
}

func GetAssetBalance(suite *f.BaseSuite, address proto.WavesAddress, assetId crypto.Digest) (int64, int64) {
	return GetAssetBalanceGo(suite, address, assetId), GetAssetBalanceScala(suite, address, assetId)
}

func GetActualDiffBalanceInWaves(suite *f.BaseSuite, address proto.WavesAddress, initBalanceGo, initBalanceScala int64) (int64, int64) {
	currentBalanceInWavesGo, currentBalanceInWavesScala := GetAvailableBalanceInWaves(suite, address)
	actualDiffBalanceInWavesGo := initBalanceGo - currentBalanceInWavesGo
	actualDiffBalanceInWavesScala := initBalanceScala - currentBalanceInWavesScala
	return actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala
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

func ExtractTxID(t *testing.T, tx proto.Transaction, scheme proto.Scheme) crypto.Digest {
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

func SendAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction, scheme proto.Scheme, timeout time.Duration) ConsideredTransaction {
	bts := marshalTransaction(suite.T(), tx)
	suite.T().Logf("CreateAlias transaction bts: %s", base64.StdEncoding.EncodeToString(bts))
	id := ExtractTxID(suite.T(), tx, scheme)
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.Conns.Reconnect(suite.T(), suite.Ports)
	suite.Conns.SendToEachNode(suite.T(), &txMsg)

	errGo, errScala := suite.Clients.WaitForTransaction(id, timeout)
	return *NewConsideredTransaction(id, nil, nil, errGo, errScala, nil, nil)
}

/*func BroadcastAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction, scheme proto.Scheme, timeout time.Duration) ConsideredTransaction {
	id := ExtractTxID(suite.T(), tx, scheme)
	respGo, errBrdCstGo := suite.Clients.GoClients.HttpClient.TransactionBroadcast(tx)
	respScala, errBrdCstScala := suite.Clients.ScalaClients.HttpClient.TransactionBroadcast(tx)
	fmt.Println(id.String())
	errWtGo, errWtScala := suite.Clients.WaitForTransaction(id, timeout)

	return *NewConsideredTransaction(id, respGo, respScala, errWtGo, errWtScala, errBrdCstGo, errBrdCstScala)
}*/

func BroadcastAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction, scheme proto.Scheme, timeout time.Duration) (BroadcastedTransaction, error, error) {
	id := ExtractTxID(suite.T(), tx, scheme)
	respGo, errBrdCstGo := suite.Clients.GoClients.HttpClient.TransactionBroadcast(tx)
	respScala, errBrdCstScala := suite.Clients.ScalaClients.HttpClient.TransactionBroadcast(tx)
	errWtGo, errWtScala := suite.Clients.WaitForTransaction(id, timeout)

	return *NewBroadcastedTransaction(id, respGo, errBrdCstGo, respScala, errBrdCstScala), errWtGo, errWtScala
}
