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
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/settings"

	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
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
	MaxDecimals                = 8
	TestChainID                = 'L'
	CommonSymbolSet            = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789~!|#$%^&*()_+=\\\";:/?><|][{}"
	LettersAndDigits           = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	DefaultInitialTimeout      = 5 * time.Millisecond
	DefaultWaitTimeout         = 15 * time.Second
	DefaultTimeInterval        = 5 * time.Second

	// DefaultSponsorshipActivationHeight sets the height at which Fee Sponsorship takes effect.
	// Although the feature itself is activated at height 1 by default, it takes 2 additional voting periods (2 blocks)
	// for it to become effective.
	DefaultSponsorshipActivationHeight = 3
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

func SafeInt64ToUint64(x int64) uint64 {
	if x < 0 {
		panic("negative number")
	}
	return uint64(x)
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
	return suite.Clients.GoClient.GRPCClient.GetAddressByAlias(suite.T(), alias)
}

func GetAddressByAliasScala(suite *f.BaseSuite, alias string) []byte {
	return suite.Clients.ScalaClient.GRPCClient.GetAddressByAlias(suite.T(), alias)
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
	return suite.Clients.GoClient.GRPCClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAvailableBalanceInWavesScala(suite *f.BaseSuite, address proto.WavesAddress) int64 {
	return suite.Clients.ScalaClient.GRPCClient.GetWavesBalance(suite.T(), address).GetAvailable()
}

func GetAvailableBalanceInWaves(suite *f.BaseSuite, address proto.WavesAddress) (int64, int64) {
	return GetAvailableBalanceInWavesGo(suite, address), GetAvailableBalanceInWavesScala(suite, address)
}

func GetAssetInfo(suite *f.BaseSuite, assetID crypto.Digest) *client.AssetsDetail {
	assetInfo, err := suite.Clients.ScalaClient.HTTPClient.GetAssetDetails(assetID)
	require.NoError(suite.T(), err, "Scala node: Can't get asset info")
	return assetInfo
}

func GetHeightGo(suite *f.BaseSuite) uint64 {
	return suite.Clients.GoClient.HTTPClient.GetHeight(suite.T()).Height
}

func GetHeightScala(suite *f.BaseSuite) uint64 {
	return suite.Clients.ScalaClient.HTTPClient.GetHeight(suite.T()).Height
}

func GetHeight(suite *f.BaseSuite) uint64 {
	goCh := make(chan uint64)
	scalaCh := make(chan uint64)
	go func() {
		goCh <- GetHeightGo(suite)
	}()
	go func() {
		scalaCh <- GetHeightScala(suite)
	}()
	goHeight := <-goCh
	scalaHeight := <-scalaCh
	if goHeight < scalaHeight {
		return goHeight
	}
	return scalaHeight
}

func WaitForHeight(suite *f.BaseSuite, height uint64, opts ...config.WaitOption) uint64 {
	opts = append(opts, config.WaitWithContext(suite.MainCtx))
	return suite.Clients.WaitForHeight(suite.T(), height, opts...)
}

func WaitForNewHeight(suite *f.BaseSuite) uint64 {
	return suite.Clients.WaitForNewHeight(suite.T())
}

func GetActivationFeaturesStatusInfoGo(suite *f.BaseSuite, h uint64) *g.ActivationStatusResponse {
	if h > math.MaxInt32 {
		panic("Height is too big node")
	}
	return suite.Clients.GoClient.GRPCClient.GetFeatureActivationStatusInfo(suite.T(), int32(h))
}

func GetActivationFeaturesStatusInfoScala(suite *f.BaseSuite, h uint64) *g.ActivationStatusResponse {
	if h > math.MaxInt32 {
		panic("Height is too big node")
	}
	return suite.Clients.ScalaClient.GRPCClient.GetFeatureActivationStatusInfo(suite.T(), int32(h))
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

func getFeatureActivationHeight(
	statusResponse *g.ActivationStatusResponse, featureID settings.Feature,
) (proto.Height, bool) {
	for _, feature := range statusResponse.GetFeatures() {
		if feature.GetId() == int32(featureID) && feature.GetBlockchainStatus().String() == FeatureStatusActivated {
			if h := feature.GetActivationHeight(); h >= 0 {
				return uint64(h), true
			}
			panic("Activation height is negative what is possible only on Scala node. " +
				"Do not use this feature of Scala node!")
		}
	}
	return 0, false
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

func GetFeatureActivationHeightGo(suite *f.BaseSuite, featureID settings.Feature, height uint64) proto.Height {
	activationHeight, ok := getFeatureActivationHeight(GetActivationFeaturesStatusInfoGo(suite, height), featureID)
	require.True(suite.T(), ok, "Feature is not activated on the Go node")
	return activationHeight
}

func GetFeatureActivationHeightScala(suite *f.BaseSuite, featureID settings.Feature, height uint64) proto.Height {
	activationHeight, ok := getFeatureActivationHeight(GetActivationFeaturesStatusInfoScala(suite, height), featureID)
	require.True(suite.T(), ok, "Feature is not activated on the Scala node")
	return activationHeight
}

func GetFeatureActivationHeight(suite *f.BaseSuite, featureID settings.Feature, height uint64) proto.Height {
	goCh := make(chan proto.Height)
	scalaCh := make(chan proto.Height)
	go func() {
		goCh <- GetFeatureActivationHeightGo(suite, featureID, height)
	}()
	go func() {
		scalaCh <- GetFeatureActivationHeightScala(suite, featureID, height)
	}()
	activationHeightGo := <-goCh
	activationHeightScala := <-scalaCh

	if activationHeightGo == activationHeightScala && activationHeightGo > 0 {
		return activationHeightGo
	}

	suite.FailNow("Activation Height from Go and Scala is different")
	return 0
}

func GetFeatureBlockchainStatus(suite *f.BaseSuite, featureID settings.Feature, height uint64) (string, error) {
	goCh := make(chan string)
	scalaCh := make(chan string)
	go func() {
		goCh <- GetFeatureBlockchainStatusGo(suite, featureID, height)
	}()
	go func() {
		scalaCh <- GetFeatureBlockchainStatusScala(suite, featureID, height)
	}()
	statusGo := <-goCh
	statusScala := <-scalaCh

	if statusGo == statusScala {
		return statusGo, nil
	}
	return "", errors.Errorf("Feature with ID %d has different statuses", featureID)
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

func WaitForFeatureActivation(suite *f.BaseSuite, featureID settings.Feature, height uint64) proto.Height {
	waitingBlocks := GetWaitingBlocks(suite, height, featureID)
	h := WaitForHeight(suite, height+waitingBlocks, config.WaitWithTimeoutInBlocks(waitingBlocks))

	goCh := make(chan proto.Height)
	scalaCh := make(chan proto.Height)

	go func() {
		goCh <- GetFeatureActivationHeightGo(suite, featureID, h)
		close(goCh)
	}()
	go func() {
		scalaCh <- GetFeatureActivationHeightScala(suite, featureID, h)
		close(scalaCh)
	}()
	activationHeightGo, ok := <-goCh
	if !ok {
		suite.FailNowf("Failed to get activation height from Go node", "Feature ID is %d", featureID)
	}
	activationHeightScala, ok := <-scalaCh
	if !ok {
		suite.FailNowf("Failed to get activation height from Scala node", "Feature ID is %d", featureID)
	}

	if activationHeightScala == activationHeightGo {
		return activationHeightGo
	}
	suite.FailNowf("Feature has different activation heights", "Feature ID is %d", featureID)
	return 0
}

func FeatureShouldBeActivated(suite *f.BaseSuite, featureID settings.Feature, height uint64) {
	activationHeight := WaitForFeatureActivation(suite, featureID, height)
	if activationHeight == 0 {
		suite.FailNowf("Feature is not activated", "Feature with ID %d", featureID)
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
	return suite.Clients.GoClient.GRPCClient.GetAssetsInfo(suite.T(), assetID.Bytes())
}

func GetAssetInfoGrpcScala(suite *f.BaseSuite, assetID crypto.Digest) *g.AssetInfoResponse {
	return suite.Clients.ScalaClient.GRPCClient.GetAssetsInfo(suite.T(), assetID.Bytes())
}

func GetAssetInfoGrpc(suite *f.BaseSuite, assetID crypto.Digest) AssetInfo {
	return AssetInfo{GetAssetInfoGrpcGo(suite, assetID), GetAssetInfoGrpcScala(suite, assetID)}
}

func GetAssetBalanceGo(suite *f.BaseSuite, address proto.WavesAddress, assetID crypto.Digest) int64 {
	return suite.Clients.GoClient.GRPCClient.GetAssetBalance(suite.T(), address, assetID.Bytes()).GetAmount()
}

func GetAssetBalanceScala(suite *f.BaseSuite, address proto.WavesAddress, assetID crypto.Digest) int64 {
	return suite.Clients.ScalaClient.GRPCClient.GetAssetBalance(suite.T(), address, assetID.Bytes()).GetAmount()
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

type txRsp struct {
	Name string
	ID   string
}

func GetTxIdsInBlockchain(suite *f.BaseSuite, ids map[string]*crypto.Digest) map[string]string {
	ctx, cancel := context.WithTimeout(suite.MainCtx, 2*DefaultWaitTimeout)
	defer cancel()

	txIDs := make(map[string]string, 2*len(ids))
	for ctx.Err() == nil {
		if len(txIDs) == 2*len(ids) { // fast path
			return txIDs
		}
		select {
		case <-ctx.Done():
			return txIDs
		case <-time.After(time.Second):
			ch := make(chan txRsp, 2*len(ids))
			wg := sync.WaitGroup{}
			for name, id := range ids {
				goTxID := "Go " + name
				if _, ok := txIDs[goTxID]; !ok {
					wg.Add(1)
					go func(name string, id crypto.Digest) {
						defer wg.Done()
						_, _, errGo := suite.Clients.GoClient.HTTPClient.TransactionInfoRaw(id)
						if errGo == nil {
							ch <- txRsp{Name: name, ID: id.String()}
						}
					}(goTxID, *id)
				}
				scalaTxID := "Scala " + name
				if _, ok := txIDs[scalaTxID]; !ok {
					wg.Add(1)
					go func(name string, id crypto.Digest) {
						defer wg.Done()
						_, _, errScala := suite.Clients.ScalaClient.HTTPClient.TransactionInfoRaw(id)
						if errScala == nil {
							ch <- txRsp{Name: name, ID: id.String()}
						}
					}(scalaTxID, *id)
				}
			}
			wg.Wait()
			close(ch)
			for rsp := range ch {
				if rsp.Name == "" || rsp.ID == "" {
					continue
				}
				if _, ok := txIDs[rsp.Name]; !ok {
					txIDs[rsp.Name] = rsp.ID
				}
			}
		}
	}
	return txIDs
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

func GetTransactionInfoAfterWaitingGo(suite *f.BaseSuite, id crypto.Digest, errWtGo error) {
	if errWtGo != nil {
		suite.T().Log(errors.Errorf("Go Errors after waiting: %s", errWtGo))
	} else {
		txInfoRawGo, respGo, goRqErr := suite.Clients.GoClient.HTTPClient.TransactionInfoRaw(id)
		if goRqErr != nil {
			suite.T().Logf("Error on requesting Tx Info Go: %v", goRqErr)
		} else {
			suite.T().Logf("Tx Info Go after waiting: %s, Response Go: %s",
				GetTransactionJsonOrErrMsg(txInfoRawGo), respGo.Status)
		}
	}
}

func GetTransactionInfoAfterWaitingScala(suite *f.BaseSuite, id crypto.Digest, errWtScala error) {
	if errWtScala != nil {
		suite.T().Log(errors.Errorf("Scala Errors after waiting: %s", errWtScala))
	} else {
		txInfoRawScala, respScala, scalaRqErr := suite.Clients.ScalaClient.HTTPClient.TransactionInfoRaw(id)
		if scalaRqErr != nil {
			suite.T().Logf("Error on requesting Tx Info Scala: %v", scalaRqErr)
		} else {
			suite.T().Logf("Tx Info Scala after waiting: %s, Response Scala: %s",
				GetTransactionJsonOrErrMsg(txInfoRawScala), respScala.Status)
		}
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

	suite.Clients.SendToNodes(suite.T(), txMsg, suite.SendToNodes)
	suite.T().Log("Tx msg was successfully send to nodes")

	suite.T().Log("Waiting for Tx appears in Blockchain")
	errWtGo, errWtScala := suite.Clients.WaitForTransaction(id, timeout)

	GetTransactionInfoAfterWaitingGo(suite, id, errWtGo)
	GetTransactionInfoAfterWaitingScala(suite, id, errWtScala)

	return NewConsideredTransaction(id, nil, nil, errWtGo, errWtScala, nil, nil)
}

func BroadcastAndWaitTransaction(suite *f.BaseSuite, tx proto.Transaction,
	scheme proto.Scheme, waitForTx bool) ConsideredTransaction {
	timeout := DefaultInitialTimeout
	id := ExtractTxID(suite.T(), tx, scheme)
	if waitForTx {
		timeout = DefaultWaitTimeout
	}

	respGo, errBrdCstGo, respScala, errBrdCstScala := suite.Clients.BroadcastToNodes(suite.T(), tx,
		suite.SendToNodes)
	suite.T().Log("Tx was successfully broadcast to nodes")

	suite.T().Log("Waiting for Tx appears in Blockchain")
	errWtGo, errWtScala := suite.Clients.WaitForTransaction(id, timeout)

	GetTransactionInfoAfterWaitingGo(suite, id, errWtGo)
	GetTransactionInfoAfterWaitingScala(suite, id, errWtScala)

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
	return suite.Clients.GoClient.GRPCClient.GetBlock(suite.T(), height).GetBlock()
}

func GetBlockScala(suite *f.BaseSuite, height uint64) *waves.Block {
	return suite.Clients.ScalaClient.GRPCClient.GetBlock(suite.T(), height).GetBlock()
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

// GetRewardsGo get response from /blockchain/rewards.
func GetRewardsGo(suite *f.BaseSuite) *client.RewardInfo {
	return suite.Clients.GoClient.HTTPClient.Rewards(suite.T())
}

func GetRewardsScala(suite *f.BaseSuite) *client.RewardInfo {
	return suite.Clients.ScalaClient.HTTPClient.Rewards(suite.T())
}

func GetRewards(suite *f.BaseSuite) (*client.RewardInfo, *client.RewardInfo) {
	return GetRewardsGo(suite), GetRewardsScala(suite)
}

// GetRewardsAtHeightGo get response from /blockchain/rewards/{height}.
func GetRewardsAtHeightGo(suite *f.BaseSuite, height uint64) *client.RewardInfo {
	return suite.Clients.GoClient.HTTPClient.RewardsAtHeight(suite.T(), height)
}

func GetRewardsAtHeightScala(suite *f.BaseSuite, height uint64) *client.RewardInfo {
	return suite.Clients.ScalaClient.HTTPClient.RewardsAtHeight(suite.T(), height)
}

func GetRewardsAtHeight(suite *f.BaseSuite, height uint64) (*client.RewardInfo, *client.RewardInfo) {
	return GetRewardsAtHeightGo(suite, height), GetRewardsAtHeightScala(suite, height)
}

func GetCurrentRewardGo(suite *f.BaseSuite, height uint64) uint64 {
	return suite.Clients.GoClient.HTTPClient.RewardsAtHeight(suite.T(), height).CurrentReward
}

func GetCurrentRewardScala(suite *f.BaseSuite, height uint64) uint64 {
	return suite.Clients.ScalaClient.HTTPClient.RewardsAtHeight(suite.T(), height).CurrentReward
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
	goCh := make(chan uint64)
	scalaCh := make(chan uint64)
	go func() {
		goCh <- GetRewardTermAtHeightGo(suite, height)
	}()
	go func() {
		scalaCh <- GetRewardTermAtHeightScala(suite, height)
	}()
	termGo := <-goCh
	termScala := <-scalaCh
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
	return suite.Clients.GoClient.HTTPClient.RollbackToHeight(suite.T(), height, returnTxToUtx)
}

func GetRollbackToHeightScala(suite *f.BaseSuite, height uint64, returnTxToUtx bool) *proto.BlockID {
	return suite.Clients.ScalaClient.HTTPClient.RollbackToHeight(suite.T(), height, returnTxToUtx)
}

func GetRollbackToHeight(suite *f.BaseSuite, height uint64, returnTxToUtx bool) (*proto.BlockID, *proto.BlockID) {
	suite.T().Logf("Rollback to height: %d from height: %d", height, GetHeight(suite))
	goCh := make(chan *proto.BlockID)
	scalaCh := make(chan *proto.BlockID)
	go func() {
		goCh <- GetRollbackToHeightGo(suite, height, returnTxToUtx)
	}()
	go func() {
		scalaCh <- GetRollbackToHeightScala(suite, height, returnTxToUtx)
	}()
	return <-goCh, <-scalaCh
}
