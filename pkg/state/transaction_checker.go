package state

import (
	"bytes"
	"math"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/estimation"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	KiB = 1024

	maxVerifierScriptSize = 8 * KiB
	maxContractScriptSize = 32 * KiB
)

type checkerInfo struct {
	initialisation   bool
	currentTimestamp uint64
	parentTimestamp  uint64
	blockID          crypto.Signature
	height           uint64
}

type transactionChecker struct {
	genesis  crypto.Signature
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

func newTransactionChecker(
	genesis crypto.Signature,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
) (*transactionChecker, error) {
	return &transactionChecker{genesis, stor, settings}, nil
}

func (tc *transactionChecker) scriptActivation(script *ast.Script) error {
	rideForDAppsActivated, err := tc.stor.features.isActivated(int16(settings.Ride4DApps))
	if err != nil {
		return err
	}
	if script.Version == 3 && !rideForDAppsActivated {
		return errors.New("Ride4DApps feature must be activated for scripts version 3")
	}
	if script.HasBlockV2 && !rideForDAppsActivated {
		return errors.New("Ride4DApps feature must be activated for scripts that have block version 2")
	}
	return nil
}

func (tc *transactionChecker) checkScriptComplexity(script *ast.Script, complexity estimation.Costs) error {
	var maxComplexity uint64
	switch script.Version {
	case 1, 2:
		maxComplexity = 2000
	case 3, 4:
		maxComplexity = 4000
	}
	complexityVal := complexity.Verifier
	if script.IsDapp() {
		complexityVal = complexity.DApp
	}
	if complexityVal > maxComplexity {
		return errors.Errorf(
			"script complexity %d exceeds maximum allowed complexity of %d\n",
			complexityVal,
			maxComplexity,
		)
	}
	return nil
}

func estimatorByScript(script *ast.Script) *estimation.Estimator {
	var variables map[string]ast.Expr
	var cat *estimation.Catalogue
	switch script.Version {
	case 1, 2:
		variables = ast.VariablesV2()
		cat = estimation.NewCatalogueV2()
	case 3:
		variables = ast.VariablesV3()
		cat = estimation.NewCatalogueV3()
	}
	return estimation.NewEstimator(1, cat, variables) //TODO: pass version 2 after BlockReward (feature 14) activation
}

type scriptInfo struct {
	complexity       estimation.Costs
	estimatorVersion byte
	isDApp           bool
}

func (tc *transactionChecker) checkScript(scriptBytes proto.Script) (*scriptInfo, error) {
	script, err := ast.BuildScript(reader.NewBytesReader(scriptBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to build ast from script bytes")
	}
	maxSize := maxVerifierScriptSize
	if script.IsDapp() {
		maxSize = maxContractScriptSize
	}
	if len(scriptBytes) > maxSize {
		return nil, errors.Errorf("script size %d is greater than limit of %d\n", len(scriptBytes), maxSize)
	}
	if err := tc.scriptActivation(script); err != nil {
		return nil, errors.Wrap(err, "script activation check failed")
	}
	estimator := estimatorByScript(script)
	complexity, err := estimator.Estimate(script)
	if err != nil {
		return nil, errors.Wrap(err, "failed to estimate script complexity")
	}
	if err := tc.checkScriptComplexity(script, complexity); err != nil {
		return nil, errors.Errorf("checkScriptComplexity(): %v\n", err)
	}
	return &scriptInfo{complexity, byte(estimator.Version), script.IsDapp()}, nil
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
	if !assets.feeAsset.Present {
		// Waves.
		return checkMinFeeWaves(tx, params)
	}
	return checkMinFeeAsset(tx, assets.feeAsset.ID, params)
}

func (tc *transactionChecker) checkFromFuture(timestamp uint64) bool {
	return timestamp > tc.settings.TxFromFutureCheckAfterTime
}

func (tc *transactionChecker) checkTimestamps(txTimestamp, blockTimestamp, prevBlockTimestamp uint64) error {
	if txTimestamp < prevBlockTimestamp-tc.settings.MaxTxTimeBackOffset {
		return errors.New("early transaction creation time")
	}
	if tc.checkFromFuture(blockTimestamp) && txTimestamp > blockTimestamp+tc.settings.MaxTxTimeForwardOffset {
		return errors.New("late transaction creation time")
	}
	return nil
}

func (tc *transactionChecker) checkAsset(asset *proto.OptionalAsset, initialisation bool) error {
	if !tc.stor.assets.newestAssetExists(*asset, !initialisation) {
		return errors.Errorf("unknown asset %s", asset.ID.String())
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
	isSponsored, err := tc.stor.sponsoredAssets.newestIsSponsored(asset.ID, !initialisation)
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
		hasScript, err := tc.stor.scriptsStorage.newestIsSmartAsset(asset.ID, !initialisation)
		if err != nil {
			return nil, err
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
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
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
		return nil, errors.Wrap(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	return nil, nil
}

func (tc *transactionChecker) checkTransfer(tx *proto.Transfer, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	if err := tc.checkAsset(&tx.AmountAsset, info.initialisation); err != nil {
		return err
	}
	if err := tc.checkFeeAsset(&tx.FeeAsset, info.initialisation); err != nil {
		return err
	}
	return nil
}

func (tc *transactionChecker) checkTransferV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.TransferV1)
	if !ok {
		return nil, errors.New("failed to convert interface to TransferV1 transaction")
	}
	allAssets := []proto.OptionalAsset{tx.AmountAsset}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkTransfer(&tx.Transfer, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkTransferV2(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.TransferV2)
	if !ok {
		return nil, errors.New("failed to convert interface to TransferV2 transaction")
	}
	allAssets := []proto.OptionalAsset{tx.AmountAsset}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: tx.FeeAsset, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
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

func (tc *transactionChecker) checkIssue(tx *proto.Issue, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	return nil
}

func (tc *transactionChecker) checkIssueV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.IssueV1)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueV1 transaction")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkIssue(&tx.Issue, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkIssueV2(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.IssueV2)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueV2 transaction")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkIssue(&tx.Issue, info); err != nil {
		return nil, err
	}
	if len(tx.Script) == 0 {
		// No script checks / actions are needed.
		return nil, nil
	}
	scriptInf, err := tc.checkScript(tx.Script)
	if err != nil {
		return nil, errors.Errorf("checkScript() tx %s: %v\n", tx.ID.String(), err)
	}
	assetID := *tx.ID
	r := &assetScriptComplexityRecord{
		complexity: scriptInf.complexity.Verifier,
		estimator:  scriptInf.estimatorVersion,
	}
	// Save complexity to storage so we won't have to calculate it every time the script is called.
	if err := tc.stor.scriptsComplexity.saveComplexityForAsset(assetID, r, info.blockID); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkReissue(tx *proto.Reissue, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	assetInfo, err := tc.stor.assets.newestAssetInfo(tx.AssetID, !info.initialisation)
	if err != nil {
		return err
	}
	if !bytes.Equal(assetInfo.issuer[:], tx.SenderPK[:]) {
		return errors.New("asset was issued by other address")
	}
	if info.currentTimestamp <= tc.settings.InvalidReissueInSameBlockUntilTime {
		// Due to bugs in existing blockchain it is valid to reissue non-reissueable asset in this time period.
		return nil
	}
	if (info.currentTimestamp >= tc.settings.ReissueBugWindowTimeStart) && (info.currentTimestamp <= tc.settings.ReissueBugWindowTimeEnd) {
		// Due to bugs in existing blockchain it is valid to reissue non-reissueable asset in this time period.
		return nil
	}
	if !assetInfo.reissuable {
		return errors.Errorf("attempt to reissue asset which is not reissuable")
	}
	// Check Int64 overflow.
	if (math.MaxInt64-int64(tx.Quantity) < assetInfo.quantity.Int64()) && (info.currentTimestamp >= tc.settings.ReissueBugWindowTimeEnd) {
		return errors.New("asset total value overflow")
	}
	return nil
}

func (tc *transactionChecker) checkReissueV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.ReissueV1)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueV1 transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkReissue(&tx.Reissue, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkReissueV2(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.ReissueV2)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueV2 transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
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
		return errors.Wrap(err, "invalid timestamp")
	}
	assetInfo, err := tc.stor.assets.newestAssetInfo(tx.AssetID, !info.initialisation)
	if err != nil {
		return err
	}
	burnAnyTokensEnabled, err := tc.stor.features.isActivated(int16(settings.BurnAnyTokens))
	if err != nil {
		return err
	}
	if !burnAnyTokensEnabled && !bytes.Equal(assetInfo.issuer[:], tx.SenderPK[:]) {
		return errors.New("asset was issued by other address")
	}
	return nil
}

func (tc *transactionChecker) checkBurnV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.BurnV1)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnV1 transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkBurn(&tx.Burn, info); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkBurnV2(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.BurnV2)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnV2 transaction")
	}
	allAssets := []proto.OptionalAsset{*proto.NewOptionalAssetFromDigest(tx.AssetID)}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
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

func (tc *transactionChecker) orderScriptedAccount(order proto.Order, initialisation bool) (bool, error) {
	sender, err := proto.NewAddressFromPublicKey(tc.settings.AddressSchemeCharacter, order.GetSenderPK())
	if err != nil {
		return false, err
	}
	return tc.stor.scriptsStorage.newestAccountHasVerifier(sender, !initialisation)
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
		return nil, errors.Wrap(err, "invalid timestamp")
	}
	if err := tc.checkEnoughVolume(tx.GetSellOrderFull(), tx.GetSellMatcherFee(), tx.GetAmount(), info); err != nil {
		return nil, errors.Wrap(err, "exchange transaction; sell order")
	}
	if err := tc.checkEnoughVolume(tx.GetBuyOrderFull(), tx.GetBuyMatcherFee(), tx.GetAmount(), info); err != nil {
		return nil, errors.Wrap(err, "exchange transaction; buy order")
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return nil, err
	}
	// Check assets.
	m := make(map[proto.OptionalAsset]struct{})
	m[sellOrder.AssetPair.AmountAsset] = struct{}{}
	m[sellOrder.AssetPair.PriceAsset] = struct{}{}
	// allAssets does not include matcher fee assets.
	allAssets := make([]proto.OptionalAsset, 0, len(m))
	for a := range m {
		allAssets = append(allAssets, a)
	}
	// Add matcher fee assets to map to checkAsset() them later.
	if so3, ok := tx.GetSellOrderFull().(*proto.OrderV3); ok {
		m[so3.MatcherFeeAsset] = struct{}{}
	}
	if bo3, ok := tx.GetBuyOrderFull().(*proto.OrderV3); ok {
		m[bo3.MatcherFeeAsset] = struct{}{}
	}
	for a := range m {
		if err := tc.checkAsset(&a, info.initialisation); err != nil {
			return nil, err
		}
	}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	smartAssetsActivated, err := tc.stor.features.isActivated(int16(settings.SmartAssets))
	if err != nil {
		return nil, err
	}
	if !smartAssetsActivated && (len(smartAssets) != 0) {
		return nil, errors.New("smart assets can't participate in Exchange because smart assets feature is disabled")
	}
	// Check smart accounts trading.
	smartTradingActivated, err := tc.stor.features.isActivated(int16(settings.SmartAccountTrading))
	if err != nil {
		return nil, err
	}
	soScriptedAccount, err := tc.orderScriptedAccount(tx.GetSellOrderFull(), info.initialisation)
	if err != nil {
		return nil, err
	}
	boScriptedAccount, err := tc.orderScriptedAccount(tx.GetBuyOrderFull(), info.initialisation)
	if err != nil {
		return nil, err
	}
	if boScriptedAccount && !smartTradingActivated {
		return nil, errors.New("buyer can't participate in Exchange because it is scripted, and smart trading is disabled")
	}
	if soScriptedAccount && !smartTradingActivated {
		return nil, errors.New("seller can't participate in Exchange because it is scripted, and smart trading is disabled")
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkExchangeV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.ExchangeV1)
	if !ok {
		return nil, errors.New("failed to convert interface to Payment transaction")
	}
	smartAssets, err := tc.checkExchange(tx, info)
	if err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkExchangeV2(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.ExchangeV2)
	if !ok {
		return nil, errors.New("failed to convert interface to ExchangeV2 transaction")
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccountTrading))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("SmartAccountsTrading feature must be activated for ExchangeV2 transactions")
	}
	smartAssets, err := tc.checkExchange(tx, info)
	if err != nil {
		return nil, err
	}
	if (tx.BuyOrder.GetVersion() != 3) && (tx.SellOrder.GetVersion() != 3) {
		return smartAssets, nil
	}
	activated, err = tc.stor.features.isActivated(int16(settings.OrderV3))
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
		return errors.Wrap(err, "invalid timestamp")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tc.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	var recipientAddr *proto.Address
	if tx.Recipient.Address == nil {
		recipientAddr, err = tc.stor.aliases.newestAddrByAlias(tx.Recipient.Alias.Alias, !info.initialisation)
		if err != nil {
			return errors.Errorf("invalid alias: %v", err)
		}
	} else {
		recipientAddr = tx.Recipient.Address
	}
	if senderAddr == *recipientAddr {
		return errors.New("trying to lease money to self")
	}
	return nil
}

func (tc *transactionChecker) checkLeaseV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.LeaseV1)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseV1 transaction")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkLease(&tx.Lease, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkLeaseV2(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.LeaseV2)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseV2 transaction")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
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
		return errors.Wrap(err, "invalid timestamp")
	}
	l, err := tc.stor.leases.newestLeasingInfo(tx.LeaseID, !info.initialisation)
	if err != nil {
		return errors.Wrap(err, "no leasing info found for this leaseID")
	}
	if !l.isActive && (info.currentTimestamp > tc.settings.AllowMultipleLeaseCancelUntilTime) {
		return errors.New("can not cancel lease which has already been cancelled")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tc.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	if (l.sender != senderAddr) && (info.currentTimestamp > tc.settings.AllowMultipleLeaseCancelUntilTime) {
		return errors.New("sender of LeaseCancel is not sender of corresponding Lease")
	}
	return nil
}

func (tc *transactionChecker) checkLeaseCancelV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.LeaseCancelV1)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelV1 transaction")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkLeaseCancel(&tx.LeaseCancel, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkLeaseCancelV2(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.LeaseCancelV2)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelV2 transaction")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
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
		return errors.Wrap(err, "invalid timestamp")
	}
	if (info.currentTimestamp >= tc.settings.StolenAliasesWindowTimeStart) && (info.currentTimestamp <= tc.settings.StolenAliasesWindowTimeEnd) {
		// At this period it is fine to steal aliases.
		return nil
	}
	// Check if alias is already taken.
	if tc.stor.aliases.exists(tx.Alias.Alias, !info.initialisation) {
		return errors.New("alias is already taken")
	}
	return nil
}

func (tc *transactionChecker) checkCreateAliasV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.CreateAliasV1)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasV1 transaction")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkCreateAlias(&tx.CreateAlias, info); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkCreateAliasV2(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.CreateAliasV2)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasV2 transaction")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
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

func (tc *transactionChecker) checkMassTransferV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.MassTransferV1)
	if !ok {
		return nil, errors.New("failed to convert interface to MassTransferV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errors.Wrap(err, "invalid timestamp")
	}
	allAssets := []proto.OptionalAsset{tx.Asset}
	smartAssets, err := tc.smartAssets(allAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.MassTransfer))
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

func (tc *transactionChecker) checkDataV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.DataV1)
	if !ok {
		return nil, errors.New("failed to convert interface to DataV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errors.Wrap(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.DataTransaction))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("Data transaction has not been activated yet")
	}
	return nil, nil
}

func (tc *transactionChecker) checkSponsorshipV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.SponsorshipV1)
	if !ok {
		return nil, errors.New("failed to convert interface to SponsorshipV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errors.Wrap(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("sponsorship has not been activated yet")
	}
	if err := tc.checkAsset(&proto.OptionalAsset{Present: false, ID: tx.AssetID}, info.initialisation); err != nil {
		return nil, err
	}
	assetInfo, err := tc.stor.assets.newestAssetInfo(tx.AssetID, !info.initialisation)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(assetInfo.issuer[:], tx.SenderPK[:]) {
		return nil, errors.New("asset was issued by other address")
	}
	isSmart, err := tc.stor.scriptsStorage.newestIsSmartAsset(tx.AssetID, !info.initialisation)
	if err != nil {
		return nil, err
	}
	if isSmart {
		return nil, errors.Errorf("can not sponsor smart asset %s", tx.AssetID.String())
	}
	return nil, nil
}

func (tc *transactionChecker) newAccountScriptComplexityRecordFromInfo(info *scriptInfo) *accountScriptComplexityRecord {
	r := &accountScriptComplexityRecord{
		verifierComplexity: info.complexity.Verifier,
		estimator:          info.estimatorVersion,
	}
	if info.isDApp {
		r.byFuncs = info.complexity.Functions
	}
	return r
}

func (tc *transactionChecker) checkSetScriptV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.SetScriptV1)
	if !ok {
		return nil, errors.New("failed to convert interface to SetScriptV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errors.Wrap(err, "invalid timestamp")
	}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if len(tx.Script) == 0 {
		// No script checks / actions are needed.
		return nil, nil
	}
	scriptInf, err := tc.checkScript(tx.Script)
	if err != nil {
		return nil, errors.Errorf("checkScript() tx %s: %v\n", tx.ID.String(), err)
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tc.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	// Save complexity to storage so we won't have to calculate it every time the script is called.
	if err := tc.stor.scriptsComplexity.saveComplexityForAddr(
		senderAddr,
		tc.newAccountScriptComplexityRecordFromInfo(scriptInf),
		info.blockID,
	); err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *transactionChecker) checkSetAssetScriptV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.SetAssetScriptV1)
	if !ok {
		return nil, errors.New("failed to convert interface to SetAssetScriptV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errors.Wrap(err, "invalid timestamp")
	}
	isSmartAsset, err := tc.stor.scriptsStorage.newestIsSmartAsset(tx.AssetID, !info.initialisation)
	if err != nil {
		return nil, err
	}
	if !isSmartAsset {
		return nil, errors.Errorf("asset %s is not smart, can not set script for it", tx.AssetID.String())
	}
	smartAssets := []crypto.Digest{tx.AssetID}
	assets := &txAssets{feeAsset: proto.OptionalAsset{Present: false}, smartAssets: smartAssets}
	if err := tc.checkFee(transaction, assets, info); err != nil {
		return nil, errors.Errorf("checkFee(): %v", err)
	}
	if len(tx.Script) == 0 {
		// No script checks / actions are needed.
		return nil, nil
	}
	scriptInf, err := tc.checkScript(tx.Script)
	if err != nil {
		return nil, errors.Errorf("checkScript() tx %s: %v\n", tx.ID.String(), err)
	}
	r := &assetScriptComplexityRecord{
		complexity: scriptInf.complexity.Verifier,
		estimator:  scriptInf.estimatorVersion,
	}
	// Save complexity to storage so we won't have to calculate it every time the script is called.
	if err := tc.stor.scriptsComplexity.saveComplexityForAsset(tx.AssetID, r, info.blockID); err != nil {
		return nil, err
	}
	return smartAssets, nil
}

func (tc *transactionChecker) checkInvokeScriptV1(transaction proto.Transaction, info *checkerInfo) ([]crypto.Digest, error) {
	tx, ok := transaction.(*proto.InvokeScriptV1)
	if !ok {
		return nil, errors.New("failed to convert interface to InvokeScriptV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return nil, errors.Wrap(err, "invalid timestamp")
	}
	activated, err := tc.stor.features.isActivated(int16(settings.Ride4DApps))
	if err != nil {
		return nil, err
	}
	if !activated {
		return nil, errors.New("can not use InvokeScript before Ride4DApps activation")
	}
	if err := tc.checkFeeAsset(&tx.FeeAsset, info.initialisation); err != nil {
		return nil, err
	}
	var paymentAssets []proto.OptionalAsset
	for _, payment := range tx.Payments {
		if err := tc.checkAsset(&payment.Asset, info.initialisation); err != nil {
			return nil, errors.Wrap(err, "bad payment asset")
		}
		paymentAssets = append(paymentAssets, payment.Asset)
	}
	// Only payment assets' scripts are called before invoke function and with
	// state that doesn't have any changes caused by this invokeScript tx yet.
	smartAssets, err := tc.smartAssets(paymentAssets, info.initialisation)
	if err != nil {
		return nil, err
	}
	return smartAssets, nil
}
