package state

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	maxEstimatorVersion = 4
)

type checkerInfo struct {
	currentTimestamp        uint64
	parentTimestamp         uint64
	blockID                 proto.BlockID
	blockVersion            proto.BlockVersion
	blockchainHeight        proto.Height
	rideV5Activated         bool
	rideV6Activated         bool
	blockRewardDistribution bool
}

func (i *checkerInfo) estimatorVersion() int {
	if i.rideV6Activated {
		return 4
	}
	switch i.blockVersion {
	case proto.ProtobufBlockVersion:
		return 3
	case proto.RewardBlockVersion:
		return 2
	default:
		return 1
	}
}

type transactionChecker struct {
	genesis  proto.BlockID
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

func newTransactionChecker(
	genesis proto.BlockID,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
) (*transactionChecker, error) {
	return &transactionChecker{genesis, stor, settings}, nil
}

type scriptFeaturesActivations struct {
	rideForDAppsActivated, blockV5Activated, rideV5Activated, rideV6Activated bool
}

func (tc *transactionChecker) scriptActivation(libVersion ast.LibraryVersion, hasBlockV2 bool) (scriptFeaturesActivations, error) {
	rideForDAppsActivated, err := tc.stor.features.newestIsActivated(int16(settings.Ride4DApps))
	if err != nil {
		return scriptFeaturesActivations{}, errs.Extend(err, "transactionChecker scriptActivation isActivated")
	}
	blockV5Activated, err := tc.stor.features.newestIsActivated(int16(settings.BlockV5))
	if err != nil {
		return scriptFeaturesActivations{}, err
	}
	rideV5Activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV5))
	if err != nil {
		return scriptFeaturesActivations{}, err
	}
	rideV6Activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV6))
	if err != nil {
		return scriptFeaturesActivations{}, err
	}
	blockRewardDistributionActivated, err := tc.stor.features.newestIsActivated(int16(settings.BlockRewardDistribution))
	if err != nil {
		return scriptFeaturesActivations{}, err
	}
	lightNodeActivated, err := tc.stor.features.newestIsActivated(int16(settings.LightNode))
	if err != nil {
		return scriptFeaturesActivations{}, err
	}
	if libVersion < ast.LibV4 && lightNodeActivated {
		return scriptFeaturesActivations{},
			errors.Errorf("scripts with versions below %d are disabled after activation of Light Node feature",
				ast.LibV4)
	}
	if libVersion == ast.LibV3 && !rideForDAppsActivated {
		return scriptFeaturesActivations{},
			errors.New("Ride4DApps feature must be activated for scripts version 3")
	}
	if hasBlockV2 && !rideForDAppsActivated {
		return scriptFeaturesActivations{},
			errors.New("Ride4DApps feature must be activated for scripts that have block version 2")
	}
	if libVersion == ast.LibV4 && !blockV5Activated {
		return scriptFeaturesActivations{},
			errors.New("MultiPaymentInvokeScript feature must be activated for scripts version 4")
	}
	if libVersion == ast.LibV5 && !rideV5Activated {
		return scriptFeaturesActivations{},
			errors.New("RideV5 feature must be activated for scripts version 5")
	}
	if libVersion == ast.LibV6 && !rideV6Activated {
		return scriptFeaturesActivations{},
			errors.New("RideV6 feature must be activated for scripts version 6")
	}
	if libVersion == ast.LibV7 && !blockRewardDistributionActivated {
		return scriptFeaturesActivations{},
			errors.New("BlockRewardDistribution feature must be activated for scripts version 7")
	}
	if libVersion == ast.LibV8 && !lightNodeActivated {
		return scriptFeaturesActivations{},
			errors.New("LightNode feature must be activated for scripts version 8")
	}
	return scriptFeaturesActivations{
		rideForDAppsActivated: rideForDAppsActivated,
		blockV5Activated:      blockV5Activated,
		rideV5Activated:       rideV5Activated,
		rideV6Activated:       rideV6Activated,
	}, nil
}

func (tc *transactionChecker) checkScriptComplexity(libVersion ast.LibraryVersion, estimation ride.TreeEstimation, isDapp, reducedVerifierComplexity bool) error {
	/*
		| Script Type                            | Max complexity before BlockV5 | Max complexity after BlockV5 |
		| -------------------------------------- | ----------------------------- | ---------------------------- |
		| Account / DApp Verifier V1, V2         | 2000                          | 2000                         |
		| Account / DApp Verifier V3, V4, V5, V6 | 4000                          | 2000                         |
		| Asset Verifier V1, V2                  | 2000                          | 2000                         |
		| Asset Verifier V3, V4, V5, V6          | 4000                          | 4000                         |
		| DApp Callable V1, V2                   | 2000                          | 2000                         |
		| DApp Callable V3, V4                   | 4000                          | 4000                         |
		| DApp Callable V5                       | 10000                         | 10000                        |
		| DApp Callable V6, V7, V8               | 52000                         | 52000                        |
	*/
	var maxCallableComplexity, maxVerifierComplexity int
	switch version := libVersion; version {
	case ast.LibV1, ast.LibV2:
		maxCallableComplexity = MaxCallableScriptComplexityV12
		maxVerifierComplexity = MaxVerifierScriptComplexityReduced
	case ast.LibV3, ast.LibV4:
		maxCallableComplexity = MaxCallableScriptComplexityV34
		maxVerifierComplexity = MaxVerifierScriptComplexity
	case ast.LibV5:
		maxCallableComplexity = MaxCallableScriptComplexityV5
		maxVerifierComplexity = MaxVerifierScriptComplexity
	case ast.LibV6, ast.LibV7, ast.LibV8:
		maxCallableComplexity = MaxCallableScriptComplexityV6
		maxVerifierComplexity = MaxVerifierScriptComplexity
	default:
		return errors.Errorf("unknown script LibVersion=%d", version)
	}
	if reducedVerifierComplexity {
		maxVerifierComplexity = MaxVerifierScriptComplexityReduced
	}
	if !isDapp { // Expression (simple) script, has only verifier.
		if complexity := estimation.Verifier; complexity > maxVerifierComplexity {
			return errors.Errorf("script complexity %d exceeds maximum allowed complexity of %d", complexity, maxVerifierComplexity)
		}
		return nil
	}
	if complexity := estimation.Verifier; complexity > maxVerifierComplexity {
		return errors.Errorf("verifier script complexity %d exceeds maximum allowed complexity of %d", complexity, maxVerifierComplexity)
	}
	if complexity := estimation.Estimation; complexity > maxCallableComplexity {
		return errors.Errorf("callable script complexity %d exceeds maximum allowed complexity of %d", complexity, maxCallableComplexity)
	}
	return nil
}

func (tc *transactionChecker) checkDAppCallables(tree *ast.Tree, rideV6Activated bool) error {
	if !rideV6Activated || tree.LibVersion < ast.LibV6 {
		return nil
	}
	for _, fn := range tree.Meta.Functions {
		for _, arg := range fn.Arguments {
			switch arg := arg.(type) {
			case meta.ListType:
				if u, ok := arg.Inner.(meta.UnionType); ok && len(u) > 1 {
					return errors.New("union type inside list type is not allowed in callable function arguments of script")
				}
			case meta.UnionType:
				if len(arg) > 1 {
					return errors.New("union type is not allowed in callable function arguments of script")
				}
			}
		}
	}
	return nil
}

func (tc *transactionChecker) checkScript(
	script proto.Script,
	estimatorVersion int,
	reducedVerifierComplexity bool,
) (ride.TreeEstimation, error) {
	tree, err := serialization.Parse(script)
	if err != nil {
		return ride.TreeEstimation{}, errs.Extend(err, "failed to build AST")
	}
	activations, err := tc.scriptActivation(tree.LibVersion, tree.HasBlockV2)
	if err != nil {
		return ride.TreeEstimation{}, errs.Extend(err, "script activation check failed")
	}
	maxSize := proto.MaxVerifierScriptSize
	if tree.IsDApp() {
		maxSize = proto.MaxContractScriptSizeV1V5
		if activations.rideV6Activated {
			maxSize = proto.MaxContractScriptSizeV6
		}
	}
	if l := len(script); l > maxSize {
		return ride.TreeEstimation{}, errors.Errorf("script size %d is greater than limit of %d", l, maxSize)
	}
	if tree.IsDApp() {
		if checkDAppErr := tc.checkDAppCallables(tree, activations.rideV6Activated); checkDAppErr != nil {
			return ride.TreeEstimation{}, errors.Wrap(checkDAppErr, "failed to check script callables")
		}
	}
	est, err := ride.EstimateTree(tree, estimatorVersion)
	if err != nil {
		return ride.TreeEstimation{}, errs.Extend(err, "failed to estimate script complexity")
	}

	if scErr := tc.checkScriptComplexity(tree.LibVersion, est, tree.IsDApp(), reducedVerifierComplexity); scErr != nil {
		return ride.TreeEstimation{}, errors.Wrap(scErr, "failed to check script complexity")
	}
	return est, nil
}

type txAssets struct {
	feeAsset    proto.OptionalAsset
	smartAssets []crypto.Digest
}

func (tc *transactionChecker) checkFee(
	tx proto.Transaction,
	assets *txAssets,
	info *checkerInfo,
) error {
	sponsorshipActivated, err := tc.stor.sponsoredAssets.isSponsorshipActivated()
	if err != nil {
		return err
	}
	if !sponsorshipActivated {
		// Sponsorship is not yet activated.
		return nil
	}
	params := &feeValidationParams{
		stor:            tc.stor,
		settings:        tc.settings,
		txAssets:        assets,
		rideV5Activated: info.rideV5Activated,
	}
	if !assets.feeAsset.Present {
		return checkMinFeeWaves(tx, params)
	}
	return checkMinFeeAsset(tx, assets.feeAsset.ID, params)
}

func (tc *transactionChecker) checkFromFuture(timestamp uint64) bool {
	return timestamp > tc.settings.TxFromFutureCheckAfterTime
}

func (tc *transactionChecker) checkTimestamps(txTimestamp, blockTimestamp, prevBlockTimestamp uint64) error {
	if tc.checkFromFuture(blockTimestamp) && txTimestamp > blockTimestamp+tc.settings.MaxTxTimeForwardOffset {
		return errs.NewMistiming(fmt.Sprintf("Transaction timestamp %d is more than %dms in the future", txTimestamp, tc.settings.MaxTxTimeForwardOffset))
	}
	if prevBlockTimestamp == 0 {
		// If we check tx from genesis block, there is no parent, so transaction can not be early.
		return nil
	}
	if txTimestamp < prevBlockTimestamp-tc.settings.MaxTxTimeBackOffset {
		return errs.NewMistiming(fmt.Sprintf("Transaction timestamp %d is more than %dms in the past: early transaction creation time", txTimestamp, tc.settings.MaxTxTimeBackOffset))
	}
	return nil
}

func (tc *transactionChecker) checkAsset(asset *proto.OptionalAsset) error {
	if !tc.stor.assets.newestAssetExists(*asset) {
		return errs.NewUnknownAsset(fmt.Sprintf("unknown asset %s", asset.String()))
	}
	return nil
}

func (tc *transactionChecker) checkFeeAsset(asset *proto.OptionalAsset) error {
	if err := tc.checkAsset(asset); err != nil {
		return err
	}
	if !asset.Present {
		// No need to check Waves.
		return nil
	}
	// Check sponsorship.
	sponsorshipActivated, err := tc.stor.sponsoredAssets.isSponsorshipActivated()
	if err != nil {
		return err
	}
	if !sponsorshipActivated {
		return nil
	}
	isSponsored, err := tc.stor.sponsoredAssets.newestIsSponsored(proto.AssetIDFromDigest(asset.ID))
	if err != nil {
		return err
	}
	if !isSponsored {
		return errors.Errorf("asset %s is not sponsored and can not be used to pay fees", asset.ID.String())
	}
	return nil
}

func (tc *transactionChecker) smartAssets(assets []proto.OptionalAsset) ([]crypto.Digest, error) {
	var smartAssets []crypto.Digest
	for _, asset := range assets {
		if !asset.Present {
			// Waves can not be scripted.
			continue
		}
		hasScript, err := tc.stor.scriptsStorage.newestIsSmartAsset(proto.AssetIDFromDigest(asset.ID))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check newestIsSmartAsset for asset %q", asset.String())
		}
		if hasScript {
			smartAssets = append(smartAssets, asset.ID)
		}
	}
	return smartAssets, nil
}

func (tc *transactionChecker) smartAssetsFromMap(assets map[proto.OptionalAsset]struct{}) ([]crypto.Digest, error) {
	var smartAssets []crypto.Digest
	for a := range assets {
		if !a.Present {
			// Waves can not be scripted.
			continue
		}
		scripted, err := tc.stor.scriptsStorage.newestIsSmartAsset(proto.AssetIDFromDigest(a.ID))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check newestIsSmartAsset for asset %q", a.String())
		}
		if scripted {
			smartAssets = append(smartAssets, a.ID)
		}
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkGenesis(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	if info.blockID != tc.genesis {
		return out, errors.New("genesis transaction inside of non-genesis block")
	}
	if info.blockchainHeight != 0 {
		return out, errors.New("genesis transaction on non zero height")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkPayment(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.Payment)
	if !ok {
		return out, errors.New("failed to convert interface to Payment transaction")
	}
	if info.blockchainHeight >= tc.settings.BlockVersion3AfterHeight {
		return out, errors.Errorf("Payment transaction is deprecated after height %d", tc.settings.BlockVersion3AfterHeight)
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkTransfer(tx *proto.Transfer, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	if err := tc.checkAsset(&tx.AmountAsset); err != nil {
		return err
	}
	if err := tc.checkFeeAsset(&tx.FeeAsset); err != nil {
		return err
	}
	return nil
}

func (tc *transactionChecker) checkTransferWithSig(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return out, errors.New("failed to convert interface to TransferWithSig transaction")
	}
	allAssets := []proto.OptionalAsset{tx.AmountAsset}
	smartAssets, err := tc.smartAssets(allAssets)
	if err != nil {
		return out, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	if err := tc.checkTransfer(&tx.Transfer, info); err != nil {
		return out, err
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

func (tc *transactionChecker) checkEthereumTransactionWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	metamaskActivated, err := tc.stor.features.newestIsActivated(int16(settings.RideV6))
	if err != nil {
		return out, errors.Errorf("failed to check whether feature %d was activated: %v", settings.RideV6, err)
	}
	if !metamaskActivated {
		return out, errors.Errorf("failed to handle ethereum transaction: feature %d has not been activated yet", settings.RideV6)
	}

	tx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return out, errors.New("failed to cast 'Transaction' interface to 'EthereumTransaction' type")
	}
	if err := tc.checkTimestamps(tx.GetTimestamp(), info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp in ethereum transaction")
	}

	needToValidateNonEmptyCallData := info.blockRewardDistribution
	var smartAssets []crypto.Digest
	switch kind := tx.TxKind.(type) {
	case *proto.EthereumTransferWavesTxKind:
		if tx.Value() == nil {
			return out, errors.New("amount of ethereum transfer waves is nil")
		}
		if l := len(tx.Data()); l != 0 {
			return out, errors.Errorf("ethereum call data must be empty for waves transfer, but size is %d", l)
		}
		res, err := proto.EthereumWeiToWavelet(tx.Value())
		if err != nil {
			return out, errors.Wrapf(err,
				"failed to convert wei amount from ethereum transaction to wavelets. value is %s",
				tx.Value().String())
		}
		if res == 0 {
			return out, errors.New("the amount of ethereum transfer waves is 0, which is forbidden")
		}
	case *proto.EthereumTransferAssetsErc20TxKind:
		if kind.Arguments.Amount == 0 {
			return out, errors.New("the amount of ethereum transfer assets is 0, which is forbidden")
		}
		if l := len(tx.Data()); needToValidateNonEmptyCallData && l != ethabi.ERC20TransferCallDataSize {
			return out, errors.Errorf("ethereum call data must be %d size for assset transfer, but size is %d",
				ethabi.ERC20TransferCallDataSize, l)
		}
		allAssets := []proto.OptionalAsset{kind.Asset}
		smartAssets, err = tc.smartAssets(allAssets)
		if err != nil {
			return out, err
		}
	case *proto.EthereumInvokeScriptTxKind:
		var (
			decodedData = kind.DecodedData()
			abiPayments = decodedData.Payments
		)
		if len(abiPayments) > maxPaymentsCountSinceRideV5Activation {
			return out, errors.New("no more than 10 payments is allowed since RideV5 activation")
		}
		if needToValidateNonEmptyCallData {
			dApp, err := tx.To().ToWavesAddress(tc.settings.AddressSchemeCharacter)
			if err != nil {
				return out, errors.Wrapf(err, "failed to convert eth addr %q to waves addr", tx.To().String())
			}
			if err := kind.ValidateCallData(dApp); err != nil {
				return out, errors.Wrap(err, "failed to validate callData")
			}
		}

		paymentAssets := make([]proto.OptionalAsset, 0, len(abiPayments))
		for _, p := range abiPayments {
			if p.Amount <= 0 && info.blockchainHeight > tc.settings.InvokeNoZeroPaymentsAfterHeight {
				return out, errors.Errorf("invalid payment amount '%d'", p.Amount)
			}
			optAsset := proto.NewOptionalAsset(p.PresentAssetID, p.AssetID)
			if optAsset.Present {
				if err := tc.checkAsset(&optAsset); err != nil {
					return out, errs.Extend(err, "bad payment asset")
				}
			}
			// if optAsset.Present == false then it's WAVES asset
			// we don't have to check WAVES asset because it can't be scripted and always exists
			paymentAssets = append(paymentAssets, optAsset)
		}
		smartAssets, err = tc.smartAssets(paymentAssets)
		if err != nil {
			return out, err
		}
	default:
		return out, errors.Errorf("failed to check ethereum transaction, wrong kind (%T) of tx", kind)
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, errors.Wrap(err, "failed fee validation for ethereum transaction")
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

func (tc *transactionChecker) checkTransferWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to TransferWithProofs transaction")
	}
	allAssets := []proto.OptionalAsset{tx.AmountAsset}
	smartAssets, err := tc.smartAssets(allAssets)
	if err != nil {
		return out, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkTransfer(&tx.Transfer, info); err != nil {
		return out, err
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

func (tc *transactionChecker) isValidUtf8(str string) error {
	if !utf8.ValidString(str) {
		return errors.Errorf("str %s is not valid UTF-8", str)
	}
	return nil
}

func (tc *transactionChecker) checkIssue(tx *proto.Issue, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	blockV5Activated, err := tc.stor.features.newestIsActivated(int16(settings.BlockV5))
	if err != nil {
		return err
	}
	if !blockV5Activated {
		// We do not check isValidUtf8() before BlockV5 activation.
		return nil
	}
	if err := tc.isValidUtf8(tx.Name); err != nil {
		return errs.Extend(err, "invalid UTF-8 in name")
	}
	if err := tc.isValidUtf8(tx.Description); err != nil {
		return errs.Extend(err, "invalid UTF-8 in description")
	}
	return nil
}

func (tc *transactionChecker) checkIssueWithSig(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return out, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	if err := tc.checkIssue(&tx.Issue, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkIssueWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	if err := tc.checkIssue(&tx.Issue, info); err != nil {
		return out, err
	}
	if len(tx.Script) == 0 {
		// No script checks / actions are needed.
		return out, nil
	}
	// For asset scripts do not reduce verifier complexity and only one estimation is required
	currentEstimatorVersion := info.estimatorVersion()
	estimation, err := tc.checkScript(tx.Script, currentEstimatorVersion, false)
	if err != nil {
		return out, errors.Errorf("checkScript() tx %s: %v", tx.ID.String(), err)
	}
	return txCheckerData{
		scriptEstimation: &scriptEstimation{
			currentEstimatorVersion: currentEstimatorVersion,
			scriptIsEmpty:           false,
			estimation:              estimation,
		},
	}, nil
}

func (tc *transactionChecker) checkReissue(tx *proto.Reissue, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	assetInfo, err := tc.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(tx.AssetID))
	if err != nil {
		return err
	}
	if !bytes.Equal(assetInfo.Issuer[:], tx.SenderPK[:]) {
		return errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}
	if tc.settings.CanReissueNonReissueablePeriod(info.currentTimestamp) {
		// Due to bugs in existing blockchain it is valid to reissue non-reissuable asset in this time period.
		return nil
	}
	if !assetInfo.reissuable {
		return errs.NewAssetIsNotReissuable("attempt to reissue asset which is not reissuable")
	}
	// Check Int64 overflow.
	if (math.MaxInt64-int64(tx.Quantity) < assetInfo.quantity.Int64()) && (info.currentTimestamp >= tc.settings.ReissueBugWindowTimeEnd) {
		return errors.New("asset total value overflow")
	}
	return nil
}

func (tc *transactionChecker) checkReissueWithSig(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return out, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets)
	if err != nil {
		return out, err
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	if err := tc.checkReissue(&tx.Reissue, info); err != nil {
		return out, err
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

func (tc *transactionChecker) checkReissueWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets)
	if err != nil {
		return out, err
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkReissue(&tx.Reissue, info); err != nil {
		return out, err
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

func (tc *transactionChecker) checkBurn(tx *proto.Burn, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	assetInfo, err := tc.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(tx.AssetID))
	if err != nil {
		return err
	}
	// Verify sender.
	burnAnyTokensEnabled, err := tc.stor.features.newestIsActivated(int16(settings.BurnAnyTokens))
	if err != nil {
		return err
	}
	if !burnAnyTokensEnabled && !bytes.Equal(assetInfo.Issuer[:], tx.SenderPK[:]) {
		return errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}
	// Check burn amount.
	quantityDiff := big.NewInt(int64(tx.Amount))
	if assetInfo.quantity.Cmp(quantityDiff) == -1 {
		return errs.NewAccountBalanceError("trying to burn more assets than exist at all")
	}
	return nil
}

func (tc *transactionChecker) checkBurnWithSig(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return out, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets)
	if err != nil {
		return out, err
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	if err := tc.checkBurn(&tx.Burn, info); err != nil {
		return out, err
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

func (tc *transactionChecker) checkBurnWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets)
	if err != nil {
		return out, err
	}

	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkBurn(&tx.Burn, info); err != nil {
		return out, err
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

// orderScriptedAccount checks that sender account is a scripted account.
// This method works for both proto.EthereumAddress and proto.WavesAddress.
// Note that only real proto.WavesAddress account can have a verifier.
func (tc *transactionChecker) orderScriptedAccount(order proto.Order) (bool, error) {
	senderAddr, err := order.GetSender(tc.settings.AddressSchemeCharacter)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get sender for order")
	}
	// senderWavesAddr needs only for newestAccountHasVerifier check
	senderWavesAddr, err := senderAddr.ToWavesAddress(tc.settings.AddressSchemeCharacter)
	if err != nil {
		return false, errors.Wrapf(err, "failed to transform (%T) address type to WavesAddress type", senderAddr)
	}
	return tc.stor.scriptsStorage.newestAccountHasVerifier(senderWavesAddr)
}

func (tc *transactionChecker) checkEnoughVolume(order proto.Order, newFee, newAmount uint64) error {
	orderID, err := order.GetID()

	if err != nil {
		return err
	}
	fullAmount := order.GetAmount()
	if newAmount > fullAmount {
		return errors.New("current amount exceeds total order amount")
	}
	fullFee := order.GetMatcherFee()
	if newFee > fullFee {
		return errors.New("current fee exceeds total order fee")
	}
	filledAmount, filledFee, err := tc.stor.ordersVolumes.newestFilled(orderID)
	if err != nil {
		return err
	}
	if fullAmount-newAmount < filledAmount {
		return errors.New("order amount volume is overflowed")
	}
	if fullFee-newFee < filledFee {
		return errors.New("order fee volume is overflowed")
	}
	return nil
}

func checkOrderWithMetamaskFeature(o proto.Order, metamaskActivated bool) error {
	if metamaskActivated {
		return nil
	}
	if o.GetVersion() >= 4 {
		if m := o.GetPriceMode(); m != proto.OrderPriceModeDefault {
			return errors.Errorf("invalid order prce mode before metamask feature activation: got %q, want %q",
				m.String(), proto.OrderPriceModeDefault.String(),
			)
		}
	}
	if _, ok := o.(*proto.EthereumOrderV4); ok {
		return errors.New("ethereum order is not allowed before metamask feature activation")
	}
	return nil
}

func (tc *transactionChecker) checkExchange(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return nil, errors.New("failed to convert interface to Exchange transaction")
	}
	if err := tc.checkTimestamps(tx.GetTimestamp(), info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errs.Extend(err, "invalid timestamp")
	}
	if tx.GetOrder1().GetOrderType() != proto.Buy && tx.GetOrder2().GetOrderType() != proto.Sell {
		if !proto.IsProtobufTx(transaction) {
			return nil, errors.New("sell order not allowed on first place in exchange transaction of versions prior 3")
		}
	}
	// validate orders
	so, err := tx.GetSellOrder()
	if err != nil {
		return nil, errs.Extend(err, "sell order")
	}
	if err := tc.checkEnoughVolume(so, tx.GetSellMatcherFee(), tx.GetAmount()); err != nil {
		return nil, errs.Extend(err, "exchange transaction; sell order")
	}
	bo, err := tx.GetBuyOrder()
	if err != nil {
		return nil, errs.Extend(err, "buy order")
	}
	if err := tc.checkEnoughVolume(bo, tx.GetBuyMatcherFee(), tx.GetAmount()); err != nil {
		return nil, errs.Extend(err, "exchange transaction; buy order")
	}
	o1 := tx.GetOrder1()
	o2 := tx.GetOrder2()
	metamaskActivated, err := tc.stor.features.newestIsActivated(int16(settings.RideV6))
	if err != nil {
		return nil, err
	}
	if errO1 := checkOrderWithMetamaskFeature(o1, metamaskActivated); errO1 != nil {
		return nil, errors.Wrap(errO1, "order1 metamask feature checks failed")
	}
	if errO2 := checkOrderWithMetamaskFeature(o2, metamaskActivated); errO2 != nil {
		return nil, errors.Wrap(errO2, "order2 metamask feature checks failed")
	}

	// Check assets.
	allAssets := map[proto.OptionalAsset]struct{}{
		so.GetAssetPair().AmountAsset: {},
		so.GetAssetPair().PriceAsset:  {},
	}
	ordersAssets := map[proto.OptionalAsset]struct{}{
		so.GetAssetPair().AmountAsset: {},
		so.GetAssetPair().PriceAsset:  {},
	}
	// Add matcher fee assets to map to checkAsset() them later.
	switch o := o1.(type) {
	case *proto.OrderV3, *proto.OrderV4, *proto.EthereumOrderV4:
		allAssets[o.GetMatcherFeeAsset()] = struct{}{}
	}
	switch o := o2.(type) {
	case *proto.OrderV3, *proto.OrderV4, *proto.EthereumOrderV4:
		allAssets[o.GetMatcherFeeAsset()] = struct{}{}
	}
	for a := range allAssets {
		if err := tc.checkAsset(&a); err != nil {
			return nil, errs.Extend(err, "Assets should be issued before they can be traded")
		}
	}
	ordersSmartAssets, err := tc.smartAssetsFromMap(ordersAssets)
	if err != nil {
		return nil, err
	}
	txa := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: ordersSmartAssets}
	if errCF := tc.checkFee(transaction, txa, info); errCF != nil {
		return nil, errCF
	}
	smartAssetsActivated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAssets))
	if err != nil {
		return nil, err
	}
	smartAssets, err := tc.smartAssetsFromMap(allAssets)
	if err != nil {
		return nil, err
	}
	if !smartAssetsActivated && (len(smartAssets) != 0) {
		return nil, errors.New("smart assets can't participate in Exchange because smart assets feature is disabled")
	}
	// Check smart accounts trading.
	smartTradingActivated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccountTrading))
	if err != nil {
		return nil, err
	}
	o1ScriptedAccount, err := tc.orderScriptedAccount(tx.GetOrder1())
	if err != nil {
		return nil, err
	}
	o2ScriptedAccount, err := tc.orderScriptedAccount(tx.GetOrder2())
	if err != nil {
		return nil, err
	}
	if o1ScriptedAccount && !smartTradingActivated {
		return nil, errors.New("first order is scripted, but smart trading is disabled")
	}
	if o2ScriptedAccount && !smartTradingActivated {
		return nil, errors.New("second order is scripted, but smart trading is disabled")
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkExchangeWithSig(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.ExchangeWithSig)
	if !ok {
		return out, errors.New("failed to convert interface to Payment transaction")
	}
	smartAssets, err := tc.checkExchange(tx, info)
	if err != nil {
		return out, err
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

func (tc *transactionChecker) checkExchangeWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.ExchangeWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to ExchangeWithProofs transaction")
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccountTrading))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("SmartAccountsTrading feature must be activated for ExchangeWithProofs transactions")
	}
	smartAssets, err := tc.checkExchange(tx, info)
	if err != nil {
		return out, err
	}
	if (tx.Order1.GetVersion() < 3) && (tx.Order2.GetVersion() < 3) { // it's not necessary to check OrderV3 feature activation
		return txCheckerData{smartAssets: smartAssets}, nil
	}
	// one or both order versions greater or equal 3, we have to check OrderV3 activation
	activated, err = tc.stor.features.newestIsActivated(int16(settings.OrderV3))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("OrderV3 feature must be activated for Exchange transactions with Order version 3")
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

func (tc *transactionChecker) checkLease(tx *proto.Lease, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tc.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	recipientAddr, err := recipientToAddress(tx.Recipient, tc.stor.aliases)
	if err != nil {
		return errors.Wrap(err, "failed convert recipient to address")
	}
	if senderAddr == recipientAddr {
		return errs.NewToSelf("trying to lease money to self")
	}
	return nil
}

func (tc *transactionChecker) checkLeaseWithSig(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return out, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	if err := tc.checkLease(&tx.Lease, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkLeaseWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkLease(&tx.Lease, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkLeaseCancel(tx *proto.LeaseCancel, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	l, err := tc.stor.leases.newestLeasingInfo(tx.LeaseID)
	if err != nil {
		return errs.Extend(err, "no leasing info found for this leaseID")
	}
	if !l.isActive() && (info.currentTimestamp > tc.settings.AllowMultipleLeaseCancelUntilTime) {
		return errs.NewTxValidationError("Reason: Cannot cancel already cancelled lease")
	}
	if (l.SenderPK != tx.SenderPK) && (info.currentTimestamp > tc.settings.AllowMultipleLeaseCancelUntilTime) {
		return errs.NewTxValidationError("LeaseTransaction was leased by other sender")
	}
	return nil
}

func (tc *transactionChecker) checkLeaseCancelWithSig(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return out, errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	if err := tc.checkLeaseCancel(&tx.LeaseCancel, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkLeaseCancelWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkLeaseCancel(&tx.LeaseCancel, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkCreateAlias(tx *proto.CreateAlias, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	if (info.currentTimestamp >= tc.settings.StolenAliasesWindowTimeStart) && (info.currentTimestamp <= tc.settings.StolenAliasesWindowTimeEnd) {
		// At this period it is fine to steal aliases.
		return nil
	}
	// Check if alias is already taken.
	if tc.stor.aliases.exists(tx.Alias.Alias) {
		return errs.NewAliasTaken("alias is already taken")
	}
	return nil
}

func (tc *transactionChecker) checkCreateAliasWithSig(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return out, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	if err := tc.checkCreateAlias(&tx.CreateAlias, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkCreateAliasWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	smartAccountsIsActivated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return out, err
	}
	if !smartAccountsIsActivated {
		return out, errors.New("SmartAccounts feature has not been activated yet")
	}
	rideV6IsActivated, err := tc.stor.features.newestIsActivated(int16(settings.RideV6))
	if err != nil {
		return out, err
	}
	// scala node can't accept more than 1 proof before RideV6 activation
	if tx.Proofs.Len() > 1 && !rideV6IsActivated {
		return out, errors.New("create alias tx with more than one proof is disabled before feature 17 (RideV6) activation")
	}
	if err := tc.checkCreateAlias(&tx.CreateAlias, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkMassTransferWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to MassTransferWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp")
	}
	allAssets := []proto.OptionalAsset{tx.Asset}
	smartAssets, err := tc.smartAssets(allAssets)
	if err != nil {
		return out, err
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.MassTransfer))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("MassTransfer transaction has not been activated yet")
	}
	if err := tc.checkAsset(&tx.Asset); err != nil {
		return out, err
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}

func (tc *transactionChecker) checkDataWithProofsSize(tx *proto.DataWithProofs, isRideV6Activated bool) error {
	switch {
	case isRideV6Activated:
		if pl := tx.Entries.PayloadSize(); pl > proto.MaxDataWithProofsV6PayloadBytes {
			return errors.Errorf("data entries payload size limit exceeded, limit=%d, actual size=%d",
				proto.MaxDataWithProofsV6PayloadBytes, pl,
			)
		}
	case proto.IsProtobufTx(tx):
		if pbSize := tx.ProtoPayloadSize(); pbSize > proto.MaxDataWithProofsProtoBytes {
			return errors.Errorf("data tx protobuf size limit exceeded, limit=%d, actual size=%d",
				proto.MaxDataWithProofsProtoBytes, pbSize,
			)
		}
	default:
		txBytes, err := tx.MarshalBinary(tc.settings.AddressSchemeCharacter)
		if err != nil {
			return err
		}
		if l := len(txBytes); l > proto.MaxDataWithProofsBytes {
			return errors.Errorf("data tx binary size limit exceeded, limit=%d, actual size=%d",
				proto.MaxDataWithProofsBytes, l,
			)
		}
	}
	return nil
}

func (tc *transactionChecker) checkDataWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to DataWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.DataTransaction))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("Data transaction has not been activated yet")
	}
	isRideV6Activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV6))
	if err != nil {
		return out, err
	}
	utf16KeyLen := tx.Version == 1 && !isRideV6Activated
	if err := tx.Entries.Valid(true, utf16KeyLen); err != nil {
		return out, errors.Wrap(err, "at least one of the DataWithProofs entry is not valid")
	}
	if err := tc.checkDataWithProofsSize(tx, isRideV6Activated); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkSponsorshipWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("sponsorship has not been activated yet")
	}
	if err := tc.checkAsset(proto.NewOptionalAssetFromDigest(tx.AssetID)); err != nil {
		return out, err
	}
	id := proto.AssetIDFromDigest(tx.AssetID)
	assetInfo, err := tc.stor.assets.newestAssetInfo(id)
	if err != nil {
		return out, err
	}
	if assetInfo.Issuer != tx.SenderPK {
		return out, errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}
	isSmart, err := tc.stor.scriptsStorage.newestIsSmartAsset(id)
	if err != nil {
		return out, errors.Wrapf(err, "failed to check newestIsSmartAsset for asset %q", tx.AssetID.String())
	}
	if isSmart {
		return out, errors.Errorf("can not sponsor smart asset %s", tx.AssetID.String())
	}
	return out, nil
}

func (tc *transactionChecker) checkSetScriptWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}

	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}

	currentEstimatorVersion := info.estimatorVersion()
	var estimation ride.TreeEstimation
	scriptIsEmpty := tx.Script.IsEmpty()
	if !scriptIsEmpty { // script isn't empty
		estimation, err = tc.checkScript(tx.Script, currentEstimatorVersion, info.blockVersion == proto.ProtobufBlockVersion)
		if err != nil {
			return out, errors.Wrapf(err, "checkScript() tx %s", tx.ID.String())
		}
	}
	return txCheckerData{
		scriptEstimation: &scriptEstimation{
			currentEstimatorVersion: currentEstimatorVersion,
			scriptIsEmpty:           scriptIsEmpty,
			estimation:              estimation,
		},
	}, nil
}

func (tc *transactionChecker) checkSetAssetScriptWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp")
	}
	id := proto.AssetIDFromDigest(tx.AssetID)
	assetInfo, err := tc.stor.assets.newestAssetInfo(id)
	if err != nil {
		return out, err
	}

	smartAssets := []crypto.Digest{tx.AssetID}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, errs.Extend(err, "check fee")
	}

	if !bytes.Equal(assetInfo.Issuer[:], tx.SenderPK[:]) {
		return out, errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}

	isSmart, err := tc.stor.scriptsStorage.newestIsSmartAsset(id)
	if err != nil {
		return out, errors.Wrapf(err, "failed to check newestIsSmartAsset for asset %q", tx.AssetID.String())
	}
	if len(tx.Script) == 0 {
		return out, errs.NewTxValidationError("Cannot set empty script")
	}
	if !isSmart {
		return out, errs.NewTxValidationError("Reason: Cannot set script on an asset issued without a script. Referenced assetId not found")
	}
	currentEstimatorVersion := info.estimatorVersion()
	// Do not reduce verifier complexity for asset scripts and only one estimation is required
	estimation, err := tc.checkScript(tx.Script, currentEstimatorVersion, false)
	if err != nil {
		return out, errors.Errorf("checkScript() tx %s: %v", tx.ID.String(), err)
	}
	return txCheckerData{
		smartAssets: smartAssets,
		scriptEstimation: &scriptEstimation{
			currentEstimatorVersion: currentEstimatorVersion,
			scriptIsEmpty:           false,
			estimation:              estimation,
		},
	}, nil
}

const maxPaymentsCountSinceRideV5Activation = 10

func (tc *transactionChecker) checkInvokeScriptWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp")
	}
	ride4DAppsActivated, err := tc.stor.features.newestIsActivated(int16(settings.Ride4DApps))
	if err != nil {
		return out, err
	}
	if !ride4DAppsActivated {
		return out, errors.New("can not use InvokeScript before Ride4DApps activation")
	}
	if err := tc.checkFeeAsset(&tx.FeeAsset); err != nil {
		return out, err
	}
	multiPaymentActivated, err := tc.stor.features.newestIsActivated(int16(settings.BlockV5))
	if err != nil {
		return out, err
	}
	rideV5activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV5))
	if err != nil {
		return out, err
	}
	l := len(tx.Payments)
	switch {
	case l > 1 && !multiPaymentActivated && !rideV5activated:
		return out, errors.New("no more than one payment is allowed")
	case l > 2 && multiPaymentActivated && !rideV5activated:
		return out, errors.New("no more than two payments is allowed")
	case l > maxPaymentsCountSinceRideV5Activation && rideV5activated:
		return out, errors.New("no more than ten payments is allowed since RideV5 activation")
	}
	var paymentAssets []proto.OptionalAsset
	for i := range tx.Payments {
		p := &tx.Payments[i]
		if paymentErr := tc.checkAsset(&p.Asset); paymentErr != nil {
			return out, errs.Extend(paymentErr, "bad payment asset")
		}
		paymentAssets = append(paymentAssets, p.Asset)
	}
	smartAssets, err := tc.smartAssets(paymentAssets)
	if err != nil {
		return out, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}

	dAppEstimationUpdate, ok, err := tc.tryCreateDAppEstimationUpdate(tx.ScriptRecipient, info.estimatorVersion())
	if err != nil {
		return out, err
	}
	var se *scriptEstimation
	if ok {
		se = &dAppEstimationUpdate
	}
	return txCheckerData{
		smartAssets:      smartAssets,
		scriptEstimation: se,
	}, nil
}

func (tc *transactionChecker) tryCreateDAppEstimationUpdate(
	rcp proto.Recipient,
	currentEstimatorVersion int,
) (scriptEstimation, bool, error) {
	rideV5Activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV5))
	if err != nil {
		return scriptEstimation{}, false, err
	}
	if rideV5Activated { // after rideV5 activation we're using estimating evaluator (see complexityCalculator)
		return scriptEstimation{}, false, nil // so we don't have to update script estimation
	}
	scriptAddr, err := recipientToAddress(rcp, tc.stor.aliases)
	if err != nil {
		return scriptEstimation{}, false, err
	}
	est, err := tc.stor.scriptsComplexity.newestScriptEstimationRecordByAddr(scriptAddr)
	if err != nil {
		return scriptEstimation{}, false, errors.Wrapf(err,
			"failed to get newest script estimation record by addr %q", scriptAddr,
		)
	}
	// we're using == because saved estimator version can't be greater than current, can be only less or equal
	if estimationIsNotStale := int(est.EstimatorVersion) == currentEstimatorVersion; estimationIsNotStale {
		return scriptEstimation{}, false, nil // no updates
	}
	// we have to create estimation update
	tree, err := tc.stor.scriptsStorage.newestScriptByAddr(scriptAddr)
	if err != nil {
		return scriptEstimation{}, false, errors.Wrapf(err, "failed to get newest script by addr %q", scriptAddr)
	}
	treeEstimation, err := ride.EstimateTree(tree, currentEstimatorVersion)
	if err != nil {
		return scriptEstimation{}, false, errors.Wrapf(err, "failed to estimate script by addr %q", scriptAddr)
	}
	return scriptEstimation{
		currentEstimatorVersion: currentEstimatorVersion,
		scriptIsEmpty:           false,
		estimation:              treeEstimation,
	}, true, nil
}

func (tc *transactionChecker) checkInvokeExpressionWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp")
	}
	isInvokeExpressionActivated, err := tc.stor.features.newestIsActivated(int16(settings.InvokeExpression))
	if err != nil {
		return out, err
	}
	if !isInvokeExpressionActivated {
		return out, errors.Errorf("can not use InvokeExpression before feature (%d) activation", settings.InvokeExpression)
	}
	if err := tc.checkFeeAsset(&tx.FeeAsset); err != nil {
		return out, err
	}

	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: nil}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	return out, nil
}

func (tc *transactionChecker) checkUpdateAssetInfoWithProofs(transaction proto.Transaction, info *checkerInfo) (out txCheckerData, err error) {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return out, errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return out, errs.Extend(err, "invalid timestamp")
	}
	if err := tc.checkFeeAsset(&tx.FeeAsset); err != nil {
		return out, errs.Extend(err, "bad fee asset")
	}
	rideV6Activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV6))
	if err != nil {
		return out, err
	}
	if tx.FeeAsset.Present && rideV6Activated {
		return out, errors.Errorf("sponsored assets are prohibited for UpdateAssetInfo after feature (%d) activation", settings.RideV6)
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets)
	if err != nil {
		return out, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return out, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.BlockV5))
	if err != nil {
		return out, err
	}
	if !activated {
		return out, errors.New("BlockV5 must be activated for UpdateAssetInfo transaction")
	}
	id := proto.AssetIDFromDigest(tx.AssetID)
	assetInfo, err := tc.stor.assets.newestAssetInfo(id)
	if err != nil {
		return out, errs.NewUnknownAsset(fmt.Sprintf("unknown asset %s: %v", tx.AssetID.String(), err))
	}
	if !bytes.Equal(assetInfo.Issuer[:], tx.SenderPK[:]) {
		return out, errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}
	lastUpdateHeight, err := tc.stor.assets.newestLastUpdateHeight(id)
	if err != nil {
		return out, errs.Extend(err, "failed to retrieve last update height")
	}
	updateAllowedAt := lastUpdateHeight + tc.settings.MinUpdateAssetInfoInterval
	blockHeight := info.blockchainHeight + 1
	if blockHeight < updateAllowedAt {
		return out, errs.NewAssetUpdateInterval(fmt.Sprintf("Can't update info of asset with id=%s before height %d, current height is %d", tx.AssetID.String(), updateAllowedAt, blockHeight))
	}
	return txCheckerData{smartAssets: smartAssets}, nil
}
