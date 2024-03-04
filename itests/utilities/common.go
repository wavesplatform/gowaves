package utilities

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/settings"

	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	DefaultMinerGo             = 0
	DefaultMinerScala          = 1
	DefaultSenderNotMiner      = 2
	DefaultRecipientNotMiner   = 3
	FirstRecipientNotMiner     = 4
	DAOAccount                 = 5
	XTNBuyBackAccount          = 6
	DefaultAccountForLoanFunds = 9
	MaxAmount                  = math.MaxInt64
	MinIssueFeeWaves           = 100000000
	MinSetAssetScriptFeeWaves  = 100000000
	MinTxFeeWaves              = 100000
	MinTxFeeWavesSmartAsset    = 500000
	MinDecimals                = 0
	MaxDecimals                = 8
	TestChainID                = 'L'
	CommonSymbolSet            = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789~!|#$%^&*()_+=\\\";:/?><|][{}"
	LettersAndDigits           = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	DefaultInitialTimeout      = 5 * time.Millisecond
	DefaultWaitTimeout         = 15 * time.Second
	DefaultTimeInterval        = 5 * time.Second
)

const (
	FeatureStatusActivated = "ACTIVATED"
	FeatureStatusApproved  = "APPROVED"
	FeatureStatusUndefined = "UNDEFINED"
)

var (
	cutCommentsRegex = regexp.MustCompile(`\s*#.*\n?`)
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

type AccountDiffBalancesSponsorshipSender struct {
	DiffBalanceWaves    BalanceInWaves
	DiffBalanceAsset    BalanceInAsset
	DiffBalanceFeeAsset BalanceInAsset
}

type AssetInfo struct {
	AssetInfoGo    *g.AssetInfoResponse
	AssetInfoScala *g.AssetInfoResponse
}

func NewBalanceInWaves(balanceGo, balanceScala int64) BalanceInWaves {
	return BalanceInWaves{
		BalanceInWavesGo:    balanceGo,
		BalanceInWavesScala: balanceScala,
	}
}

func NewBalanceInAsset(balanceGo, balanceScala int64) BalanceInAsset {
	return BalanceInAsset{
		BalanceInAssetGo:    balanceGo,
		BalanceInAssetScala: balanceScala,
	}
}

type AccountsDiffBalancesTxWithSponsorship struct {
	DiffBalancesSender    AccountDiffBalancesSponsorshipSender
	DiffBalancesRecipient AccountDiffBalances
	DiffBalancesSponsor   AccountDiffBalances
}

func NewDiffBalancesTxWithSponsorship(diffBalanceWavesGoSender, diffBalanceWavesScalaSender, diffBalanceAssetGoSender,
	diffBalanceAssetScalaSender, diffBalanceFeeAssetGoSender, diffBalanceFeeAssetScalaSender,
	diffBalanceWavesGoRecipient, diffBalanceWavesScalaRecipient, diffBalanceAssetGoRecipient,
	diffBalanceAssetScalaRecipient, diffBalanceWavesGoSponsor, diffBalanceWavesScalaSponsor, diffBalanceAssetGoSponsor,
	diffBalanceAssetScalaSponsor int64) AccountsDiffBalancesTxWithSponsorship {
	return AccountsDiffBalancesTxWithSponsorship{
		DiffBalancesSender: AccountDiffBalancesSponsorshipSender{
			DiffBalanceWaves: BalanceInWaves{
				BalanceInWavesGo:    diffBalanceWavesGoSender,
				BalanceInWavesScala: diffBalanceWavesScalaSender,
			},
			DiffBalanceAsset: BalanceInAsset{
				BalanceInAssetGo:    diffBalanceAssetGoSender,
				BalanceInAssetScala: diffBalanceAssetScalaSender,
			},
			DiffBalanceFeeAsset: BalanceInAsset{
				BalanceInAssetGo:    diffBalanceFeeAssetGoSender,
				BalanceInAssetScala: diffBalanceFeeAssetScalaSender,
			},
		},
		DiffBalancesRecipient: AccountDiffBalances{
			DiffBalanceWaves: BalanceInWaves{
				BalanceInWavesGo:    diffBalanceWavesGoRecipient,
				BalanceInWavesScala: diffBalanceWavesScalaRecipient,
			},
			DiffBalanceAsset: BalanceInAsset{
				BalanceInAssetGo:    diffBalanceAssetGoRecipient,
				BalanceInAssetScala: diffBalanceAssetScalaRecipient,
			},
		},
		DiffBalancesSponsor: AccountDiffBalances{
			DiffBalanceWaves: BalanceInWaves{
				BalanceInWavesGo:    diffBalanceWavesGoSponsor,
				BalanceInWavesScala: diffBalanceWavesScalaSponsor,
			},
			DiffBalanceAsset: BalanceInAsset{
				BalanceInAssetGo:    diffBalanceAssetGoSponsor,
				BalanceInAssetScala: diffBalanceAssetScalaSponsor,
			},
		},
	}
}

func NewConsideredTransaction(txId crypto.Digest, respGo, respScala *client.Response,
	errWtGo, errWtScala, errBrdCstGo, errBrdCstScala error) ConsideredTransaction {
	return ConsideredTransaction{
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

type AvailableVersions struct {
	Binary   []byte
	Protobuf []byte
	Sum      []byte
}

func NewAvailableVersions(binary []byte, protobuf []byte) AvailableVersions {
	sum := append(binary, protobuf...)
	return AvailableVersions{
		Binary:   binary,
		Protobuf: protobuf,
		Sum:      sum,
	}
}

func GetAvailableVersions(t *testing.T, txType proto.TransactionType, minVersion, maxVersion byte) AvailableVersions {
	var binary, protobuf []byte
	minPBVersion := proto.ProtobufTransactionsVersions[txType]
	require.GreaterOrEqual(t, minPBVersion, minVersion,
		"Min binary version greater then min protobuf version")
	for i := minVersion; i < minPBVersion; i++ {
		binary = append(binary, i)
	}
	for i := minPBVersion; i < maxVersion+1; i++ {
		protobuf = append(protobuf, i)
	}
	return NewAvailableVersions(binary, protobuf)
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

func GetScriptBytes(suite *f.BaseSuite, scriptStr string) []byte {
	script, err := base64.StdEncoding.DecodeString(scriptStr)
	require.NoError(suite.T(), err, "Failed to decode script string to byte array")
	return script
}

func GetCurrentTimestampInMs() uint64 {
	return uint64(time.Now().UnixMilli())
}

func GetTestcaseNameWithVersion(name string, v byte) string {
	return fmt.Sprintf("%s (v %d)", name, v)
}

func AssetWithVersion(assetID crypto.Digest, v int) string {
	return fmt.Sprintf(" asset %s (v %d)", assetID, v)
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

func MustGetAccountByAddress(suite *f.BaseSuite, address proto.WavesAddress) config.AccountInfo {
	for _, account := range suite.Cfg.Accounts {
		if account.Address.Equal(address) {
			return account
		}
	}
	require.FailNow(suite.T(), "Account with address %q wasn't found", address.String())
	panic("unreachable point reached")
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

func GetAssetInfo(suite *f.BaseSuite, assetID crypto.Digest) *client.AssetsDetail {
	assetInfo, err := suite.Clients.ScalaClients.HttpClient.GetAssetDetails(assetID)
	require.NoError(suite.T(), err, "Scala node: Can't get asset info")
	return assetInfo
}

func GetHeightGo(suite *f.BaseSuite) uint64 {
	return suite.Clients.GoClients.HttpClient.GetHeight(suite.T()).Height
}

func GetHeightScala(suite *f.BaseSuite) uint64 {
	return suite.Clients.ScalaClients.HttpClient.GetHeight(suite.T()).Height
}

func GetHeight(suite *f.BaseSuite) uint64 {
	goHeight := GetHeightGo(suite)
	scalaHeight := GetHeightScala(suite)
	if goHeight < scalaHeight {
		return goHeight
	}
	return scalaHeight
}

func WaitForHeight(suite *f.BaseSuite, height uint64) uint64 {
	return suite.Clients.WaitForHeight(suite.T(), height)
}

func WaitForNewHeight(suite *f.BaseSuite) uint64 {
	return suite.Clients.WaitForNewHeight(suite.T())
}

func GetActivationFeaturesStatusInfoGo(suite *f.BaseSuite, h uint64) *g.ActivationStatusResponse {
	return suite.Clients.GoClients.GrpcClient.GetFeatureActivationStatusInfo(suite.T(), int32(h))
}

func GetActivationFeaturesStatusInfoScala(suite *f.BaseSuite, h uint64) *g.ActivationStatusResponse {
	return suite.Clients.ScalaClients.GrpcClient.GetFeatureActivationStatusInfo(suite.T(), int32(h))
}

func getFeatureBlockchainStatus(statusResponse *g.ActivationStatusResponse, fID settings.Feature) (string, error) {
	var status string
	var err error
	for _, feature := range statusResponse.GetFeatures() {
		if feature.GetId() == int32(fID) {
			status = feature.GetBlockchainStatus().String()
			break
		}
	}
	if status == "" {
		err = errors.Errorf("Feature with ID %d not found", fID)
	}
	return status, err
}

func getFeatureActivationHeight(statusResponse *g.ActivationStatusResponse, featureID settings.Feature) (int32, error) {
	var err error
	var activationHeight int32
	activationHeight = -1
	for _, feature := range statusResponse.GetFeatures() {
		if feature.GetId() == int32(featureID) && feature.GetBlockchainStatus().String() == FeatureStatusActivated {
			activationHeight = feature.GetActivationHeight()
			break
		}
	}
	if activationHeight == -1 {
		err = errors.Errorf("Feature with Id %d not found", featureID)
	}
	return activationHeight, err
}

func GetFeatureBlockchainStatusGo(suite *f.BaseSuite, featureID settings.Feature, h uint64) string {
	status, err := getFeatureBlockchainStatus(GetActivationFeaturesStatusInfoGo(suite, h), featureID)
	require.NoError(suite.T(), err, "Couldn't get feature status info")
	suite.T().Logf("Go: Status of feature %d on height @%d: %s\n", featureID, h, status)
	return status
}

func GetFeatureBlockchainStatusScala(suite *f.BaseSuite, featureID settings.Feature, h uint64) string {
	status, err := getFeatureBlockchainStatus(GetActivationFeaturesStatusInfoScala(suite, h), featureID)
	require.NoError(suite.T(), err, "Couldn't get feature status info")
	suite.T().Logf("Scala: Status of feature %d on height @%d: %s\n", featureID, h, status)
	return status
}

func GetFeatureActivationHeightGo(suite *f.BaseSuite, featureID settings.Feature, height uint64) int32 {
	activationHeight, err := getFeatureActivationHeight(GetActivationFeaturesStatusInfoGo(suite, height), featureID)
	require.NoError(suite.T(), err)
	return activationHeight
}

func GetFeatureActivationHeightScala(suite *f.BaseSuite, featureID settings.Feature, height uint64) int32 {
	activationHeight, err := getFeatureActivationHeight(GetActivationFeaturesStatusInfoScala(suite, height), featureID)
	require.NoError(suite.T(), err)
	return activationHeight
}

func GetFeatureActivationHeight(suite *f.BaseSuite, featureID settings.Feature, height uint64) int32 {
	var err error
	var activationHeight int32
	activationHeight = -1
	activationHeightGo := GetFeatureActivationHeightGo(suite, featureID, height)
	activationHeightScala := GetFeatureActivationHeightScala(suite, featureID, height)
	if activationHeightGo == activationHeightScala && activationHeightGo > -1 {
		activationHeight = activationHeightGo
	} else {
		err = errors.New("Activation Height from Go and Scala is different")
	}
	require.NoError(suite.T(), err)
	return activationHeight
}

func GetFeatureBlockchainStatus(suite *f.BaseSuite, featureID settings.Feature, height uint64) (string, error) {
	var status string
	var err error
	statusGo := GetFeatureBlockchainStatusGo(suite, featureID, height)
	statusScala := GetFeatureBlockchainStatusScala(suite, featureID, height)
	if statusGo == statusScala {
		status = statusGo
	} else {
		err = errors.Errorf("Feature with Id %d has different statuses", featureID)
	}
	return status, err
}

func GetWaitingBlocks(suite *f.BaseSuite, height uint64, featureID settings.Feature) uint64 {
	var waitingBlocks uint64
	activationWindowSize := suite.Cfg.BlockchainSettings.ActivationWindowSize(height)
	votingPeriod := suite.Cfg.BlockchainSettings.FeaturesVotingPeriod
	status, err := GetFeatureBlockchainStatus(suite, featureID, height)
	require.NoError(suite.T(), err)
	switch status {
	case FeatureStatusActivated:
		waitingBlocks = 0
	case FeatureStatusApproved:
		waitingBlocks = activationWindowSize - (height - (height/votingPeriod)*votingPeriod)
	case FeatureStatusUndefined:
		if (votingPeriod == 1) && (height == 1) {
			waitingBlocks = 1 + 2*activationWindowSize - (height - (height/votingPeriod)*votingPeriod)
		} else {
			waitingBlocks = 2*activationWindowSize - (height - (height/votingPeriod)*votingPeriod)
		}
	default:
		suite.FailNow("Status is unknown")
	}
	return waitingBlocks
}

func WaitForFeatureActivation(suite *f.BaseSuite, featureID settings.Feature, height uint64) int32 {
	var activationHeight int32
	waitingBlocks := GetWaitingBlocks(suite, height, featureID)
	h := WaitForHeight(suite, height+waitingBlocks)
	activationHeightGo := GetFeatureActivationHeightGo(suite, featureID, h)
	activationHeightScala := GetFeatureActivationHeightScala(suite, featureID, h)
	if activationHeightScala == activationHeightGo {
		activationHeight = activationHeightGo
	} else {
		suite.FailNowf("Feature has different activation heights", "Feature ID is %d", featureID)
	}
	return activationHeight
}

func FeatureShouldBeActivated(suite *f.BaseSuite, featureID settings.Feature, height uint64) {
	activationHeight := WaitForFeatureActivation(suite, featureID, height)
	if activationHeight == -1 {
		suite.FailNowf("Feature is not activated", "Feature with Id %d", featureID)
	}
	suite.T().Logf("Feature %d is activated on height @%d\n", featureID, activationHeight)
}

func GetActivationOfFeatures(suite *f.BaseSuite, featureIDs ...settings.Feature) {
	h := GetHeight(suite)
	// features that should be activated
	for _, featureID := range featureIDs {
		FeatureShouldBeActivated(suite, featureID, h)
	}
}

func GetAssetInfoGrpcGo(suite *f.BaseSuite, assetID crypto.Digest) *g.AssetInfoResponse {
	return suite.Clients.GoClients.GrpcClient.GetAssetsInfo(suite.T(), assetID.Bytes())
}

func GetAssetInfoGrpcScala(suite *f.BaseSuite, assetID crypto.Digest) *g.AssetInfoResponse {
	return suite.Clients.ScalaClients.GrpcClient.GetAssetsInfo(suite.T(), assetID.Bytes())
}

func GetAssetInfoGrpc(suite *f.BaseSuite, assetID crypto.Digest) AssetInfo {
	return AssetInfo{GetAssetInfoGrpcGo(suite, assetID), GetAssetInfoGrpcScala(suite, assetID)}
}

func GetAssetBalanceGo(suite *f.BaseSuite, address proto.WavesAddress, assetID crypto.Digest) int64 {
	return suite.Clients.GoClients.GrpcClient.GetAssetBalance(suite.T(), address, assetID.Bytes()).GetAmount()
}

func GetAssetBalanceScala(suite *f.BaseSuite, address proto.WavesAddress, assetID crypto.Digest) int64 {
	return suite.Clients.ScalaClients.GrpcClient.GetAssetBalance(suite.T(), address, assetID.Bytes()).GetAmount()
}

func GetAssetBalance(suite *f.BaseSuite, address proto.WavesAddress, assetID crypto.Digest) (int64, int64) {
	return GetAssetBalanceGo(suite, address, assetID), GetAssetBalanceScala(suite, address, assetID)
}

func GetActualDiffBalanceInWaves(suite *f.BaseSuite, address proto.WavesAddress,
	initBalanceGo, initBalanceScala int64) BalanceInWaves {
	currentBalanceInWavesGo, currentBalanceInWavesScala := GetAvailableBalanceInWaves(suite, address)
	actualDiffBalanceInWavesGo := Abs(initBalanceGo - currentBalanceInWavesGo)
	actualDiffBalanceInWavesScala := Abs(initBalanceScala - currentBalanceInWavesScala)
	return NewBalanceInWaves(actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
}

func GetActualDiffBalanceInAssets(suite *f.BaseSuite, address proto.WavesAddress, assetID crypto.Digest,
	initBalanceGo, initBalanceScala int64) BalanceInAsset {
	currentBalanceInAssetGo, currentBalanceInAssetScala := GetAssetBalance(suite, address, assetID)
	actualDiffBalanceInAssetGo := Abs(currentBalanceInAssetGo - initBalanceGo)
	actualDiffBalanceInAssetScala := Abs(currentBalanceInAssetScala - initBalanceScala)
	return NewBalanceInAsset(actualDiffBalanceInAssetGo, actualDiffBalanceInAssetScala)
}

func GetTxIdsInBlockchain(suite *f.BaseSuite, ids map[string]*crypto.Digest) map[string]string {
	tick := time.Second
	timeout := DefaultWaitTimeout
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

func SendAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction, scheme proto.Scheme,
	waitForTx bool) ConsideredTransaction {
	timeout := DefaultInitialTimeout
	id := ExtractTxID(suite.T(), tx, scheme)
	txMsg := MarshalTxAndGetTxMsg(suite.T(), scheme, tx)
	if waitForTx {
		timeout = DefaultWaitTimeout
	}
	scala := !waitForTx

	connections, err := net.NewNodeConnections(suite.Ports)
	suite.Require().NoError(err, "failed to create new node connections")
	defer connections.Close(suite.T())

	connections.SendToNodes(suite.T(), txMsg, scala)
	suite.T().Log("Tx msg was successfully send to nodes")

	suite.T().Log("Waiting for Tx appears in Blockchain")
	errGo, errScala := suite.Clients.WaitForTransaction(id, timeout)
	if errGo != nil {
		suite.T().Log(errors.Errorf("Errors after waiting: %s", errGo))
	} else {
		txInfoRawGo, respGo, goRqErr := suite.Clients.GoClients.HttpClient.TransactionInfoRaw(id)
		if goRqErr != nil {
			suite.T().Logf("Error on requesting Tx Info Go: %v", goRqErr)
		} else {
			suite.T().Logf("Tx Info Go after waiting: %s, Response Go: %s",
				GetTransactionJsonOrErrMsg(txInfoRawGo), respGo.Status)
		}
	}
	if errScala != nil {
		suite.T().Log(errors.Errorf("Errors after waiting: %s", errScala))
	} else {
		txInfoRawScala, respScala, scalaRqErr := suite.Clients.ScalaClients.HttpClient.TransactionInfoRaw(id)
		if scalaRqErr != nil {
			suite.T().Logf("Error on requesting Tx Info Scala: %v", scalaRqErr)
		} else {
			suite.T().Logf("Tx Info Scala after waiting: %s, Response Scala: %s",
				GetTransactionJsonOrErrMsg(txInfoRawScala), respScala.Status)
		}
	}
	return NewConsideredTransaction(id, nil, nil, errGo, errScala, nil, nil)
}

func BroadcastAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction,
	scheme proto.Scheme, waitForTx bool) ConsideredTransaction {
	timeout := DefaultWaitTimeout
	id := ExtractTxID(suite.T(), tx, scheme)
	respGo, errBrdCstGo := suite.Clients.GoClients.HttpClient.TransactionBroadcast(tx)
	var respScala *client.Response = nil
	var errBrdCstScala error = nil
	if !waitForTx {
		timeout = DefaultInitialTimeout
		respScala, errBrdCstScala = suite.Clients.ScalaClients.HttpClient.TransactionBroadcast(tx)
	}
	suite.T().Log("Tx msg was successfully Broadcast to nodes")

	suite.T().Log("Waiting for Tx appears in Blockchain")
	errWtGo, errWtScala := suite.Clients.WaitForTransaction(id, timeout)
	if errWtGo != nil {
		suite.T().Log(errors.Errorf("Errors after waiting: %s", errWtGo))
	} else {
		txInfoRawGo, responseGo, goRqErr := suite.Clients.GoClients.HttpClient.TransactionInfoRaw(id)
		if goRqErr != nil {
			suite.T().Logf("Error on requesting Tx Info Go: %v", goRqErr)
		} else {
			suite.T().Logf("Tx Info Go after waiting: %s, Response Go: %s",
				GetTransactionJsonOrErrMsg(txInfoRawGo), responseGo.Status)
		}
	}
	if errWtScala != nil {
		suite.T().Log(errors.Errorf("Errors after waiting: %s", errWtScala))
	} else {
		txInfoRawScala, responseScala, scalaRqErr := suite.Clients.ScalaClients.HttpClient.TransactionInfoRaw(id)
		if scalaRqErr != nil {
			suite.T().Logf("Error on requesting Tx Info Scals: %v", scalaRqErr)
		} else {
			suite.T().Logf("Tx Info Scala after waiting: %s, Response Scala: %s",
				GetTransactionJsonOrErrMsg(txInfoRawScala), responseScala.Status)
		}
	}
	return NewConsideredTransaction(id, respGo, respScala, errWtGo, errWtScala, errBrdCstGo, errBrdCstScala)
}

func getItestsDir() (string, error) {
	filename, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(filename), "itests"), nil
}

func ReadScript(scriptDir, fileName string) ([]byte, error) {
	dir, err := getItestsDir()
	if err != nil {
		return nil, err
	}
	scriptPath := filepath.Join(dir, "testdata", "scripts", scriptDir, fileName)
	scriptFileContent, err := os.ReadFile(filepath.Clean(scriptPath))
	if err != nil {
		return nil, err
	}
	scriptBase64WithComments := string(scriptFileContent)
	scriptBase64WithoutComments := cutCommentsRegex.ReplaceAllString(scriptBase64WithComments, "")
	scriptBase64 := strings.TrimSpace(scriptBase64WithoutComments)

	return base64.StdEncoding.DecodeString(scriptBase64)
}

type RewardDiffBalancesInWaves struct {
	MinersSumDiffBalance  BalanceInWaves
	DAODiffBalance        BalanceInWaves
	XTNBuyBackDiffBalance BalanceInWaves
}

func NewRewardDiffBalances(diffBalanceGoMiners, diffBalanceScalaMiners, diffBalanceGoDao, diffBalanceScalaDao,
	diffBalanceGoXtn, diffBalanceScalaXtn int64) RewardDiffBalancesInWaves {
	return RewardDiffBalancesInWaves{
		MinersSumDiffBalance: BalanceInWaves{
			BalanceInWavesGo:    diffBalanceGoMiners,
			BalanceInWavesScala: diffBalanceScalaMiners,
		},
		DAODiffBalance: BalanceInWaves{
			BalanceInWavesGo:    diffBalanceGoDao,
			BalanceInWavesScala: diffBalanceScalaDao,
		},
		XTNBuyBackDiffBalance: BalanceInWaves{
			BalanceInWavesGo:    diffBalanceGoXtn,
			BalanceInWavesScala: diffBalanceScalaXtn,
		},
	}
}

func GetBlockGo(suite *f.BaseSuite, height uint64) *waves.Block {
	return suite.Clients.GoClients.GrpcClient.GetBlock(suite.T(), height).GetBlock()
}

func GetBlockScala(suite *f.BaseSuite, height uint64) *waves.Block {
	return suite.Clients.ScalaClients.GrpcClient.GetBlock(suite.T(), height).GetBlock()
}

func GetDesiredRewardGo(suite *f.BaseSuite, height uint64) int64 {
	block := GetBlockGo(suite, height)
	return block.GetHeader().RewardVote
}

func GetDesiredRewardScala(suite *f.BaseSuite, height uint64) int64 {
	block := GetBlockScala(suite, height)
	return block.GetHeader().RewardVote
}

func GetDesiredReward(suite *f.BaseSuite, height uint64) int64 {
	var desiredR int64
	desiredRGo := GetDesiredRewardGo(suite, height)
	desiredRScala := GetDesiredRewardScala(suite, height)
	if desiredRGo == desiredRScala {
		desiredR = desiredRGo
	} else {
		suite.FailNow("Desired Reward from Go and Scala is different")
	}
	return desiredR
}

// GetRewardTermCfg is max period of voting (term).
func GetRewardTermCfg(suite *f.BaseSuite) uint64 {
	return suite.Cfg.BlockchainSettings.BlockRewardTerm
}

// GetRewardTermAfter20Cfg returns term after feature 20 activation (term-after-capped-reward-feature), =1/2 term.
func GetRewardTermAfter20Cfg(suite *f.BaseSuite) uint64 {
	return suite.Cfg.BlockchainSettings.BlockRewardTermAfter20
}

// GetRewards get response from /blockchain/rewards.
func GetRewardsGo(suite *f.BaseSuite) *client.RewardInfo {
	return suite.Clients.GoClients.HttpClient.Rewards(suite.T())
}

func GetRewardsScala(suite *f.BaseSuite) *client.RewardInfo {
	return suite.Clients.ScalaClients.HttpClient.Rewards(suite.T())
}

func GetRewards(suite *f.BaseSuite) (*client.RewardInfo, *client.RewardInfo) {
	return GetRewardsGo(suite), GetRewardsScala(suite)
}

// GetRewards get response from /blockchain/rewards/{height}.
func GetRewardsAtHeightGo(suite *f.BaseSuite, height uint64) *client.RewardInfo {
	return suite.Clients.GoClients.HttpClient.RewardsAtHeight(suite.T(), height)
}

func GetRewardsAtHeightScala(suite *f.BaseSuite, height uint64) *client.RewardInfo {
	return suite.Clients.ScalaClients.HttpClient.RewardsAtHeight(suite.T(), height)
}

func GetRewardsAtHeight(suite *f.BaseSuite, height uint64) (*client.RewardInfo, *client.RewardInfo) {
	return GetRewardsAtHeightGo(suite, height), GetRewardsAtHeightScala(suite, height)
}

func GetCurrentRewardGo(suite *f.BaseSuite, height uint64) uint64 {
	return suite.Clients.GoClients.HttpClient.RewardsAtHeight(suite.T(), height).CurrentReward
}

func GetCurrentRewardScala(suite *f.BaseSuite, height uint64) uint64 {
	return suite.Clients.ScalaClients.HttpClient.RewardsAtHeight(suite.T(), height).CurrentReward
}

func GetCurrentReward(suite *f.BaseSuite, height uint64) uint64 {
	var currentReward uint64
	currentRewardGo := GetCurrentRewardGo(suite, height)
	currentRewardScala := GetCurrentRewardScala(suite, height)
	if currentRewardGo == currentRewardScala {
		currentReward = currentRewardGo
	} else {
		suite.FailNow("Current reward is different")
	}
	return currentReward
}

func GetRewardTermAtHeightGo(suite *f.BaseSuite, height uint64) uint64 {
	return GetRewardsAtHeightGo(suite, height).Term
}

func GetRewardTermAtHeightScala(suite *f.BaseSuite, height uint64) uint64 {
	return GetRewardsAtHeightScala(suite, height).Term
}

func GetRewardTermAtHeight(suite *f.BaseSuite, height uint64) RewardTerm {
	termGo := GetRewardTermAtHeightGo(suite, height)
	termScala := GetRewardTermAtHeightScala(suite, height)
	suite.T().Logf("Go: Reward Term: %d, Scala: Reward Term: %d, height: %d",
		termGo, termScala, height)
	return NewRewardTerm(termGo, termScala)
}

type RewardTerm struct {
	TermGo    uint64
	TermScala uint64
}

func NewRewardTerm(termGo, termScala uint64) RewardTerm {
	return RewardTerm{
		TermGo:    termGo,
		TermScala: termScala,
	}
}

func GetXtnBuybackPeriodCfg(suite *f.BaseSuite) uint64 {
	return suite.Cfg.BlockchainSettings.MinXTNBuyBackPeriod
}

func GetRollbackToHeightGo(suite *f.BaseSuite, height uint64, returnTxToUtx bool) *proto.BlockID {
	return suite.Clients.GoClients.HttpClient.RollbackToHeight(suite.T(), height, returnTxToUtx)
}

func GetRollbackToHeightScala(suite *f.BaseSuite, height uint64, returnTxToUtx bool) *proto.BlockID {
	return suite.Clients.ScalaClients.HttpClient.RollbackToHeight(suite.T(), height, returnTxToUtx)
}

func GetRollbackToHeight(suite *f.BaseSuite, height uint64, returnTxToUtx bool) (*proto.BlockID, *proto.BlockID) {
	suite.T().Logf("Rollback to height: %d from height: %d", height, GetHeight(suite))
	return GetRollbackToHeightGo(suite, height, returnTxToUtx), GetRollbackToHeightScala(suite, height, returnTxToUtx)
}
