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
	//maxContractScriptSize = 32 * KiB
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

func (tc *transactionChecker) scriptActivation(script ast.Script) error {
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

func (tc *transactionChecker) checkScriptComplexity(script ast.Script, complexity int64) error {
	var maxComplexity int64
	switch script.Version {
	case 1, 2:
		maxComplexity = 2000
	case 3, 4:
		maxComplexity = 4000
	}
	if complexity > maxComplexity {
		return errors.Errorf(
			"script complexity %d exceeds maximum allowed complexity of %d\n",
			complexity,
			maxComplexity,
		)
	}
	return nil
}

func (tc *transactionChecker) estimatorByScript(script ast.Script) *estimation.EstimatorV1 {
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
	return estimation.NewEstimatorV1(cat, variables)
}

func (tc *transactionChecker) checkScript(scriptBytes proto.Script) error {
	if len(scriptBytes) == 0 {
		// Empty script is always valid.
		return nil
	}
	// TODO: use RIDE package to check script size depending on whether it's dApp or simple Verifier.
	if len(scriptBytes) > maxVerifierScriptSize {
		return errors.Errorf("script size %d is greater than limit of %d\n", len(scriptBytes), maxVerifierScriptSize)
	}
	script, err := ast.BuildAst(reader.NewBytesReader(scriptBytes))
	if err != nil {
		return errors.Wrap(err, "failed to build ast from script bytes")
	}
	if err := tc.scriptActivation(script); err != nil {
		return errors.Wrap(err, "script activation check failed")
	}
	estimator := tc.estimatorByScript(script)
	complexity, err := estimator.Estimate(script)
	if err != nil {
		return errors.Wrap(err, "failed to estimate script complexity")
	}
	if err := tc.checkScriptComplexity(script, complexity); err != nil {
		return errors.Errorf("checkScriptComplexity(): %v\n", err)
	}
	return nil
}

func (tc *transactionChecker) checkFee(tx proto.Transaction, feeAsset proto.OptionalAsset, info *checkerInfo) error {
	sponsorshipActivated, err := tc.stor.sponsoredAssets.isSponsorshipActivated()
	if err != nil {
		return err
	}
	if !sponsorshipActivated {
		// Sponsorship is not yet activated.
		return nil
	}
	params := &feeValidationParams{stor: tc.stor, settings: tc.settings, initialisation: info.initialisation}
	if !feeAsset.Present {
		// Waves.
		return checkMinFeeWaves(tx, params)
	}
	return checkMinFeeAsset(tx, feeAsset.ID, params)
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
	if !asset.Present {
		// Waves always valid.
		return nil
	}
	if _, err := tc.stor.assets.newestAssetInfo(asset.ID, !initialisation); err != nil {
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

func (tc *transactionChecker) checkGenesis(transaction proto.Transaction, info *checkerInfo) error {
	if info.blockID != tc.genesis {
		return errors.New("genesis transaction inside of non-genesis block")
	}
	if !info.initialisation {
		return errors.New("genesis transaction in non-initialisation mode")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	return nil
}

func (tc *transactionChecker) checkPayment(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.Payment)
	if !ok {
		return errors.New("failed to convert interface to Payment transaction")
	}
	if info.height >= tc.settings.BlockVersion3AfterHeight {
		return errors.Errorf("Payment transaction is deprecated after height %d", tc.settings.BlockVersion3AfterHeight)
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	return nil
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

func (tc *transactionChecker) checkTransferV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.TransferV1)
	if !ok {
		return errors.New("failed to convert interface to TransferV1 transaction")
	}
	if err := tc.checkFee(transaction, tx.FeeAsset, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	return tc.checkTransfer(&tx.Transfer, info)
}

func (tc *transactionChecker) checkTransferV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.TransferV2)
	if !ok {
		return errors.New("failed to convert interface to TransferV2 transaction")
	}
	if err := tc.checkFee(transaction, tx.FeeAsset, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("SmartAccounts feature has not been activated yet")
	}
	return tc.checkTransfer(&tx.Transfer, info)
}

func (tc *transactionChecker) checkIssue(tx *proto.Issue, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	return nil
}

func (tc *transactionChecker) checkIssueV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.IssueV1)
	if !ok {
		return errors.New("failed to convert interface to IssueV1 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	return tc.checkIssue(&tx.Issue, info)
}

func (tc *transactionChecker) checkIssueV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.IssueV2)
	if !ok {
		return errors.New("failed to convert interface to IssueV2 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkScript(tx.Script); err != nil {
		return errors.Errorf("checkScript(): %v\n", err)
	}
	return tc.checkIssue(&tx.Issue, info)
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

func (tc *transactionChecker) checkReissueV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.ReissueV1)
	if !ok {
		return errors.New("failed to convert interface to ReissueV1 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	return tc.checkReissue(&tx.Reissue, info)
}

func (tc *transactionChecker) checkReissueV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.ReissueV2)
	if !ok {
		return errors.New("failed to convert interface to ReissueV2 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("SmartAccounts feature has not been activated yet")
	}
	return tc.checkReissue(&tx.Reissue, info)
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

func (tc *transactionChecker) checkBurnV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.BurnV1)
	if !ok {
		return errors.New("failed to convert interface to BurnV1 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	return tc.checkBurn(&tx.Burn, info)
}

func (tc *transactionChecker) checkBurnV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.BurnV2)
	if !ok {
		return errors.New("failed to convert interface to BurnV2 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("SmartAccounts feature has not been activated yet")
	}
	return tc.checkBurn(&tx.Burn, info)
}

func (tc *transactionChecker) checkExchange(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return errors.New("failed to convert interface to Exchange transaction")
	}
	if err := tc.checkTimestamps(tx.GetTimestamp(), info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return err
	}
	// Check assets.
	if err := tc.checkAsset(&sellOrder.AssetPair.AmountAsset, info.initialisation); err != nil {
		return err
	}
	if err := tc.checkAsset(&sellOrder.AssetPair.PriceAsset, info.initialisation); err != nil {
		return err
	}
	return nil
}

func (tc *transactionChecker) checkExchangeV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.ExchangeV1)
	if !ok {
		return errors.New("failed to convert interface to Payment transaction")
	}
	return tc.checkExchange(tx, info)
}

func (tc *transactionChecker) checkExchangeV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.ExchangeV2)
	if !ok {
		return errors.New("failed to convert interface to ExchangeV2 transaction")
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccountTrading))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("SmartAccountsTrading feature must be activated for ExchangeV2 transactions")
	}
	if (tx.BuyOrder.GetVersion() != 3) && (tx.SellOrder.GetVersion() != 3) {
		return nil
	}
	activated, err = tc.stor.features.isActivated(int16(settings.OrderV3))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("OrderV3 feature must be activated for Exchange transactions with Order version 3")
	}
	return tc.checkExchange(tx, info)
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

func (tc *transactionChecker) checkLeaseV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.LeaseV1)
	if !ok {
		return errors.New("failed to convert interface to LeaseV1 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	return tc.checkLease(&tx.Lease, info)
}

func (tc *transactionChecker) checkLeaseV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.LeaseV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseV2 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("SmartAccounts feature has not been activated yet")
	}
	return tc.checkLease(&tx.Lease, info)
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

func (tc *transactionChecker) checkLeaseCancelV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.LeaseCancelV1)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelV1 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	return tc.checkLeaseCancel(&tx.LeaseCancel, info)
}

func (tc *transactionChecker) checkLeaseCancelV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.LeaseCancelV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelV2 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("SmartAccounts feature has not been activated yet")
	}
	return tc.checkLeaseCancel(&tx.LeaseCancel, info)
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

func (tc *transactionChecker) checkCreateAliasV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.CreateAliasV1)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV1 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	return tc.checkCreateAlias(&tx.CreateAlias, info)
}

func (tc *transactionChecker) checkCreateAliasV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.CreateAliasV2)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV2 transaction")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.SmartAccounts))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("SmartAccounts feature has not been activated yet")
	}
	return tc.checkCreateAlias(&tx.CreateAlias, info)
}

func (tc *transactionChecker) checkMassTransferV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.MassTransferV1)
	if !ok {
		return errors.New("failed to convert interface to MassTransferV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.MassTransfer))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("MassTransfer transaction has not been activated yet")
	}
	if err := tc.checkAsset(&tx.Asset, info.initialisation); err != nil {
		return err
	}
	return nil
}

func (tc *transactionChecker) checkDataV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.DataV1)
	if !ok {
		return errors.New("failed to convert interface to DataV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.DataTransaction))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("Data transaction has not been activated yet")
	}
	return nil
}

func (tc *transactionChecker) checkSponsorshipV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.SponsorshipV1)
	if !ok {
		return errors.New("failed to convert interface to SponsorshipV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	activated, err := tc.stor.features.isActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return err
	}
	if !activated {
		return errors.New("sponsorship has not been activated yet")
	}
	if err := tc.checkAsset(&proto.OptionalAsset{Present: false, ID: tx.AssetID}, info.initialisation); err != nil {
		return err
	}
	assetInfo, err := tc.stor.assets.newestAssetInfo(tx.AssetID, !info.initialisation)
	if err != nil {
		return err
	}
	if !bytes.Equal(assetInfo.issuer[:], tx.SenderPK[:]) {
		return errors.New("asset was issued by other address")
	}
	return nil
}

func (tc *transactionChecker) checkSetScriptV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.SetScriptV1)
	if !ok {
		return errors.New("failed to convert interface to SetScriptV1 transaction")
	}
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	if err := tc.checkFee(transaction, proto.OptionalAsset{Present: false}, info); err != nil {
		return errors.Errorf("checkFee(): %v", err)
	}
	if err := tc.checkScript(tx.Script); err != nil {
		return errors.Errorf("checkScript(): %v\n", err)
	}
	return nil
}
