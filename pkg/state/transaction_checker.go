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
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	KiB = 1024
	MiB = 1024 * KiB

	maxVerifierScriptSize = 8 * KiB
	maxContractScriptSize = 32 * KiB

	maxEstimatorVersion = 3
)

type checkerInfo struct {
	initialisation   bool
	currentTimestamp uint64
	parentTimestamp  uint64
	blockID          proto.BlockID
	blockVersion     proto.BlockVersion
	height           uint64
}

func (i *checkerInfo) estimatorVersion() int {
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

func (tc *transactionChecker) scriptActivation(libVersion int, hasBlockV2 bool) error {
	rideForDAppsActivated, err := tc.stor.features.newestIsActivated(int16(settings.Ride4DApps))
	if err != nil {
		return errs.Extend(err, "transactionChecker scriptActivation isActivated")
	}
	blockV5Activated, err := tc.stor.features.newestIsActivated(int16(settings.BlockV5))
	if err != nil {
		return err
	}
	rideV5Activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV5))
	if err != nil {
		return err
	}
	rideV6Activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV6))
	if err != nil {
		return err
	}
	if libVersion == 3 && !rideForDAppsActivated {
		return errors.New("Ride4DApps feature must be activated for scripts version 3")
	}
	if hasBlockV2 && !rideForDAppsActivated {
		return errors.New("Ride4DApps feature must be activated for scripts that have block version 2")
	}
	if libVersion == 4 && !blockV5Activated {
		return errors.New("MultiPaymentInvokeScript feature must be activated for scripts version 4")
	}
	if libVersion == 5 && !rideV5Activated {
		return errors.New("RideV5 feature must be activated for scripts version 5")
	}
	if libVersion == 6 && !rideV6Activated {
		return errors.New("RideV6 feature must be activated for scripts version 6")
	}
	return nil
}

func (tc *transactionChecker) checkScriptComplexity(tree *ride.Tree, estimation ride.TreeEstimation, reducedVerifierComplexity bool) error {
	/*
		| Script Type                        | Max complexity before BlockV5 | Max complexity after BlockV5 |
		| ---------------------------------- | ----------------------------- | ---------------------------- |
		| Account / DApp Verifier V1, V2     | 2000                          | 2000                         |
		| Account / DApp Verifier V3, V4, V5 | 4000                          | 2000                         |
		| Asset Verifier V1, V2              | 2000                          | 2000                         |
		| Asset Verifier V3, V4, V5          | 4000                          | 4000                         |
		| DApp Callable V1, V2               | 2000                          | 2000                         |
		| DApp Callable V3, V4               | 4000                          | 4000                         |
		| DApp Callable V5                   | 10000                         | 10000                        |
	*/
	var maxCallableComplexity, maxVerifierComplexity int
	switch tree.LibVersion {
	case 1, 2:
		maxCallableComplexity = MaxCallableScriptComplexityV12
		maxVerifierComplexity = MaxVerifierScriptComplexityReduced
	case 3, 4:
		maxCallableComplexity = MaxCallableScriptComplexityV34
		maxVerifierComplexity = MaxVerifierScriptComplexity
	case 5:
		maxCallableComplexity = MaxCallableScriptComplexityV5
		maxVerifierComplexity = MaxVerifierScriptComplexity
	}
	if reducedVerifierComplexity {
		maxVerifierComplexity = MaxVerifierScriptComplexityReduced
	}
	if !tree.IsDApp() { // Expression (simple) script, has only verifier.
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

func (tc *transactionChecker) checkScript(script proto.Script, estimatorVersion int, reducedVerifierComplexity, expandEstimations bool) (map[int]ride.TreeEstimation, error) {
	tree, err := ride.Parse(script)
	if err != nil {
		return nil, errs.Extend(err, "failed to build AST")
	}
	maxSize := maxVerifierScriptSize
	if tree.IsDApp() {
		maxSize = maxContractScriptSize
	}
	if len(script) > maxSize {
		return nil, errors.Errorf("script size %d is greater than limit of %d", len(script), maxSize)
	}
	if err := tc.scriptActivation(tree.LibVersion, tree.HasBlockV2); err != nil {
		return nil, errs.Extend(err, "script activation check failed")
	}

	estimations := make(map[int]ride.TreeEstimation)
	maxVersion := maxEstimatorVersion
	if !expandEstimations {
		maxVersion = estimatorVersion
	}
	for ev := estimatorVersion; ev <= maxVersion; ev++ {
		est, err := ride.EstimateTree(tree, ev)
		if err != nil {
			return nil, errs.Extend(err, "failed to estimate script complexity")
		}
		estimations[ev] = est
	}
	if err := tc.checkScriptComplexity(tree, estimations[estimatorVersion], reducedVerifierComplexity); err != nil {
		return nil, errors.Wrap(err, "failed to check script complexity")
	}
	return estimations, nil
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
		stor:           tc.stor,
		settings:       tc.settings,
		initialisation: info.initialisation,
		txAssets:       assets,
	}

	isRideV5Activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV5))
	if err != nil {
		return errors.Errorf("failed to check if feature is was activated, %v", err)
	}
	if !assets.feeAsset.Present {
		// Waves.
		return checkMinFeeWaves(tx, params, isRideV5Activated, info.estimatorVersion())
	}
	return checkMinFeeAsset(tx, assets.feeAsset.ID, params, isRideV5Activated, info.estimatorVersion())
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

func (tc *transactionChecker) checkAsset(asset *proto.OptionalAsset, initialisation bool) error {
	if !tc.stor.assets.newestAssetExists(*asset, !initialisation) {
		return errs.NewUnknownAsset(fmt.Sprintf("unknown asset %s", asset.String()))
	}
	return nil
}

func (tc *transactionChecker) checkFeeAsset(asset *proto.OptionalAsset, initialisation bool) error {
	if err := tc.checkAsset(asset, initialisation); err != nil {
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
	isSponsored, err := tc.stor.sponsoredAssets.newestIsSponsored(proto.AssetIDFromDigest(asset.ID), !initialisation)
	if err != nil {
		return err
	}
	if !isSponsored {
		return errors.Errorf("asset %s is not sponsored and can not be used to pay fees", asset.ID.String())
	}
	return nil
}

func (tc *transactionChecker) smartAssets(assets []proto.OptionalAsset, initialisation bool) ([]crypto.Digest, error) {
	var smartAssets []crypto.Digest
	for _, asset := range assets {
		if !asset.Present {
			// Waves can not be scripted.
			continue
		}
		hasScript, err := tc.stor.scriptsStorage.newestIsSmartAsset(proto.AssetIDFromDigest(asset.ID), !initialisation)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check newestIsSmartAsset for asset %q", asset.String())
		}
		if hasScript {
			smartAssets = append(smartAssets, asset.ID)
		}
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkGenesis(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	if info.blockID != tc.genesis {
		return nil, errors.New("genesis transaction inside of non-genesis block")
	}
	if !info.initialisation {
		return nil, errors.New("genesis transaction in non-initialisation mode")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkPayment(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.Payment)
	if !ok {
		return nil, errors.New("failed to convert interface to Payment transaction")
	}
	if info.height >= tc.settings.BlockVersion3AfterHeight {
		return nil, errors.Errorf("Payment transaction is deprecated after height %d", tc.settings.BlockVersion3AfterHeight)
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errs.Extend(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkTransfer(tx *proto.Transfer, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	if err := tc.checkAsset(&tx.AmountAsset, info.initialisation); err != nil {
		return err
	}
	if err := tc.checkFeeAsset(&tx.FeeAsset, info.initialisation); err != nil {
		return err
	}
	return nil
}

func (tc *transactionChecker) checkTransferWithSig(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to TransferWithSig transaction")
	}
	allAssets := []proto.OptionalAsset{tx.AmountAsset}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	if err := tc.checkTransfer(&tx.Transfer, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkEthereumTransactionWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	metamaskActivated, err := tc.stor.features.newestIsActivated(int16(settings.RideV6))
	if err != nil {
		return nil, errors.Errorf("failed to check whether feature %d was activated: %v", settings.RideV6, err)
	}
	if !metamaskActivated {
		return nil, errors.Errorf("failed to handle ethereum transaction: feature %d has not been activated yet", settings.RideV6)
	}

	tx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return nil, errors.New("failed to cast 'Transaction' interface to 'EthereumTransaction' type")
	}
	switch kind := tx.TxKind.(type) {
	case *proto.EthereumTransferWavesTxKind:
		// check fee
		if tx.GetFee() < proto.EthereumTransferMinFee {
			return nil, errors.Errorf("the fee for ethereum transfer waves tx is not enough, min fee is %d, got %d", proto.EthereumTransferMinFee, tx.GetFee())
		}

		// check if the amount is 0
		if tx.Value() == nil {
			return nil, errors.New("the amount of ethereum transfer waves is 0, which is forbidden")
		}
		res := new(big.Int).Div(tx.Value(), big.NewInt(int64(proto.DiffEthWaves)))
		if ok := res.IsInt64(); !ok {
			return nil, errors.Errorf("failed to convert amount from ethreum transaction (big int) to int64. value is %s", tx.Value().String())
		}
		if res.Int64() == 0 {
			return nil, errors.New("the amount of ethereum transfer waves is 0, which is forbidden")
		}

		return nil, nil
	case *proto.EthereumTransferAssetsErc20TxKind:
		// check fee
		minFee := proto.EthereumTransferMinFee

		if kind.Arguments.Amount == 0 {
			return nil, errors.New("the amount of ethereum transfer assets is 0, which is forbidden")
		}

		isSmart, err := tc.stor.scriptsStorage.newestIsSmartAsset(proto.AssetIDFromDigest(kind.Asset.ID), true)
		if err != nil {
			return nil, errors.Errorf("failed to get asset info, %v", err)
		}
		if isSmart {
			minFee += proto.EthereumScriptedAssetMinFee
		}

		if tx.GetFee() < minFee {
			return nil, errors.Errorf("the fee for ethereum transfer assets tx is not enough, min fee is %d, got %d", proto.EthereumTransferMinFee, tx.GetFee())
		}

		allAssets := []proto.OptionalAsset{kind.Asset}
		smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
		if err != nil {
			return nil, err
		}

		return smartAssets, nil
	case *proto.EthereumInvokeScriptTxKind:
		// check fee
		minFee := proto.EthereumInvokeMinFee

		if err := tc.checkTimestamps(tx.GetTimestamp(), info.currentTimestamp, info.parentTimestamp); err != nil {
			return nil, errs.Extend(err, "invalid timestamp")
		}
		decodedData := tx.TxKind.DecodedData()
		abiPayments := decodedData.Payments

		if len(abiPayments) > 10 {
			return nil, errors.New("no more than ten payments is allowed since RideV5 activation")
		}
		paymentAssets := make([]proto.OptionalAsset, 0, len(abiPayments))
		for _, payment := range abiPayments {
			var optionalAsset proto.OptionalAsset
			if payment.PresentAssetID {
				isSmart, err := tc.stor.scriptsStorage.newestIsSmartAsset(proto.AssetIDFromDigest(payment.AssetID), true)
				if err != nil {
					return nil, err
				}
				if isSmart {
					minFee += proto.EthereumScriptedAssetMinFee
				}

				optionalAsset = *proto.NewOptionalAssetFromDigest(payment.AssetID)
				if err := tc.checkAsset(&optionalAsset, info.initialisation); err != nil {
					return nil, errs.Extend(err, "bad payment asset")
				}
			} else {
				// we don't have to check WAVES asset because it can't be scripted and always exists
				optionalAsset = proto.NewOptionalAssetWaves()
			}
			paymentAssets = append(paymentAssets, optionalAsset)
		}

		if tx.GetFee() < minFee {
			return nil, errors.Errorf("the fee for ethereum invoke tx is not enough, min fee is %d, got %d", proto.EthereumInvokeMinFee, tx.GetFee())
		}

		smartAssets, err := tc.smartAssets(paymentAssets, info.initialisation)
		if err != nil {
			return nil, err
		}

		return smartAssets, nil

	default:
		return nil, errors.New("failed to check ethereum transaction, wrong kind of tx")
	}
}

func (tc *transactionChecker) checkTransferWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to TransferWithProofs transaction")
	}
	allAssets := []proto.OptionalAsset{tx.AmountAsset}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkTransfer(&tx.Transfer, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
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

func (tc *transactionChecker) checkIssueWithSig(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	if err := tc.checkIssue(&tx.Issue, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkIssueWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	if err := tc.checkIssue(&tx.Issue, info); err != nil {
		return nil, err
	}
	if len(tx.Script) == 0 {
		// No script checks / actions are needed.
		return nil, nil
	}
	// For asset scripts do not reduce verifier complexity and only one estimation is required
	currentEstimatorVersion := info.estimatorVersion()
	estimations, err := tc.checkScript(tx.Script, currentEstimatorVersion, false, false)
	if err != nil {
		return nil, errors.Errorf("checkScript() tx %s: %v", tx.ID.String(), err)
	}
	assetID := *tx.ID
	// Save complexities to storage so we won't have to calculate it every time the script is called.
	complexity, ok := estimations[currentEstimatorVersion]
	if !ok {
		return nil, errors.Errorf("failed to calculate asset script complexity by estimator version %d", currentEstimatorVersion)
	}
	if err := tc.stor.scriptsComplexity.saveComplexitiesForAsset(assetID, complexity, info.blockID); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkReissue(tx *proto.Reissue, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	assetInfo, err := tc.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(tx.AssetID), !info.initialisation)
	if err != nil {
		return err
	}
	if !bytes.Equal(assetInfo.issuer[:], tx.SenderPK[:]) {
		return errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}
	if info.currentTimestamp <= tc.settings.InvalidReissueInSameBlockUntilTime {
		// Due to bugs in existing blockchain it is valid to reissue non-reissuable asset in this time period.
		return nil
	}
	if (info.currentTimestamp >= tc.settings.ReissueBugWindowTimeStart) && (info.currentTimestamp <= tc.settings.ReissueBugWindowTimeEnd) {
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

func (tc *transactionChecker) checkReissueWithSig(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	if err := tc.checkReissue(&tx.Reissue, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkReissueWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkReissue(&tx.Reissue, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkBurn(tx *proto.Burn, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	assetInfo, err := tc.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(tx.AssetID), !info.initialisation)
	if err != nil {
		return err
	}
	// Verify sender.
	burnAnyTokensEnabled, err := tc.stor.features.newestIsActivated(int16(settings.BurnAnyTokens))
	if err != nil {
		return err
	}
	if !burnAnyTokensEnabled && !bytes.Equal(assetInfo.issuer[:], tx.SenderPK[:]) {
		return errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}
	// Check burn amount.
	quantityDiff := big.NewInt(int64(tx.Amount))
	if assetInfo.quantity.Cmp(quantityDiff) == -1 {
		return errs.NewAccountBalanceError("trying to burn more assets than exist at all")
	}
	return nil
}

func (tc *transactionChecker) checkBurnWithSig(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	if err := tc.checkBurn(&tx.Burn, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkBurnWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}

	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkBurn(&tx.Burn, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

// orderScriptedAccount checks that sender account is a scripted account.
// This method works for both proto.EthereumAddress and proto.WavesAddress.
// Note that only real proto.WavesAddress account can have a verifier.
func (tc *transactionChecker) orderScriptedAccount(order proto.Order, initialisation bool) (bool, error) {
	senderAddr, err := order.GetSender(tc.settings.AddressSchemeCharacter)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get sender for order")
	}
	// senderWavesAddr needs only for newestAccountHasVerifier check
	senderWavesAddr, err := senderAddr.ToWavesAddress(tc.settings.AddressSchemeCharacter)
	if err != nil {
		return false, errors.Wrapf(err, "failed to transform (%T) address type to WavesAddress type", senderAddr)
	}
	return tc.stor.scriptsStorage.newestAccountHasVerifier(senderWavesAddr, !initialisation)
}

func (tc *transactionChecker) checkEnoughVolume(order proto.Order, newFee, newAmount uint64, info *checkerInfo) error {
	orderId, err := order.GetID()
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
	filledAmount, err := tc.stor.ordersVolumes.newestFilledAmount(orderId, !info.initialisation)
	if err != nil {
		return err
	}
	if fullAmount-newAmount < filledAmount {
		return errors.New("order amount volume is overflowed")
	}
	filledFee, err := tc.stor.ordersVolumes.newestFilledFee(orderId, !info.initialisation)
	if err != nil {
		return err
	}
	if fullFee-newFee < filledFee {
		return errors.New("order fee volume is overflowed")
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
	so, err := tx.GetSellOrder()
	if err != nil {
		return nil, errs.Extend(err, "sell order")
	}
	if err := tc.checkEnoughVolume(so, tx.GetSellMatcherFee(), tx.GetAmount(), info); err != nil {
		return nil, errs.Extend(err, "exchange transaction; sell order")
	}
	bo, err := tx.GetBuyOrder()
	if err != nil {
		return nil, errs.Extend(err, "buy order")
	}
	if err := tc.checkEnoughVolume(bo, tx.GetBuyMatcherFee(), tx.GetAmount(), info); err != nil {
		return nil, errs.Extend(err, "exchange transaction; buy order")
	}
	// Check assets.
	m := make(map[proto.OptionalAsset]struct{})
	m[so.GetAssetPair().AmountAsset] = struct{}{}
	m[so.GetAssetPair().PriceAsset] = struct{}{}
	// Add matcher fee assets to map to checkAsset() them later.
	if o2v3, ok := tx.GetOrder2().(*proto.OrderV3); ok {
		m[o2v3.MatcherFeeAsset] = struct{}{}
	}
	if o1v3, ok := tx.GetOrder1().(*proto.OrderV3); ok {
		m[o1v3.MatcherFeeAsset] = struct{}{}
	}
	if o2v4, ok := tx.GetOrder2().(*proto.OrderV4); ok {
		m[o2v4.MatcherFeeAsset] = struct{}{}
	}
	if o2v4, ok := tx.GetOrder1().(*proto.OrderV4); ok {
		m[o2v4.MatcherFeeAsset] = struct{}{}
	}
	for a := range m {
		if err := tc.checkAsset(&a, info.initialisation); err != nil {
			return nil, errs.Extend(err, "Assets should be issued before they can be traded")
		}
	}
	allAssets := make([]proto.OptionalAsset, 0, len(m))
	for a := range m {
		allAssets = append(allAssets, a)
	}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	smartAssetsActivated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAssets))
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
	o1ScriptedAccount, err := tc.orderScriptedAccount(tx.GetOrder1(), info.initialisation)
	if err != nil {
		return nil, err
	}
	o2ScriptedAccount, err := tc.orderScriptedAccount(tx.GetOrder2(), info.initialisation)
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

func (tc *transactionChecker) checkExchangeWithSig(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.ExchangeWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to Payment transaction")
	}
	smartAssets, err := tc.checkExchange(tx, info)
	if err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkExchangeWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.ExchangeWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to ExchangeWithProofs transaction")
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccountTrading))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("SmartAccountsTrading feature must be activated for ExchangeWithProofs transactions")
	}
	smartAssets, err := tc.checkExchange(tx, info)
	if err != nil {
		return nil, err
	}
	if (tx.Order1.GetVersion() != 3) && (tx.Order2.GetVersion() != 3) {
		return smartAssets, nil
	}
	activated, err = tc.stor.features.newestIsActivated(int16(settings.OrderV3))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("OrderV3 feature must be activated for Exchange transactions with Order version 3")
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkLease(tx *proto.Lease, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tc.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	var recipientAddr *proto.WavesAddress
	if tx.Recipient.Address == nil {
		recipientAddr, err = tc.stor.aliases.newestAddrByAlias(tx.Recipient.Alias.Alias, !info.initialisation)
		if err != nil {
			return errors.Errorf("invalid alias: %v", err)
		}
	} else {
		recipientAddr = tx.Recipient.Address
	}
	if senderAddr == *recipientAddr {
		return errs.NewToSelf("trying to lease money to self")
	}
	return nil
}

func (tc *transactionChecker) checkLeaseWithSig(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	if err := tc.checkLease(&tx.Lease, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkLeaseWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkLease(&tx.Lease, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkLeaseCancel(tx *proto.LeaseCancel, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errs.Extend(err, "invalid timestamp")
	}
	l, err := tc.stor.leases.newestLeasingInfo(tx.LeaseID, !info.initialisation)
	if err != nil {
		return errs.Extend(err, "no leasing info found for this leaseID")
	}
	if !l.isActive() && (info.currentTimestamp > tc.settings.AllowMultipleLeaseCancelUntilTime) {
		return errs.NewTxValidationError("Reason: Cannot cancel already cancelled lease")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tc.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	if (l.Sender != senderAddr) && (info.currentTimestamp > tc.settings.AllowMultipleLeaseCancelUntilTime) {
		return errs.NewTxValidationError("LeaseTransaction was leased by other sender")
	}
	return nil
}

func (tc *transactionChecker) checkLeaseCancelWithSig(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	if err := tc.checkLeaseCancel(&tx.LeaseCancel, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkLeaseCancelWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkLeaseCancel(&tx.LeaseCancel, info); err != nil {
		return nil, err
	}
	return nil, nil
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
	if tc.stor.aliases.exists(tx.Alias.Alias, !info.initialisation) {
		return errs.NewAliasTaken("alias is already taken")
	}
	return nil
}

func (tc *transactionChecker) checkCreateAliasWithSig(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	if err := tc.checkCreateAlias(&tx.CreateAlias, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkCreateAliasWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("SmartAccounts feature has not been activated yet")
	}
	if err := tc.checkCreateAlias(&tx.CreateAlias, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkMassTransferWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to MassTransferWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errs.Extend(err, "invalid timestamp")
	}
	allAssets := []proto.OptionalAsset{tx.Asset}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.MassTransfer))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("MassTransfer transaction has not been activated yet")
	}
	if err := tc.checkAsset(&tx.Asset, info.initialisation); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkDataWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to DataWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errs.Extend(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.DataTransaction))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("Data transaction has not been activated yet")
	}
	return nil, nil
}

func (tc *transactionChecker) checkSponsorshipWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errs.Extend(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("sponsorship has not been activated yet")
	}
	if err := tc.checkAsset(proto.NewOptionalAssetFromDigest(tx.AssetID), info.initialisation); err != nil {
		return nil, err
	}
	id := proto.AssetIDFromDigest(tx.AssetID)
	assetInfo, err := tc.stor.assets.newestAssetInfo(id, !info.initialisation)
	if err != nil {
		return nil, err
	}
	if assetInfo.issuer != tx.SenderPK {
		return nil, errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}
	isSmart, err := tc.stor.scriptsStorage.newestIsSmartAsset(id, !info.initialisation)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check newestIsSmartAsset for asset %q", tx.AssetID.String())
	}
	if isSmart {
		return nil, errors.Errorf("can not sponsor smart asset %s", tx.AssetID.String())
	}
	return nil, nil
}

func (tc *transactionChecker) checkSetScriptWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errs.Extend(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves()}

	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}

	addr, err := proto.NewAddressFromPublicKey(tc.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	if len(tx.Script) == 0 {
		// No script checks / actions are needed.
		if err := tc.stor.scriptsComplexity.saveComplexitiesForAddr(addr, nil, info.blockID); err != nil {
			return nil, err
		}
		return nil, nil
	}
	estimations, err := tc.checkScript(tx.Script, info.estimatorVersion(), info.blockVersion == proto.ProtobufBlockVersion, true)
	if err != nil {
		return nil, errors.Errorf("checkScript() tx %s: %v", tx.ID.String(), err)
	}
	// Save complexity to storage so we won't have to calculate it every time the script is called.
	if err := tc.stor.scriptsComplexity.saveComplexitiesForAddr(addr, estimations, info.blockID); err != nil {
		return nil, err
	}

	return nil, nil
}

func (tc *transactionChecker) checkSetAssetScriptWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errs.Extend(err, "invalid timestamp")
	}
	id := proto.AssetIDFromDigest(tx.AssetID)
	assetInfo, err := tc.stor.assets.newestAssetInfo(id, !info.initialisation)
	if err != nil {
		return nil, err
	}

	smartAssets := []crypto.Digest{tx.AssetID}
	assets := &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errs.Extend(err, "check fee")
	}

	if !bytes.Equal(assetInfo.issuer[:], tx.SenderPK[:]) {
		return nil, errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}

	isSmart, err := tc.stor.scriptsStorage.newestIsSmartAsset(id, !info.initialisation)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check newestIsSmartAsset for asset %q", tx.AssetID.String())
	}
	if len(tx.Script) == 0 {
		return nil, errs.NewTxValidationError("Cannot set empty script")
	}
	if !isSmart {
		return nil, errs.NewTxValidationError("Reason: Cannot set script on an asset issued without a script. Referenced assetId not found")
	}
	currentEstimatorVersion := info.estimatorVersion()
	// Do not reduce verifier complexity for asset scripts and only one estimation is required
	estimations, err := tc.checkScript(tx.Script, currentEstimatorVersion, false, false)
	if err != nil {
		return nil, errors.Errorf("checkScript() tx %s: %v", tx.ID.String(), err)
	}
	// Save complexity to storage so we won't have to calculate it every time the script is called.
	estimation, ok := estimations[currentEstimatorVersion]
	if !ok {
		return nil, errors.Errorf("failed to calculate asset script complexity by estimator version %d", currentEstimatorVersion)
	}
	if err := tc.stor.scriptsComplexity.saveComplexitiesForAsset(tx.AssetID, estimation, info.blockID); err != nil {
		return nil, errs.Extend(err, "saveComplexityForAsset")
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkInvokeScriptWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errs.Extend(err, "invalid timestamp")
	}
	ride4DAppsActivated, err := tc.stor.features.newestIsActivated(int16(settings.Ride4DApps))
	if err != nil {
		return nil, err
	}
	if !ride4DAppsActivated {
		return nil, errors.New("can not use InvokeScript before Ride4DApps activation")
	}
	if err := tc.checkFeeAsset(&tx.FeeAsset, info.initialisation); err != nil {
		return nil, err
	}
	multiPaymentActivated, err := tc.stor.features.newestIsActivated(int16(settings.BlockV5))
	if err != nil {
		return nil, err
	}
	rideV5activated, err := tc.stor.features.newestIsActivated(int16(settings.RideV5))
	if err != nil {
		return nil, err
	}
	l := len(tx.Payments)
	switch {
	case l > 1 && !multiPaymentActivated && !rideV5activated:
		return nil, errors.New("no more than one payment is allowed")
	case l > 2 && multiPaymentActivated && !rideV5activated:
		return nil, errors.New("no more than two payments is allowed")
	case l > 10 && rideV5activated:
		return nil, errors.New("no more than ten payments is allowed since RideV5 activation")
	}
	var paymentAssets []proto.OptionalAsset
	for _, payment := range tx.Payments {
		if err := tc.checkAsset(&payment.Asset, info.initialisation); err != nil {
			return nil, errs.Extend(err, "bad payment asset")
		}
		paymentAssets = append(paymentAssets, payment.Asset)
	}
	smartAssets, err := tc.smartAssets(paymentAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkUpdateAssetInfoWithProofs(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errs.Extend(err, "invalid timestamp")
	}
	if err := tc.checkFeeAsset(&tx.FeeAsset, info.initialisation); err != nil {
		return nil, errs.Extend(err, "bad fee asset")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, err
	}
	activated, err := tc.stor.features.newestIsActivated(int16(settings.BlockV5))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("BlockV5 must be activated for UpdateAssetInfo transaction")
	}
	id := proto.AssetIDFromDigest(tx.AssetID)
	assetInfo, err := tc.stor.assets.newestAssetInfo(id, !info.initialisation)
	if err != nil {
		return nil, errs.NewUnknownAsset(fmt.Sprintf("unknown asset %s", tx.AssetID.String()))
	}
	if !bytes.Equal(assetInfo.issuer[:], tx.SenderPK[:]) {
		return nil, errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
	}
	lastUpdateHeight, err := tc.stor.assets.newestLastUpdateHeight(id, !info.initialisation)
	if err != nil {
		return nil, errs.Extend(err, "failed to retrieve last update height")
	}
	updateAllowedAt := lastUpdateHeight + tc.settings.MinUpdateAssetInfoInterval
	blockHeight := info.height + 1
	if blockHeight < updateAllowedAt {
		return nil, errs.NewAssetUpdateInterval(fmt.Sprintf("Can't update info of asset with id=%s before height %d, current height is %d", tx.AssetID.String(), updateAllowedAt, blockHeight))
	}
	return smartAssets, nil
}
