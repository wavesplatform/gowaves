package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
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
	if _, err := tc.stor.assets.newestAssetRecord(asset.ID, !initialisation); err != nil {
		return errors.New("unknown asset")
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
	return nil
}

func (tc *transactionChecker) checkTransfer(tx *proto.Transfer, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	if err := tc.checkAsset(&tx.AmountAsset, info.initialisation); err != nil {
		return err
	}
	if err := tc.checkAsset(&tx.FeeAsset, info.initialisation); err != nil {
		return err
	}
	return nil
}

func (tc *transactionChecker) checkTransferV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.TransferV1)
	if !ok {
		return errors.New("failed to convert interface to TransferV1 transaction")
	}
	return tc.checkTransfer(&tx.Transfer, info)
}

func (tc *transactionChecker) checkTransferV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.TransferV2)
	if !ok {
		return errors.New("failed to convert interface to TransferV2 transaction")
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
	return tc.checkIssue(&tx.Issue, info)
}

func (tc *transactionChecker) checkIssueV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.IssueV2)
	if !ok {
		return errors.New("failed to convert interface to IssueV2 transaction")
	}
	return tc.checkIssue(&tx.Issue, info)
}

func (tc *transactionChecker) checkReissue(tx *proto.Reissue, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	// Check if it's "legal" to modify given asset.
	record, err := tc.stor.assets.newestAssetRecord(tx.AssetID, !info.initialisation)
	if err != nil {
		return err
	}
	if (info.currentTimestamp > tc.settings.InvalidReissueInSameBlockUntilTime) && !record.reissuable {
		return errors.New("attempt to reissue asset which is not reissuable")
	}
	return nil
}

func (tc *transactionChecker) checkReissueV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.ReissueV1)
	if !ok {
		return errors.New("failed to convert interface to ReissueV1 transaction")
	}
	return tc.checkReissue(&tx.Reissue, info)
}

func (tc *transactionChecker) checkReissueV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.ReissueV2)
	if !ok {
		return errors.New("failed to convert interface to ReissueV2 transaction")
	}
	return tc.checkReissue(&tx.Reissue, info)
}

func (tc *transactionChecker) checkBurn(tx *proto.Burn, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	return nil
}

func (tc *transactionChecker) checkBurnV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.BurnV1)
	if !ok {
		return errors.New("failed to convert interface to BurnV1 transaction")
	}
	return tc.checkBurn(&tx.Burn, info)
}

func (tc *transactionChecker) checkBurnV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.BurnV2)
	if !ok {
		return errors.New("failed to convert interface to BurnV2 transaction")
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

func (tc *transactionChecker) checkLease(tx *proto.Lease, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tc.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	recipientAddr := &proto.Address{}
	if tx.Recipient.Address == nil {
		recipientAddr, err = tc.stor.aliases.newestAddrByAlias(tx.Recipient.Alias.Alias, !info.initialisation)
		if err != nil {
			return errors.Errorf("invalid alias: %v\n", err)
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
	return tc.checkLease(&tx.Lease, info)
}

func (tc *transactionChecker) checkLeaseV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.LeaseV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseV2 transaction")
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
	return tc.checkLeaseCancel(&tx.LeaseCancel, info)
}

func (tc *transactionChecker) checkLeaseCancelV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.LeaseCancelV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelV2 transaction")
	}
	return tc.checkLeaseCancel(&tx.LeaseCancel, info)
}

func (tc *transactionChecker) checkCreateAlias(tx *proto.CreateAlias, info *checkerInfo) error {
	if err := tc.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return errors.Wrap(err, "invalid timestamp")
	}
	// Check if alias already taken.
	if _, err := tc.stor.aliases.newestAddrByAlias(tx.Alias.Alias, !info.initialisation); err == nil {
		return errors.New("alias is already taken")
	}
	return nil
}

func (tc *transactionChecker) checkCreateAliasV1(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.CreateAliasV1)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV1 transaction")
	}
	return tc.checkCreateAlias(&tx.CreateAlias, info)
}

func (tc *transactionChecker) checkCreateAliasV2(transaction proto.Transaction, info *checkerInfo) error {
	tx, ok := transaction.(*proto.CreateAliasV2)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV2 transaction")
	}
	return tc.checkCreateAlias(&tx.CreateAlias, info)
}
