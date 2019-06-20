package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	// priceConstant is used for exchange calculations.
	priceConstant = 10e7
)

func byteKey(addr proto.Address, assetID []byte) []byte {
	if assetID == nil {
		k := wavesBalanceKey{addr}
		return k.bytes()
	}
	k := assetBalanceKey{addr, assetID}
	return k.bytes()
}

// balanceDiff represents atomic balance change, which is a result of applying transaction.
// Transaction may produce one or more balance diffs, with single diff corresponding to certain address.
type balanceDiff struct {
	allowLeasedTransfer bool
	// Balance change.
	balance int64
	// LeaseIn change.
	leaseIn int64
	// LeaseOut change.
	leaseOut int64
	// blockID of block this diff belongs to.
	blockID crypto.Signature
}

// spendableBalanceDiff() returns the difference of spendable balance which given diff produces.
func (diff *balanceDiff) spendableBalanceDiff() int64 {
	return diff.balance - diff.leaseOut
}

// applyTo() applies diff to the profile given.
// It does not change input profile, and returns the updated version.
// It also checks that it is legitimate to apply this diff to the profile (negative balances / overflows).
func (diff *balanceDiff) applyTo(profile *balanceProfile) (*balanceProfile, error) {
	newBalance, err := util.AddInt64(diff.balance, int64(profile.balance))
	if err != nil {
		return nil, errors.Errorf("failed to add balance and balance diff: %v\n", err)
	}
	newLeaseIn, err := util.AddInt64(diff.leaseIn, profile.leaseIn)
	if err != nil {
		return nil, errors.Errorf("failed to add leaseIn and leaseIn diff: %v\n", err)
	}
	newLeaseOut, err := util.AddInt64(diff.leaseOut, profile.leaseOut)
	if err != nil {
		return nil, errors.Errorf("failed to add leaseOut and leaseOut diff: %v\n", err)
	}
	if newBalance < 0 {
		return nil, errors.New("negative result balance")
	}
	if (newBalance-newLeaseOut < 0) && !diff.allowLeasedTransfer {
		return nil, errors.New("leased balance is greater than own")
	}
	newProfile := &balanceProfile{}
	newProfile.balance = uint64(newBalance)
	newProfile.leaseIn = newLeaseIn
	newProfile.leaseOut = newLeaseOut
	return newProfile, nil
}

// add() sums two diffs, checking for overflows.
// It does not change the input diff.
func (diff *balanceDiff) add(prevDiff *balanceDiff) error {
	var err error
	if diff.balance, err = util.AddInt64(diff.balance, prevDiff.balance); err != nil {
		return errors.Errorf("failed to add balance diffs: %v\n", err)
	}
	if diff.leaseIn, err = util.AddInt64(diff.leaseIn, prevDiff.leaseIn); err != nil {
		return errors.Errorf("failed to add LeaseIn diffs: %v\n", err)
	}
	if diff.leaseOut, err = util.AddInt64(diff.leaseOut, prevDiff.leaseOut); err != nil {
		return errors.Errorf("failed to add LeaseOut diffs: %v\n", err)
	}
	return nil
}

type differInfo struct {
	initialisation bool
	minerPK        crypto.PublicKey
}

func (i *differInfo) hasMiner() bool {
	return i.minerPK != (crypto.PublicKey{})
}

type txDiff map[string]balanceDiff

func newTxDiff() txDiff {
	return make(txDiff)
}

func (diff txDiff) balancesChanges() []balanceChanges {
	changes := make([]balanceChanges, 0, len(diff))
	for key, diff := range diff {
		change := newBalanceChanges([]byte(key), diff)
		changes = append(changes, *change)
	}
	return changes
}

func (diff txDiff) keys() []string {
	keys := make([]string, 0, len(diff))
	for k := range diff {
		keys = append(keys, k)
	}
	return keys
}

func (diff txDiff) appendBalanceDiff(key []byte, balanceDiff balanceDiff) error {
	keyStr := string(key)
	if prevDiff, ok := diff[keyStr]; ok {
		if err := balanceDiff.add(&prevDiff); err != nil {
			return err
		}
		diff[keyStr] = balanceDiff
	} else {
		// New balance diff for this key.
		diff[keyStr] = balanceDiff
	}
	return nil
}

type transactionDiffer struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

func newTransactionDiffer(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*transactionDiffer, error) {
	return &transactionDiffer{stor, settings}, nil
}

func (td *transactionDiffer) calculateTxFee(txFee uint64) (uint64, error) {
	ngActivated, err := td.stor.features.isActivated(int16(settings.NG))
	if err != nil {
		return 0, err
	}
	return calculateCurrentBlockTxFee(txFee, ngActivated), nil
}

// minerPayout adds current fee part of given tx to txDiff.
func (td *transactionDiffer) minerPayout(diff txDiff, fee uint64, info *differInfo, feeAsset []byte) error {
	minerAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, info.minerPK)
	if err != nil {
		return err
	}
	minerKey := byteKey(minerAddr, feeAsset)
	minerBalanceDiff, err := td.calculateTxFee(fee)
	if err != nil {
		return err
	}
	if err := diff.appendBalanceDiff(minerKey, balanceDiff{balance: int64(minerBalanceDiff)}); err != nil {
		return err
	}
	return nil
}

func (td *transactionDiffer) createDiffGenesis(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.Genesis)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to Genesis transaction")
	}
	diff := newTxDiff()
	key := wavesBalanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(key.bytes(), balanceDiff{balance: receiverBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffPayment(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.Payment)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to Payment transaction")
	}
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderKey := wavesBalanceKey{address: senderAddr}
	senderBalanceDiff := -int64(tx.Amount) - int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), balanceDiff{balance: senderBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	receiverKey := wavesBalanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), balanceDiff{balance: receiverBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffTransfer(tx *proto.Transfer, info *differInfo) (txDiff, error) {
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := byteKey(senderAddr, tx.FeeAsset.ToID())
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, balanceDiff{balance: senderFeeBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	senderAmountKey := byteKey(senderAddr, tx.AmountAsset.ToID())
	senderAmountBalanceDiff := -int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderAmountKey, balanceDiff{balance: senderAmountBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	recipientAddr := &proto.Address{}
	if tx.Recipient.Address == nil {
		recipientAddr, err = td.stor.aliases.newestAddrByAlias(tx.Recipient.Alias.Alias, !info.initialisation)
		if err != nil {
			return txDiff{}, errors.Errorf("invalid alias: %v\n", err)
		}
	} else {
		recipientAddr = tx.Recipient.Address
	}
	receiverKey := byteKey(*recipientAddr, tx.AmountAsset.ToID())
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey, balanceDiff{balance: receiverBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, tx.FeeAsset.ToID()); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffTransferV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.TransferV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to TransferV1 transaction")
	}
	return td.createDiffTransfer(&tx.Transfer, info)
}

func (td *transactionDiffer) createDiffTransferV2(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.TransferV2)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to TransferV2 transaction")
	}
	return td.createDiffTransfer(&tx.Transfer, info)
}

func (td *transactionDiffer) createDiffIssue(tx *proto.Issue, id []byte, info *differInfo) (txDiff, error) {
	diff := newTxDiff()
	assetID, err := crypto.NewDigestFromBytes(id)
	if err != nil {
		return txDiff{}, err
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: assetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), balanceDiff{balance: senderAssetBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffIssueV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.IssueV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to IssueV1 transaction")
	}
	return td.createDiffIssue(&tx.Issue, tx.GetID(), info)
}

func (td *transactionDiffer) createDiffIssueV2(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.IssueV2)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to IssueV2 transaction")
	}
	return td.createDiffIssue(&tx.Issue, tx.GetID(), info)
}

func (td *transactionDiffer) createDiffReissue(tx *proto.Reissue, info *differInfo) (txDiff, error) {
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), balanceDiff{balance: senderAssetBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffReissueV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.ReissueV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to ReissueV1 transaction")
	}
	return td.createDiffReissue(&tx.Reissue, info)
}

func (td *transactionDiffer) createDiffReissueV2(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.ReissueV2)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to ReissueV2 transaction")
	}
	return td.createDiffReissue(&tx.Reissue, info)
}

func (td *transactionDiffer) createDiffBurn(tx *proto.Burn, info *differInfo) (txDiff, error) {
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := -int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), balanceDiff{balance: senderAssetBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffBurnV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.BurnV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to BurnV1 transaction")
	}
	return td.createDiffBurn(&tx.Burn, info)
}

func (td *transactionDiffer) createDiffBurnV2(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.BurnV2)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to BurnV2 transaction")
	}
	return td.createDiffBurn(&tx.Burn, info)
}

func (td *transactionDiffer) createDiffExchange(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to Exchange transaction")
	}
	diff := newTxDiff()
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return txDiff{}, err
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return txDiff{}, err
	}
	// Perform exchange.
	var val, amount, price big.Int
	priceConst := big.NewInt(priceConstant)
	amount.SetUint64(tx.GetAmount())
	price.SetUint64(tx.GetPrice())
	val.Mul(&amount, &price)
	val.Quo(&val, priceConst)
	if !val.IsInt64() {
		return txDiff{}, errors.New("price * amount exceeds MaxInt64")
	}
	priceDiff := val.Int64()
	amountDiff := int64(tx.GetAmount())
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, sellOrder.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderPriceKey := byteKey(senderAddr, sellOrder.AssetPair.PriceAsset.ToID())
	if err := diff.appendBalanceDiff(senderPriceKey, balanceDiff{balance: priceDiff}); err != nil {
		return txDiff{}, err
	}
	senderAmountKey := byteKey(senderAddr, sellOrder.AssetPair.AmountAsset.ToID())
	if err := diff.appendBalanceDiff(senderAmountKey, balanceDiff{balance: -amountDiff}); err != nil {
		return txDiff{}, err
	}
	senderFeeKey := wavesBalanceKey{senderAddr}
	senderFeeDiff := -int64(tx.GetSellMatcherFee())
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), balanceDiff{balance: senderFeeDiff}); err != nil {
		return txDiff{}, err
	}
	receiverAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, buyOrder.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	receiverPriceKey := byteKey(receiverAddr, sellOrder.AssetPair.PriceAsset.ToID())
	if err := diff.appendBalanceDiff(receiverPriceKey, balanceDiff{balance: -priceDiff}); err != nil {
		return txDiff{}, err
	}
	receiverAmountKey := byteKey(receiverAddr, sellOrder.AssetPair.AmountAsset.ToID())
	if err := diff.appendBalanceDiff(receiverAmountKey, balanceDiff{balance: amountDiff}); err != nil {
		return txDiff{}, err
	}
	receiverFeeKey := wavesBalanceKey{receiverAddr}
	receiverFeeDiff := -int64(tx.GetBuyMatcherFee())
	if err := diff.appendBalanceDiff(receiverFeeKey.bytes(), balanceDiff{balance: receiverFeeDiff}); err != nil {
		return txDiff{}, err
	}
	// Update matcher.
	matcherAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, buyOrder.MatcherPK)
	if err != nil {
		return txDiff{}, err
	}
	matcherKey := wavesBalanceKey{matcherAddr}
	matcherFee, err := util.AddInt64(int64(tx.GetBuyMatcherFee()), int64(tx.GetSellMatcherFee()))
	if err != nil {
		return txDiff{}, err
	}
	matcherBalanceDiff := matcherFee - int64(tx.GetFee())
	if err := diff.appendBalanceDiff(matcherKey.bytes(), balanceDiff{balance: matcherBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.GetFee(), info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffLease(tx *proto.Lease, id *crypto.Digest, info *differInfo) (txDiff, error) {
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderKey := wavesBalanceKey{address: senderAddr}
	senderLeaseOutDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderKey.bytes(), balanceDiff{leaseOut: senderLeaseOutDiff}); err != nil {
		return txDiff{}, err
	}
	senderFeeDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), balanceDiff{balance: senderFeeDiff}); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	recipientAddr := &proto.Address{}
	if tx.Recipient.Address == nil {
		recipientAddr, err = td.stor.aliases.newestAddrByAlias(tx.Recipient.Alias.Alias, !info.initialisation)
		if err != nil {
			return txDiff{}, errors.Errorf("invalid alias: %v\n", err)
		}
	} else {
		recipientAddr = tx.Recipient.Address
	}
	receiverKey := wavesBalanceKey{address: *recipientAddr}
	receiverLeaseInDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), balanceDiff{leaseIn: receiverLeaseInDiff}); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffLeaseV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.LeaseV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to LeaseV1 transaction")
	}
	return td.createDiffLease(&tx.Lease, tx.ID, info)
}

func (td *transactionDiffer) createDiffLeaseV2(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.LeaseV2)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to LeaseV2 transaction")
	}
	return td.createDiffLease(&tx.Lease, tx.ID, info)
}

func (td *transactionDiffer) createDiffLeaseCancel(tx *proto.LeaseCancel, info *differInfo) (txDiff, error) {
	diff := newTxDiff()
	l, err := td.stor.leases.newestLeasingInfo(tx.LeaseID, !info.initialisation)
	if err != nil {
		return txDiff{}, errors.Wrap(err, "no leasing info found for this leaseID")
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderKey := wavesBalanceKey{address: senderAddr}
	senderLeaseOutDiff := -int64(l.leaseAmount)
	if err := diff.appendBalanceDiff(senderKey.bytes(), balanceDiff{leaseOut: senderLeaseOutDiff}); err != nil {
		return txDiff{}, err
	}
	senderFeeDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), balanceDiff{balance: senderFeeDiff}); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	receiverKey := wavesBalanceKey{address: l.recipient}
	receiverLeaseInDiff := -int64(l.leaseAmount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), balanceDiff{leaseIn: receiverLeaseInDiff}); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffLeaseCancelV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.LeaseCancelV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to LeaseCancelV1 transaction")
	}
	return td.createDiffLeaseCancel(&tx.LeaseCancel, info)
}

func (td *transactionDiffer) createDiffLeaseCancelV2(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.LeaseCancelV2)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to LeaseCancelV2 transaction")
	}
	return td.createDiffLeaseCancel(&tx.LeaseCancel, info)
}

func (td *transactionDiffer) createDiffCreateAlias(tx *proto.CreateAlias, info *differInfo) (txDiff, error) {
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff}); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffCreateAliasV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.CreateAliasV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to CreateAliasV1 transaction")
	}
	return td.createDiffCreateAlias(&tx.CreateAlias, info)
}

func (td *transactionDiffer) createDiffCreateAliasV2(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.CreateAliasV2)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to CreateAliasV2 transaction")
	}
	return td.createDiffCreateAlias(&tx.CreateAlias, info)
}
