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

func stringKey(addr proto.Address, assetID []byte) string {
	return string(byteKey(addr, assetID))
}

// balanceDiff represents atomic balance change, which is a result of applying transaction.
// Transaction may produce one or more balance diffs, with single diff corresponding to certain address.
// Same diffs are then used to store balance changes by blocks in `diffStorage`.

/* Note About minBalance.
`minBalance` is sum of all negative diffs that were added to single transaction.
It is needed to check that total spend amount does not lead to negative balance.
For instance, if someone sent more money to himself than he ever had, minBalance would help to detect it.
See balanceDiff.addInsideTx() for more info.

When dealing with diffs at block level, minBalance takes the lowest minBalance among all transactions
for given key (address). But it also takes into account previous changes for this address, so overspend
will be checked like:
`balance_from_db` + `all_diffs_before` - `minBalance_for_thix_tx` > 0;
not just `balance_from_db` - `minBalance_for_thix_tx` > 0.
So we increase transactions' minBalances by `all_diffs_before` when adding them to block.
See balanceDiff.addInsideBlock() for more info.
*/

type balanceDiff struct {
	allowLeasedTransfer          bool
	updateMinIntermediateBalance bool
	// Min intermediate balance change.
	minBalance int64
	// Balance change.
	balance int64
	// LeaseIn change.
	leaseIn int64
	// LeaseOut change.
	leaseOut int64
	blockID  crypto.Signature
}

func newBalanceDiff(balance, leaseIn, leaseOut int64, updateMinIntermediateBalance bool) balanceDiff {
	diff := balanceDiff{
		updateMinIntermediateBalance: updateMinIntermediateBalance,
		balance:                      balance,
		leaseIn:                      leaseIn,
		leaseOut:                     leaseOut,
	}
	if updateMinIntermediateBalance {
		diff.minBalance = balance
	}
	return diff
}

// spendableBalanceDiff() returns the difference of spendable balance which given diff produces.
//func (diff *balanceDiff) spendableBalanceDiff() int64 {
//	return diff.balance - diff.leaseOut
//}

// applyTo() applies diff to the profile given.
// It does not change input profile, and returns the updated version.
// It also checks that it is legitimate to apply this diff to the profile (negative balances / overflows).
func (diff *balanceDiff) applyTo(profile *balanceProfile) (*balanceProfile, error) {
	// Check min intermediate change.
	minBalance, err := util.AddInt64(diff.minBalance, int64(profile.balance))
	if err != nil {
		return nil, errors.Errorf("failed to add balance and min balance diff: %v\n", err)
	}
	if minBalance < 0 {
		return nil, errors.Errorf("negative intermediate balance: balance is %d; diff is: %d\n", profile.balance, diff.minBalance)
	}
	// Chech main balance diff.
	newBalance, err := util.AddInt64(diff.balance, int64(profile.balance))
	if err != nil {
		return nil, errors.Errorf("failed to add balance and balance diff: %v\n", err)
	}
	if newBalance < 0 {
		return nil, errors.New("negative result balance")
	}
	newLeaseIn, err := util.AddInt64(diff.leaseIn, profile.leaseIn)
	if err != nil {
		return nil, errors.Errorf("failed to add leaseIn and leaseIn diff: %v\n", err)
	}
	// Check leasing change.
	newLeaseOut, err := util.AddInt64(diff.leaseOut, profile.leaseOut)
	if err != nil {
		return nil, errors.Errorf("failed to add leaseOut and leaseOut diff: %v\n", err)
	}
	if (newBalance-newLeaseOut < 0) && !diff.allowLeasedTransfer {
		return nil, errors.New("leased balance is greater than own")
	}
	// Create new profile.
	newProfile := &balanceProfile{}
	newProfile.balance = uint64(newBalance)
	newProfile.leaseIn = newLeaseIn
	newProfile.leaseOut = newLeaseOut
	return newProfile, nil
}

// applyToAssetBalance() is similar to applyTo() but does not deal with leasing.
func (diff *balanceDiff) applyToAssetBalance(balance uint64) (uint64, error) {
	// Check min intermediate change.
	minBalance, err := util.AddInt64(diff.minBalance, int64(balance))
	if err != nil {
		return 0, errors.Errorf("failed to add balance and min balance diff: %v\n", err)
	}
	if minBalance < 0 {
		return 0, errors.New("negative intermediate asset balance")
	}
	// Chech main balance diff.
	newBalance, err := util.AddInt64(diff.balance, int64(balance))
	if err != nil {
		return 0, errors.Errorf("failed to add balance and balance diff: %v\n", err)
	}
	if newBalance < 0 {
		return 0, errors.New("negative result balance")
	}
	return uint64(newBalance), nil
}

// addCommon() sums fields of any diffs.
func (diff *balanceDiff) addCommon(prevDiff *balanceDiff) error {
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

// addInsideTx() sums diffs inside single transaction.
// It also makes sure that minimum intermediate change gets updated properly.
func (diff *balanceDiff) addInsideTx(prevDiff *balanceDiff) error {
	if diff.updateMinIntermediateBalance {
		// If updateMinIntermediateBalance is true, this tx may produce negative intermediate changes.
		// It is only true for few tx types: Payment, Transfer, MassTransfer, InvokeScript.
		// Add current diff to previous minBalance (aka intermediate change) to get newMinBalance.
		newMinBalance, err := util.AddInt64(diff.balance, prevDiff.minBalance)
		if err != nil {
			return errors.Errorf("failed to update min balance diff: %v\n", err)
		}
		// Copy previous minBalance at first.
		diff.minBalance = prevDiff.minBalance
		if newMinBalance < diff.minBalance {
			// newMinBalance is less than previous minBalance, so we should use it.
			// This is basically always the case when diff.balance < 0.
			diff.minBalance = newMinBalance
		}
	}
	return diff.addCommon(prevDiff)
}

// addInsideBlock() sums diffs inside block.
// It also makes sure that minimum intermediate change gets updated properly.
func (diff *balanceDiff) addInsideBlock(prevDiff *balanceDiff) error {
	// Add previous cumulative diff to tx diff's minBalance to make it correspond to cumulative block diff.
	newMinBalance, err := util.AddInt64(diff.minBalance, prevDiff.balance)
	if err != nil {
		return errors.Errorf("failed to update min balance diff: %v\n", err)
	}
	// Copy previous minBalance at first.
	diff.minBalance = prevDiff.minBalance
	if newMinBalance < diff.minBalance {
		// newMinBalance is less than previous minBalance, so we should use it.
		diff.minBalance = newMinBalance
	}
	return diff.addCommon(prevDiff)
}

type differInfo struct {
	initialisation bool
	blockInfo      *proto.BlockInfo
}

func (i *differInfo) hasMiner() bool {
	return i.blockInfo.GeneratorPublicKey != (crypto.PublicKey{})
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

/* TODO: unused code, need to write tests if it is needed or otherwise remove it.
func (diff txDiff) keys() []string {
	keys := make([]string, 0, len(diff))
	for k := range diff {
		keys = append(keys, k)
	}
	return keys
}
*/

func (diff txDiff) appendBalanceDiffStr(key string, balanceDiff balanceDiff) error {
	if prevDiff, ok := diff[key]; ok {
		if err := balanceDiff.addInsideTx(&prevDiff); err != nil {
			return err
		}
		diff[key] = balanceDiff
	} else {
		// New balance diff for this key.
		diff[key] = balanceDiff
	}
	return nil
}

func (diff txDiff) appendBalanceDiff(key []byte, balanceDiff balanceDiff) error {
	return diff.appendBalanceDiffStr(string(key), balanceDiff)
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
	minerAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, info.blockInfo.GeneratorPublicKey)
	if err != nil {
		return err
	}
	minerKey := byteKey(minerAddr, feeAsset)
	minerBalanceDiff, err := td.calculateTxFee(fee)
	if err != nil {
		return err
	}
	if err := diff.appendBalanceDiff(minerKey, newBalanceDiff(int64(minerBalanceDiff), 0, 0, false)); err != nil {
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
	if err := diff.appendBalanceDiff(key.bytes(), newBalanceDiff(receiverBalanceDiff, 0, 0, false)); err != nil {
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
	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime {
		updateMinIntermediateBalance = true
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderKey := wavesBalanceKey{address: senderAddr}
	senderBalanceDiff := -int64(tx.Amount) - int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(senderBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	receiverKey := wavesBalanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func recipientToAddress(rcp proto.Recipient, aliases *aliases, filter bool) (*proto.Address, error) {
	if rcp.Address != nil {
		return rcp.Address, nil
	}
	recipientAddr, err := aliases.newestAddrByAlias(rcp.Alias.Alias, filter)
	if err != nil {
		return &proto.Address{}, errors.Errorf("invalid alias: %v\n", err)
	}
	return recipientAddr, nil
}

func (td *transactionDiffer) handleSponsorship(diff txDiff, fee uint64, feeAsset proto.OptionalAsset, info *differInfo) error {
	sponsorshipActivated, err := td.stor.sponsoredAssets.isSponsorshipActivated()
	if err != nil {
		return err
	}
	needToApplySponsorship := sponsorshipActivated && feeAsset.Present
	if !needToApplySponsorship {
		// No assets sponsorship.
		if info.hasMiner() {
			if err := td.minerPayout(diff, fee, info, feeAsset.ToID()); err != nil {
				return errors.Wrap(err, "failed to append miner payout")
			}
		}
		return nil
	}
	// Sponsorship logic.
	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime {
		updateMinIntermediateBalance = true
	}
	assetInfo, err := td.stor.assets.newestAssetInfo(feeAsset.ID, !info.initialisation)
	if err != nil {
		return err
	}
	// Append issuer asset balance diff.
	issuerAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, assetInfo.issuer)
	if err != nil {
		return err
	}
	issuerAssetKey := byteKey(issuerAddr, feeAsset.ID.Bytes())
	issuerAssetBalanceDiff := int64(fee)
	if err := diff.appendBalanceDiff(issuerAssetKey, newBalanceDiff(issuerAssetBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return err
	}
	// Append issuer Waves balance diff.
	feeInWaves, err := td.stor.sponsoredAssets.sponsoredAssetToWaves(feeAsset.ID, fee)
	if err != nil {
		return err
	}
	issuerWavesKey := (&wavesBalanceKey{issuerAddr}).bytes()
	issuerWavesBalanceDiff := -int64(feeInWaves)
	if err := diff.appendBalanceDiff(issuerWavesKey, newBalanceDiff(issuerWavesBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return err
	}
	// Miner payout using sponsorship.
	if info.hasMiner() {
		if err := td.minerPayout(diff, feeInWaves, info, nil); err != nil {
			return errors.Wrap(err, "failed to append miner payout")
		}
	}
	return nil
}

func (td *transactionDiffer) createDiffTransfer(tx *proto.Transfer, info *differInfo) (txDiff, error) {
	diff := newTxDiff()
	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime {
		updateMinIntermediateBalance = true
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := byteKey(senderAddr, tx.FeeAsset.ToID())
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	senderAmountKey := byteKey(senderAddr, tx.AmountAsset.ToID())
	senderAmountBalanceDiff := -int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderAmountKey, newBalanceDiff(senderAmountBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	recipientAddr, err := recipientToAddress(tx.Recipient, td.stor.aliases, !info.initialisation)
	if err != nil {
		return txDiff{}, err
	}
	receiverKey := byteKey(*recipientAddr, tx.AmountAsset.ToID())
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey, newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	if err := td.handleSponsorship(diff, tx.Fee, tx.FeeAsset, info); err != nil {
		return txDiff{}, err
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
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: assetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
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
	txID, err := tx.GetID()
	if err != nil {
		return txDiff{}, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	return td.createDiffIssue(&tx.Issue, txID, info)
}

func (td *transactionDiffer) createDiffIssueV2(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.IssueV2)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to IssueV2 transaction")
	}
	txID, err := tx.GetID()
	if err != nil {
		return txDiff{}, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	return td.createDiffIssue(&tx.Issue, txID, info)
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
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
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
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := -int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
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

func (td *transactionDiffer) orderFeeKey(address proto.Address, order proto.Order) []byte {
	switch o := order.(type) {
	case *proto.OrderV3:
		return byteKey(address, o.MatcherFeeAsset.ToID())
	default:
		k := wavesBalanceKey{address}
		return k.bytes()
	}
}

func (td *transactionDiffer) createDiffExchange(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to Exchange transaction")
	}
	diff := newTxDiff()
	buyOrder := tx.GetBuyOrderFull()
	sellOrder := tx.GetSellOrderFull()
	amountAsset := buyOrder.GetAssetPair().AmountAsset
	priceAsset := buyOrder.GetAssetPair().PriceAsset
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
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, sellOrder.GetSenderPK())
	if err != nil {
		return txDiff{}, err
	}
	senderPriceKey := byteKey(senderAddr, priceAsset.ToID())
	if err := diff.appendBalanceDiff(senderPriceKey, newBalanceDiff(priceDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	senderAmountKey := byteKey(senderAddr, amountAsset.ToID())
	if err := diff.appendBalanceDiff(senderAmountKey, newBalanceDiff(-amountDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	receiverAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, buyOrder.GetSenderPK())
	if err != nil {
		return txDiff{}, err
	}
	receiverPriceKey := byteKey(receiverAddr, priceAsset.ToID())
	if err := diff.appendBalanceDiff(receiverPriceKey, newBalanceDiff(-priceDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	receiverAmountKey := byteKey(receiverAddr, amountAsset.ToID())
	if err := diff.appendBalanceDiff(receiverAmountKey, newBalanceDiff(amountDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	// Fees
	matcherAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, buyOrder.GetMatcherPK())
	if err != nil {
		return txDiff{}, err
	}
	senderFee := int64(tx.GetSellMatcherFee())
	senderFeeKey := td.orderFeeKey(senderAddr, sellOrder)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(-senderFee, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	matcherFeeFromSenderKey := td.orderFeeKey(matcherAddr, sellOrder)
	if err := diff.appendBalanceDiff(matcherFeeFromSenderKey, newBalanceDiff(senderFee, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	receiverFee := int64(tx.GetBuyMatcherFee())
	receiverFeeKey := td.orderFeeKey(receiverAddr, buyOrder)
	if err := diff.appendBalanceDiff(receiverFeeKey, newBalanceDiff(-receiverFee, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	matcherFeeFromReceiverKey := td.orderFeeKey(matcherAddr, buyOrder)
	if err := diff.appendBalanceDiff(matcherFeeFromReceiverKey, newBalanceDiff(receiverFee, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	matcherKey := wavesBalanceKey{matcherAddr}
	matcherFee := int64(tx.GetFee())
	if err := diff.appendBalanceDiff(matcherKey.bytes(), newBalanceDiff(-matcherFee, 0, 0, false)); err != nil {
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
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(0, 0, senderLeaseOutDiff, false)); err != nil {
		return txDiff{}, err
	}
	senderFeeDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(senderFeeDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	recipientAddr, err := recipientToAddress(tx.Recipient, td.stor.aliases, !info.initialisation)
	if err != nil {
		return txDiff{}, err
	}
	receiverKey := wavesBalanceKey{address: *recipientAddr}
	receiverLeaseInDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), newBalanceDiff(0, receiverLeaseInDiff, 0, false)); err != nil {
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
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(0, 0, senderLeaseOutDiff, false)); err != nil {
		return txDiff{}, err
	}
	senderFeeDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(senderFeeDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	receiverKey := wavesBalanceKey{address: l.recipient}
	receiverLeaseInDiff := -int64(l.leaseAmount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), newBalanceDiff(0, receiverLeaseInDiff, 0, false)); err != nil {
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
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
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

func (td *transactionDiffer) createDiffMassTransferV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.MassTransferV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to MassTransferV1 transaction")
	}
	diff := newTxDiff()
	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime {
		updateMinIntermediateBalance = true
	}
	// Append sender fee diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	// Append amount diffs.
	senderAmountKey := byteKey(senderAddr, tx.Asset.ToID())
	for _, entry := range tx.Transfers {
		// Sender.
		senderAmountBalanceDiff := -int64(entry.Amount)
		if err := diff.appendBalanceDiff(senderAmountKey, newBalanceDiff(senderAmountBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txDiff{}, err
		}
		// Recipient.
		recipientAddr, err := recipientToAddress(entry.Recipient, td.stor.aliases, !info.initialisation)
		if err != nil {
			return txDiff{}, err
		}
		recipientKey := byteKey(*recipientAddr, tx.Asset.ToID())
		recipientBalanceDiff := int64(entry.Amount)
		if err := diff.appendBalanceDiff(recipientKey, newBalanceDiff(recipientBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txDiff{}, err
		}
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffDataV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.DataV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to DataV1 transaction")
	}
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffSponsorshipV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.SponsorshipV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to SponsorshipV1 transaction")
	}
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffSetScriptV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.SetScriptV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to SetScriptV1 transaction")
	}
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffSetAssetScriptV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.SetAssetScriptV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to SetAssetScriptV1 transaction")
	}
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (td *transactionDiffer) createDiffInvokeScriptV1(transaction proto.Transaction, info *differInfo) (txDiff, error) {
	tx, ok := transaction.(*proto.InvokeScriptV1)
	if !ok {
		return txDiff{}, errors.New("failed to convert interface to InvokeScriptV1 transaction")
	}
	updateMinIntermediateBalance := false
	noPayments := len(tx.Payments) == 0
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime && !noPayments {
		updateMinIntermediateBalance = true
	}
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := byteKey(senderAddr, tx.FeeAsset.ToID())
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	if err := td.handleSponsorship(diff, tx.Fee, tx.FeeAsset, info); err != nil {
		return txDiff{}, err
	}
	scriptAddr, err := recipientToAddress(tx.ScriptRecipient, td.stor.aliases, !info.initialisation)
	if err != nil {
		return txDiff{}, err
	}
	for _, payment := range tx.Payments {
		senderPaymentKey := byteKey(senderAddr, payment.Asset.ToID())
		senderBalanceDiff := -int64(payment.Amount)
		if err := diff.appendBalanceDiff(senderPaymentKey, newBalanceDiff(senderBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txDiff{}, err
		}
		receiverKey := byteKey(*scriptAddr, payment.Asset.ToID())
		receiverBalanceDiff := int64(payment.Amount)
		if err := diff.appendBalanceDiff(receiverKey, newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txDiff{}, err
		}
	}
	return diff, nil
}
