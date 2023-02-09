package utilities

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	DefaultSenderNotMiner      = 2
	DefaultRecipientNotMiner   = 3
	DefaultAccountForLoanFunds = 9
	MaxAmount                  = math.MaxInt64
	MinIssueFeeWaves           = 100000000
	MinTxFeeWaves              = 100000
	TestChainID                = 'L'
	CommonSymbolSet            = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789~!|#$%^&*()_+=\\\";:/?><|][{}"
	LettersAndDigits           = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

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

type AccountDiffBalances struct {
	DiffBalanceWaves BalanceInWaves
	DiffBalanceAsset BalanceInAsset
}

func NewBalanceInWaves(balanceGo, balanceScala int64) *BalanceInWaves {
	return &BalanceInWaves{
		BalanceInWavesGo:    balanceGo,
		BalanceInWavesScala: balanceScala,
	}
}

func NewBalanceInAsset(balanceGo, balanceScala int64) *BalanceInAsset {
	return &BalanceInAsset{
		BalanceInAssetGo:    balanceGo,
		BalanceInAssetScala: balanceScala,
	}
}

func NewDiffBalances(diffBalanceWavesGo, diffBalanceWavesScala, diffBalanceAssetGo, diffBalanceAssetScala int64) *AccountDiffBalances {
	return &AccountDiffBalances{
		DiffBalanceWaves: BalanceInWaves{
			BalanceInWavesGo:    diffBalanceWavesGo,
			BalanceInWavesScala: diffBalanceWavesScala,
		},
		DiffBalanceAsset: BalanceInAsset{
			BalanceInAssetGo:    diffBalanceAssetGo,
			BalanceInAssetScala: diffBalanceAssetScala,
		},
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

func GetVersions() []byte {
	return []byte{1, 2, 3}
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

func RandDigest(t *testing.T, n int, symbolSet string) crypto.Digest {
	id, err := crypto.NewDigestFromBytes([]byte(RandStringBytes(n, symbolSet)))
	require.NoError(t, err, "Failed to create random Digest")
	return id
}

func GetCurrentTimestampInMs() uint64 {
	return uint64(time.Now().UnixMilli())
}

func GetTestcaseNameWithVersion(name string, v byte) string {
	return fmt.Sprintf("%s (version %d)", name, v)
}

// Abs returns the absolute value of x.
func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func SetFromToAccounts(accountNumbers ...int) (int, int, error) {
	var from, to int
	switch len(accountNumbers) {
	case 0:
		from = 2
		to = 3
	case 1:
		from = accountNumbers[0]
		to = 3
	case 2:
		from = accountNumbers[0]
		to = accountNumbers[1]
	default:
		return 0, 0, errors.New("More than two parameters")
	}
	return from, to, nil
}

// AddNewAccount function creates and adds new AccountInfo to suite accounts list. Returns index of new account in
// the list and AccountInfo.
func AddNewAccount(suite *f.BaseSuite, chainID proto.Scheme) (int, config.AccountInfo) {
	seed := strconv.FormatInt(time.Now().UnixNano(), 10)
	sk, pk, err := crypto.GenerateKeyPair([]byte(seed))
	require.NoError(suite.T(), err)
	addr, err := proto.NewAddressFromPublicKey(chainID, pk)
	require.NoError(suite.T(), err)
	acc := config.AccountInfo{
		PublicKey: pk,
		SecretKey: sk,
		Amount:    0,
		Address:   addr,
	}
	suite.Cfg.Accounts = append(suite.Cfg.Accounts, acc)
	return len(suite.Cfg.Accounts) - 1, acc
}

func GetAccount(suite *f.BaseSuite, i int) config.AccountInfo {
	if i < 0 || i > len(suite.Cfg.Accounts)-1 {
		require.FailNow(suite.T(),
			fmt.Sprintf("Invalid account index %d, should be between 0 and %d", i, len(suite.Cfg.Accounts)))
	}
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

func GetAddressWithNewSchema(suite *f.BaseSuite, chainId proto.Scheme, address proto.WavesAddress) proto.WavesAddress {
	newAddr, err := proto.RebuildAddress(chainId, address.Body())
	require.NoError(suite.T(), err, "Can't rebuild address")
	return newAddr
}

func GetAddressFromRecipient(suite *f.BaseSuite, recipient proto.Recipient) proto.WavesAddress {
	if addr := recipient.Address(); addr != nil {
		return *addr
	}
	alias := recipient.Alias()
	require.NotNil(suite.T(), alias, "Address and Alias shouldn't be nil at the same time")
	address, err := proto.NewAddressFromBytes(GetAddressByAliasGo(suite, alias.Alias))
	require.NoError(suite.T(), err, "Can't get address from bytes")
	return address
}

func GetAvailableBalanceInWavesGo(suite *f.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.GoClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAvailableBalanceInWavesScala(suite *f.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.ScalaClients.GrpcClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAvailableBalanceInWaves(suite *f.BaseSuite, address proto.WavesAddress) (int64, int64) {
	return GetAvailableBalanceInWavesGo(suite, address), GetAvailableBalanceInWavesScala(suite, address)
}

func GetAssetBalanceGo(suite *f.BaseSuite, address proto.WavesAddress, assetId crypto.Digest) int64 {
	return suite.Clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, assetId.Bytes()).GetAmount()
}

func GetAssetBalanceScala(suite *f.BaseSuite, address proto.WavesAddress, assetId crypto.Digest) int64 {
	return suite.Clients.ScalaClients.GrpcClient.GetAssetBalance(suite.T(), address, assetId.Bytes()).GetAmount()
}

func GetAssetBalance(suite *f.BaseSuite, address proto.WavesAddress, assetId crypto.Digest) (int64, int64) {
	return GetAssetBalanceGo(suite, address, assetId), GetAssetBalanceScala(suite, address, assetId)
}

func GetActualDiffBalanceInWaves(suite *f.BaseSuite, address proto.WavesAddress, initBalanceGo, initBalanceScala int64) (int64, int64) {
	currentBalanceInWavesGo, currentBalanceInWavesScala := GetAvailableBalanceInWaves(suite, address)
	actualDiffBalanceInWavesGo := Abs(initBalanceGo - currentBalanceInWavesGo)
	actualDiffBalanceInWavesScala := Abs(initBalanceScala - currentBalanceInWavesScala)
	return actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala
}

func GetActualDiffBalanceInAssets(suite *f.BaseSuite, address proto.WavesAddress, assetId crypto.Digest, initBalanceGo, initBalanceScala int64) (int64, int64) {
	currentBalanceInAssetGo, currentBalanceInAssetScala := GetAssetBalance(suite, address, assetId)
	actualDiffBalanceInAssetGo := Abs(currentBalanceInAssetGo - initBalanceGo)
	actualDiffBalanceInAssetScala := Abs(currentBalanceInAssetScala - initBalanceScala)
	return actualDiffBalanceInAssetGo, actualDiffBalanceInAssetScala
}

func GetTxIdsInBlockchain(suite *f.BaseSuite, ids map[string]*crypto.Digest) map[string]string {
	timeout := 20 * time.Second
	tick := timeout
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

func MarshalTxAndGetTxMsg(t *testing.T, scheme proto.Scheme, tx proto.Transaction) proto.Message {
	bts, err := proto.MarshalTx(scheme, tx)
	require.NoError(t, err, "failed to marshal tx")
	t.Logf("Transaction bytes: %s", base64.StdEncoding.EncodeToString(bts))
	if proto.IsProtobufTx(tx) {
		return &proto.PBTransactionMessage{Transaction: bts}
	} else {
		return &proto.TransactionMessage{Transaction: bts}
	}

}

func SendAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction, scheme proto.Scheme, waitForTx bool) ConsideredTransaction {
	timeout := 5 * time.Millisecond
	id := ExtractTxID(suite.T(), tx, scheme)
	txMsg := MarshalTxAndGetTxMsg(suite.T(), scheme, tx)
	if waitForTx {
		timeout = 15 * time.Second
	}
	scala := !waitForTx

	connections, err := net.NewNodeConnections(suite.Ports)
	suite.Require().NoError(err, "failed to create new node connections")
	defer connections.Close(suite.T())
	connections.SendToNodes(suite.T(), txMsg, scala)

	errGo, errScala := suite.Clients.WaitForTransaction(id, timeout)
	return *NewConsideredTransaction(id, nil, nil, errGo, errScala, nil, nil)
}

func BroadcastAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction, scheme proto.Scheme, waitForTx bool) ConsideredTransaction {
	timeout := 15 * time.Second
	id := ExtractTxID(suite.T(), tx, scheme)
	respGo, errBrdCstGo := suite.Clients.GoClients.HttpClient.TransactionBroadcast(tx)
	var respScala *client.Response = nil
	var errBrdCstScala error = nil
	if !waitForTx {
		timeout = time.Millisecond
		respScala, errBrdCstScala = suite.Clients.ScalaClients.HttpClient.TransactionBroadcast(tx)
	}
	errWtGo, errWtScala := suite.Clients.WaitForTransaction(id, timeout)
	return *NewConsideredTransaction(id, respGo, respScala, errWtGo, errWtScala, errBrdCstGo, errBrdCstScala)
}
