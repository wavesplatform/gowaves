package state

import (
	"math/big"

	"github.com/ericlagergren/decimal"
	"github.com/ericlagergren/decimal/math"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func byteKey(addrID proto.AddressID, asset proto.OptionalAsset) []byte {
	if !asset.Present {
		k := wavesBalanceKey{addrID}
		return k.bytes()
	}
	k := assetBalanceKey{addrID, proto.AssetIDFromDigest(asset.ID)}
	return k.bytes()
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
`balance_from_db` + `all_diffs_before` - `minBalance_for_this_tx` > 0;
not just `balance_from_db` - `minBalance_for_this_tx` > 0.
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
	blockID  proto.BlockID
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
	minBalance, err := common.AddInt64(diff.minBalance, int64(profile.balance))
	if err != nil {
		return nil, errors.Errorf("failed to add balance and min balance diff: %v\n", err)
	}
	if minBalance < 0 {
		return nil, errors.Errorf(
			"negative intermediate balance (Attempt to transfer unavailable funds): balance is %d; diff is: %d\n",
			profile.balance,
			diff.minBalance,
		)
	}
	// Check main balance diff.
	newBalance, err := common.AddInt64(diff.balance, int64(profile.balance))
	if err != nil {
		return nil, errors.Errorf("failed to add balance and balance diff: %v\n", err)
	}
	if newBalance < 0 {
		return nil, errors.New("negative result balance (Attempt to transfer unavailable funds)")
	}
	newLeaseIn, err := common.AddInt64(diff.leaseIn, profile.leaseIn)
	if err != nil {
		return nil, errors.Errorf("failed to add leaseIn and leaseIn diff: %v\n", err)
	}
	// Check leasing change.
	newLeaseOut, err := common.AddInt64(diff.leaseOut, profile.leaseOut)
	if err != nil {
		return nil, errors.Errorf("failed to add leaseOut and leaseOut diff: %v\n", err)
	}
	if (newBalance < newLeaseOut) && !diff.allowLeasedTransfer {
		return nil, errs.NewTxValidationError("Reason: Cannot lease more than own")
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
	minBalance, err := common.AddInt64(diff.minBalance, int64(balance))
	if err != nil {
		return 0, errors.Errorf("failed to add balance and min balance diff: %v\n", err)
	}
	if minBalance < 0 {
		return 0, errors.New("negative intermediate asset balance (Attempt to transfer unavailable funds)")
	}
	// Check main balance diff.
	newBalance, err := common.AddInt64(diff.balance, int64(balance))
	if err != nil {
		return 0, errors.Errorf("failed to add balance and balance diff: %v\n", err)
	}
	if newBalance < 0 {
		return 0, errors.New("negative result balance (Attempt to transfer unavailable funds)")
	}
	return uint64(newBalance), nil
}

// addCommon() sums fields of any diffs.
func (diff *balanceDiff) addCommon(prevDiff *balanceDiff) error {
	var err error
	if diff.balance, err = common.AddInt64(diff.balance, prevDiff.balance); err != nil {
		return errors.Errorf("failed to add balance diffs: %v\n", err)
	}
	if diff.leaseIn, err = common.AddInt64(diff.leaseIn, prevDiff.leaseIn); err != nil {
		return errors.Errorf("failed to add LeaseIn diffs: %v\n", err)
	}
	if diff.leaseOut, err = common.AddInt64(diff.leaseOut, prevDiff.leaseOut); err != nil {
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
		newMinBalance, err := common.AddInt64(diff.balance, prevDiff.minBalance)
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
	newMinBalance, err := common.AddInt64(diff.minBalance, prevDiff.balance)
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
	blockInfo *proto.BlockInfo
}

func newDifferInfo(blockInfo *proto.BlockInfo) *differInfo {
	return &differInfo{blockInfo: blockInfo}
}

func (i *differInfo) hasMiner() bool {
	return i.blockInfo.GeneratorPublicKey != (crypto.PublicKey{})
}

type txBalanceChanges struct {
	addrs map[proto.WavesAddress]struct{} // Addresses affected by this transactions, excluding miners.
	diff  txDiff                          // Balance diffs.
}

func newTxBalanceChanges(addresses []proto.WavesAddress, diff txDiff) txBalanceChanges {
	addressesMap := make(map[proto.WavesAddress]struct{})
	for _, addr := range addresses {
		addressesMap[addr] = empty
	}
	return txBalanceChanges{addrs: addressesMap, diff: diff}
}

func (ch txBalanceChanges) appendAddr(addr proto.WavesAddress) {
	ch.addrs[addr] = empty
}

func (ch txBalanceChanges) addresses() []proto.WavesAddress {
	res := make([]proto.WavesAddress, len(ch.addrs))
	index := 0
	for addr := range ch.addrs {
		res[index] = addr
		index++
	}
	return res
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
	ngActivated, err := td.stor.features.newestIsActivatedForNBlocks(int16(settings.NG), 1)
	if err != nil {
		return 0, err
	}
	return calculateCurrentBlockTxFee(txFee, ngActivated), nil
}

func (td *transactionDiffer) minerPayoutInWaves(diff txDiff, fee uint64, info *differInfo) error {
	return td.minerPayout(diff, fee, info, proto.NewOptionalAssetWaves())
}

// minerPayout adds current fee part of given tx to txDiff.
func (td *transactionDiffer) minerPayout(diff txDiff, fee uint64, info *differInfo, feeAsset proto.OptionalAsset) error {
	minerAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, info.blockInfo.GeneratorPublicKey)
	if err != nil {
		return err
	}
	minerKey := byteKey(minerAddr.ID(), feeAsset)
	minerBalanceDiff, err := td.calculateTxFee(fee)
	if err != nil {
		return err
	}
	if err := diff.appendBalanceDiff(minerKey, newBalanceDiff(int64(minerBalanceDiff), 0, 0, false)); err != nil {
		return err
	}
	return nil
}

func (td *transactionDiffer) createDiffGenesis(transaction proto.Transaction, _ *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.Genesis)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to Genesis transaction")
	}
	diff := newTxDiff()
	key := wavesBalanceKey{address: tx.Recipient.ID()}
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(key.bytes(), newBalanceDiff(receiverBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	addresses := []proto.WavesAddress{tx.Recipient}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffPayment(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.Payment)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to Payment transaction")
	}
	diff := newTxDiff()
	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime {
		updateMinIntermediateBalance = true
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderKey := wavesBalanceKey{address: senderAddr.ID()}
	senderBalanceDiff := -int64(tx.Amount) - int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(senderBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	// Append receiver diff.
	receiverKey := wavesBalanceKey{address: tx.Recipient.ID()}
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addresses := []proto.WavesAddress{senderAddr, tx.Recipient}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func recipientToAddress(recipient proto.Recipient, aliases *aliases) (*proto.WavesAddress, error) {
	if addr := recipient.Address(); addr != nil {
		return addr, nil
	}
	addr, err := aliases.newestAddrByAlias(recipient.Alias().Alias)
	if err != nil {
		return nil, errors.Wrap(err, "invalid alias")
	}
	return addr, nil
}

func (td *transactionDiffer) payoutMinerWithSponsorshipHandling(ch *txBalanceChanges, fee uint64, feeAsset proto.OptionalAsset, info *differInfo) error {
	sponsorshipActivated, err := td.stor.sponsoredAssets.isSponsorshipActivated()
	if err != nil {
		return err
	}
	needToApplySponsorship := sponsorshipActivated && feeAsset.Present
	if !needToApplySponsorship {
		// No assets sponsorship.
		if info.hasMiner() {
			if err := td.minerPayout(ch.diff, fee, info, feeAsset); err != nil {
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
	shortAssetID := proto.AssetIDFromDigest(feeAsset.ID)
	assetInfo, err := td.stor.assets.newestAssetInfo(shortAssetID)
	if err != nil {
		return err
	}
	// Append issuer asset balance diff.
	issuerAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, assetInfo.issuer)
	if err != nil {
		return err
	}
	issuerAddrID := issuerAddr.ID()

	issuerAssetKey := byteKey(issuerAddrID, feeAsset)
	issuerAssetBalanceDiff := int64(fee)
	if err := ch.diff.appendBalanceDiff(issuerAssetKey, newBalanceDiff(issuerAssetBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return err
	}
	// Append issuer Waves balance diff.
	feeInWaves, err := td.stor.sponsoredAssets.sponsoredAssetToWaves(shortAssetID, fee)
	if err != nil {
		return err
	}
	issuerWavesKey := (&wavesBalanceKey{issuerAddrID}).bytes()
	issuerWavesBalanceDiff := -int64(feeInWaves)
	if err := ch.diff.appendBalanceDiff(issuerWavesKey, newBalanceDiff(issuerWavesBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return err
	}
	// Sponsor is also added to list of modified addresses.
	ch.appendAddr(issuerAddr)
	// Miner payout using sponsorship.
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(ch.diff, feeInWaves, info); err != nil {
			return errors.Wrap(err, "failed to append miner payout")
		}
	}
	return nil
}

func (td *transactionDiffer) createDiffTransfer(tx *proto.Transfer, info *differInfo) (txBalanceChanges, error) {
	diff := newTxDiff()
	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime {
		updateMinIntermediateBalance = true
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderAddrID := senderAddr.ID()

	senderFeeKey := byteKey(senderAddrID, tx.FeeAsset)
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	senderAmountKey := byteKey(senderAddrID, tx.AmountAsset)
	senderAmountBalanceDiff := -int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderAmountKey, newBalanceDiff(senderAmountBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	// Append receiver diff.
	recipientAddr, err := recipientToAddress(tx.Recipient, td.stor.aliases)
	if err != nil {
		return txBalanceChanges{}, err
	}
	receiverKey := byteKey(recipientAddr.ID(), tx.AmountAsset)
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey, newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	addrs := []proto.WavesAddress{senderAddr, *recipientAddr}
	changes := newTxBalanceChanges(addrs, diff)
	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.Fee, tx.FeeAsset, info); err != nil {
		return txBalanceChanges{}, err
	}
	return changes, nil
}

func (td *transactionDiffer) createDiffEthereumTransferWaves(tx *proto.EthereumTransaction, info *differInfo) (txBalanceChanges, error) {
	diff := newTxDiff()

	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime {
		updateMinIntermediateBalance = true
	}
	// Append sender diff.
	senderAddress, err := tx.WavesAddressFrom(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}
	wavesAsset := proto.NewOptionalAssetWaves()

	senderFeeKey := byteKey(senderAddress.ID(), wavesAsset)
	senderFeeBalanceDiff := -int64(tx.GetFee())
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}

	res := new(big.Int).Div(tx.Value(), big.NewInt(int64(proto.DiffEthWaves)))
	if ok := res.IsInt64(); !ok {
		return txBalanceChanges{}, errors.Errorf("failed to convert amount from ethreum transaction (big int) to int64. value is %s", tx.Value().String())
	}
	amount := res.Int64()

	senderAmountKey := byteKey(senderAddress.ID(), wavesAsset)

	senderAmountBalanceDiff := -amount
	if err := diff.appendBalanceDiff(senderAmountKey, newBalanceDiff(senderAmountBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	// Append receiver diff.
	recipientAddress, err := tx.To().ToWavesAddress(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}
	receiverKey := byteKey(recipientAddress.ID(), wavesAsset)
	receiverBalanceDiff := amount
	if err := diff.appendBalanceDiff(receiverKey, newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	addrs := []proto.WavesAddress{senderAddress, recipientAddress}
	changes := newTxBalanceChanges(addrs, diff)

	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.GetFee(), proto.NewOptionalAssetWaves(), info); err != nil {
		return txBalanceChanges{}, err
	}
	return changes, nil
}

func (td *transactionDiffer) createDiffEthereumErc20(tx *proto.EthereumTransaction, info *differInfo) (txBalanceChanges, error) {
	diff := newTxDiff()

	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime {
		updateMinIntermediateBalance = true
	}

	txErc20Kind, ok := tx.TxKind.(*proto.EthereumTransferAssetsErc20TxKind)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert ethereum tx kind to EthereumTransferAssetsErc20TxKind")
	}

	decodedData := txErc20Kind.DecodedData()

	var senderAddress proto.WavesAddress
	// Append sender diff.

	if !ethabi.IsERC20TransferSelector(decodedData.Signature.Selector()) {
		return txBalanceChanges{}, errors.New("unexpected type of eth selector")
	}

	EthSenderAddr, err := tx.From()
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderAddress, err = EthSenderAddr.ToWavesAddress(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}

	// Fee
	wavesAsset := proto.NewOptionalAssetWaves()
	senderFeeKey := byteKey(senderAddress.ID(), wavesAsset)
	senderFeeBalanceDiff := -int64(tx.GetFee())
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}

	// transfer

	senderAmountKey := byteKey(senderAddress.ID(), txErc20Kind.Asset)

	senderAmountBalanceDiff := -txErc20Kind.Arguments.Amount
	if err := diff.appendBalanceDiff(senderAmountKey, newBalanceDiff(senderAmountBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}

	etc20TransferRecipient, err := proto.EthereumAddress(txErc20Kind.Arguments.Recipient).ToWavesAddress(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}

	// Append receiver diff.
	receiverKey := byteKey(etc20TransferRecipient.ID(), txErc20Kind.Asset)
	receiverBalanceDiff := txErc20Kind.Arguments.Amount
	if err := diff.appendBalanceDiff(receiverKey, newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	addrs := []proto.WavesAddress{senderAddress, etc20TransferRecipient}
	changes := newTxBalanceChanges(addrs, diff)
	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.GetFee(), proto.NewOptionalAssetWaves(), info); err != nil {
		return txBalanceChanges{}, err
	}
	return changes, nil
}

func (td *transactionDiffer) createDiffTransferWithSig(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to TransferWithSig transaction")
	}
	return td.createDiffTransfer(&tx.Transfer, info)
}

func (td *transactionDiffer) createDiffEthereumTransactionWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	ethTx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to EthereumTransaction transaction")
	}

	switch ethTx.TxKind.(type) {
	case *proto.EthereumTransferWavesTxKind:
		return td.createDiffEthereumTransferWaves(ethTx, info)
	case *proto.EthereumTransferAssetsErc20TxKind:
		return td.createDiffEthereumErc20(ethTx, info)
	case *proto.EthereumInvokeScriptTxKind:
		return td.createDiffEthereumInvokeScript(ethTx, info)
	default:
		return txBalanceChanges{}, errors.New("wrong kind of ethereum transaction")

	}
}

func (td *transactionDiffer) createDiffTransferWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to TransferWithProofs transaction")
	}
	return td.createDiffTransfer(&tx.Transfer, info)
}

func (td *transactionDiffer) createDiffIssue(tx *proto.Issue, id []byte, info *differInfo) (txBalanceChanges, error) {
	diff := newTxDiff()
	assetID, err := crypto.NewDigestFromBytes(id)
	if err != nil {
		return txBalanceChanges{}, err
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderAddrID := senderAddr.ID()
	senderFeeKey := wavesBalanceKey{address: senderAddrID}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddrID, asset: proto.AssetIDFromDigest(assetID)}
	senderAssetBalanceDiff := int64(tx.Quantity)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addrs := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addrs, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffIssueWithSig(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	txID, err := tx.GetID(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	return td.createDiffIssue(&tx.Issue, txID, info)
}

func (td *transactionDiffer) createDiffIssueWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	txID, err := tx.GetID(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	return td.createDiffIssue(&tx.Issue, txID, info)
}

func (td *transactionDiffer) createDiffReissue(tx *proto.Reissue, info *differInfo) (txBalanceChanges, error) {
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderAddrID := senderAddr.ID()
	senderFeeKey := wavesBalanceKey{address: senderAddrID}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddrID, asset: proto.AssetIDFromDigest(tx.AssetID)}
	senderAssetBalanceDiff := int64(tx.Quantity)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addrs := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addrs, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffReissueWithSig(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	return td.createDiffReissue(&tx.Reissue, info)
}

func (td *transactionDiffer) createDiffReissueWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	return td.createDiffReissue(&tx.Reissue, info)
}

func (td *transactionDiffer) createDiffBurn(tx *proto.Burn, info *differInfo) (txBalanceChanges, error) {
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderAddrID := senderAddr.ID()
	senderFeeKey := wavesBalanceKey{address: senderAddrID}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddrID, asset: proto.AssetIDFromDigest(tx.AssetID)}
	senderAssetBalanceDiff := -int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addrs := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addrs, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffBurnWithSig(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	return td.createDiffBurn(&tx.Burn, info)
}

func (td *transactionDiffer) createDiffBurnWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	return td.createDiffBurn(&tx.Burn, info)
}

func (td *transactionDiffer) orderFeeKey(address proto.AddressID, order proto.Order) []byte {
	switch o := order.(type) {
	case *proto.EthereumOrderV4, *proto.OrderV4, *proto.OrderV3:
		matcherFeeAsset := o.GetMatcherFeeAsset()
		return byteKey(address, matcherFeeAsset)
	default:
		k := wavesBalanceKey{address}
		return k.bytes()
	}
}

func (td *transactionDiffer) orderAssetDecimals(transaction proto.Transaction, priceAsset bool) (int, error) {
	exchange, ok := transaction.(proto.Exchange)
	if !ok {
		return 0, errors.Errorf("unsupported transaction type '%T'", transaction)
	}
	switch v := transaction.GetVersion(); v {
	case 1, 2:
		// For old transaction version function returns 8.
		return 8, nil
	case 3, 4:
		buy, err := exchange.GetBuyOrder()
		if err != nil {
			return 0, err
		}
		asset := buy.GetAssetPair().AmountAsset
		if priceAsset {
			asset = buy.GetAssetPair().PriceAsset
		}
		if asset.Present {
			info, err := td.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(asset.ID))
			if err != nil {
				return 0, err
			}
			return int(info.decimals), nil
		}
		// Waves in pair, return 8
		return 8, nil
	default:
		return 0, errors.Errorf("unsupported exchange transaction version %d", v)
	}
}

var ten = decimal.WithContext(decimal.Context128).SetUint64(10)

func convertPrice(price int64, amountDecimals, priceDecimals int) (uint64, error) {
	p := decimal.WithContext(decimal.Context128).SetMantScale(price, 0)
	e := decimal.WithContext(decimal.Context128).SetMantScale(int64(priceDecimals-amountDecimals), 0)
	x := decimal.WithContext(decimal.Context128)
	math.Pow(x, ten, e)
	p.QuoInt(p, x)
	r, ok := p.Int64()
	if !ok {
		return 0, errors.New("int64 overflow")
	}
	if r <= 0 {
		return 0, errors.New("price should be positive")
	}
	return uint64(r), nil
}

func orderPrice(exchangeVersion byte, order proto.Order, amountDecimals, priceDecimals int) (uint64, error) {
	price := order.GetPrice()
	if exchangeVersion >= 3 {
		if order.GetVersion() < 4 || order.GetPriceMode() == proto.OrderPriceModeAssetDecimals {
			return convertPrice(int64(price), amountDecimals, priceDecimals)
		}
	}
	return price, nil
}

// amount = matchAmount * matchPrice * 10^(priceDecimals - amountDecimals - 8)
func calculateAmount(matchAmount, matchPrice uint64, amountDecimal, priceDecimals int) (int64, error) {
	a := decimal.WithContext(decimal.Context128).SetUint64(matchAmount)
	p := decimal.WithContext(decimal.Context128).SetUint64(matchPrice)
	e := decimal.WithContext(decimal.Context128).SetMantScale(int64(priceDecimals-amountDecimal-8), 0)
	x := decimal.WithContext(decimal.Context128)
	math.Pow(x, ten, e)
	y := decimal.WithContext(decimal.Context128)
	y.Mul(a, p)
	y.Mul(y, x)
	r, ok := y.Int64()
	if !ok {
		return 0, errors.New("int64 overflow")
	}
	if r < 0 {
		return 0, errors.New("result should not be negative")
	}
	return r, nil
}

func (td *transactionDiffer) createDiffExchange(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to Exchange transaction")
	}
	diff := newTxDiff()
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return txBalanceChanges{}, err
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return txBalanceChanges{}, err
	}
	amountAsset := buyOrder.GetAssetPair().AmountAsset
	priceAsset := buyOrder.GetAssetPair().PriceAsset
	amountDecimals, err := td.orderAssetDecimals(transaction, false)
	if err != nil {
		return txBalanceChanges{}, err
	}
	priceDecimals, err := td.orderAssetDecimals(transaction, true)
	if err != nil {
		return txBalanceChanges{}, err
	}
	// For old orders and exchanges convert price to new formula
	buyOrderPrice, err := orderPrice(transaction.GetVersion(), buyOrder, amountDecimals, priceDecimals)
	if err != nil {
		return txBalanceChanges{}, err
	}
	sellOrderPrice, err := orderPrice(transaction.GetVersion(), sellOrder, amountDecimals, priceDecimals)
	if err != nil {
		return txBalanceChanges{}, err
	}
	if tx.GetPrice() > buyOrderPrice || tx.GetPrice() < sellOrderPrice {
		return txBalanceChanges{}, errors.Errorf("invalid exchange transaction price (%d), should be between %d and %d", tx.GetPrice(), sellOrderPrice, buyOrderPrice)
	}
	// Perform exchange.
	priceAssetDiff, err := calculateAmount(tx.GetAmount(), tx.GetPrice(), amountDecimals, priceDecimals)
	if err != nil {
		id, _ := transaction.GetID(td.settings.AddressSchemeCharacter)
		return txBalanceChanges{}, errors.Wrapf(err, "invalid exchange transaction ('%s') amount", base58.Encode(id))
	}
	amountDiff := int64(tx.GetAmount())

	// because sender can be either of EthereumAddress or WavesAddress we have to convert both of them to WavesAddress
	senderAddr, err := sellOrder.GetSender(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderAddrID := senderAddr.ID()

	senderPriceKey := byteKey(senderAddrID, priceAsset)
	if err := diff.appendBalanceDiff(senderPriceKey, newBalanceDiff(priceAssetDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	senderAmountKey := byteKey(senderAddrID, amountAsset)
	if err := diff.appendBalanceDiff(senderAmountKey, newBalanceDiff(-amountDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}

	// because sender can be either of EthereumAddress or WavesAddress we have to convert both of them to WavesAddress
	receiverAddr, err := buyOrder.GetSender(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}
	receiverAddrID := receiverAddr.ID()

	receiverPriceKey := byteKey(receiverAddrID, priceAsset)
	if err := diff.appendBalanceDiff(receiverPriceKey, newBalanceDiff(-priceAssetDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	receiverAmountKey := byteKey(receiverAddrID, amountAsset)
	if err := diff.appendBalanceDiff(receiverAmountKey, newBalanceDiff(amountDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}

	// Fees.
	matcherAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, buyOrder.GetMatcherPK())
	if err != nil {
		return txBalanceChanges{}, err
	}
	matcherAddrID := matcherAddr.ID()

	senderFee := int64(tx.GetSellMatcherFee())
	senderFeeKey := td.orderFeeKey(senderAddrID, sellOrder)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(-senderFee, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	matcherFeeFromSenderKey := td.orderFeeKey(matcherAddrID, sellOrder)
	if err := diff.appendBalanceDiff(matcherFeeFromSenderKey, newBalanceDiff(senderFee, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	receiverFee := int64(tx.GetBuyMatcherFee())
	receiverFeeKey := td.orderFeeKey(receiverAddrID, buyOrder)
	if err := diff.appendBalanceDiff(receiverFeeKey, newBalanceDiff(-receiverFee, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	matcherFeeFromReceiverKey := td.orderFeeKey(matcherAddrID, buyOrder)
	if err := diff.appendBalanceDiff(matcherFeeFromReceiverKey, newBalanceDiff(receiverFee, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	matcherKey := wavesBalanceKey{matcherAddrID}
	matcherFee := int64(tx.GetFee())
	if err := diff.appendBalanceDiff(matcherKey.bytes(), newBalanceDiff(-matcherFee, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.GetFee(), info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}

	txSenderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderWavesAddr, err := senderAddr.ToWavesAddress(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}
	receiverWavesAddr, err := receiverAddr.ToWavesAddress(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}

	addresses := []proto.WavesAddress{txSenderAddr, senderWavesAddr, receiverWavesAddr, matcherAddr}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffForExchangeFeeValidation(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to Exchange transaction")
	}
	diff := newTxDiff()
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return txBalanceChanges{}, err
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return txBalanceChanges{}, err
	}

	// because sender can be either of EthereumAddress or WavesAddress we have to convert both of them to WavesAddress
	senderAddr, err := sellOrder.GetSender(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}

	// because sender can be either of EthereumAddress or WavesAddress we have to convert both of them to WavesAddress
	receiverAddr, err := buyOrder.GetSender(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}

	matcherAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, buyOrder.GetMatcherPK())
	if err != nil {
		return txBalanceChanges{}, err
	}
	matcherAddrID := matcherAddr.ID()

	matcherKey := wavesBalanceKey{matcherAddrID}
	matcherFee := int64(tx.GetFee())
	if err := diff.appendBalanceDiff(matcherKey.bytes(), newBalanceDiff(-matcherFee, 0, 0, true)); err != nil {
		return txBalanceChanges{}, err
	}
	senderFee := int64(tx.GetSellMatcherFee())
	senderFeeKey := td.orderFeeKey(senderAddr.ID(), sellOrder)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(-senderFee, 0, 0, true)); err != nil {
		return txBalanceChanges{}, err
	}
	matcherFeeFromSenderKey := td.orderFeeKey(matcherAddrID, sellOrder)
	if err := diff.appendBalanceDiff(matcherFeeFromSenderKey, newBalanceDiff(senderFee, 0, 0, true)); err != nil {
		return txBalanceChanges{}, err
	}
	receiverFee := int64(tx.GetBuyMatcherFee())
	receiverFeeKey := td.orderFeeKey(receiverAddr.ID(), buyOrder)
	if err := diff.appendBalanceDiff(receiverFeeKey, newBalanceDiff(-receiverFee, 0, 0, true)); err != nil {
		return txBalanceChanges{}, err
	}
	matcherFeeFromReceiverKey := td.orderFeeKey(matcherAddrID, buyOrder)
	if err := diff.appendBalanceDiff(matcherFeeFromReceiverKey, newBalanceDiff(receiverFee, 0, 0, true)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.GetFee(), info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	txSenderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return txBalanceChanges{}, err
	}
	addresses := []proto.WavesAddress{txSenderAddr, matcherAddr}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createFeeDiffExchange(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to Exchange transaction")
	}
	diff := newTxDiff()
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return txBalanceChanges{}, err
	}
	matcherAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, buyOrder.GetMatcherPK())
	if err != nil {
		return txBalanceChanges{}, err
	}
	matcherKey := wavesBalanceKey{matcherAddr.ID()}
	matcherFee := int64(tx.GetFee())
	if err := diff.appendBalanceDiff(matcherKey.bytes(), newBalanceDiff(-matcherFee, 0, 0, true)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.GetFee(), info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	txSenderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return txBalanceChanges{}, err
	}
	addresses := []proto.WavesAddress{txSenderAddr, matcherAddr}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffLease(tx *proto.Lease, info *differInfo) (txBalanceChanges, error) {
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderKey := wavesBalanceKey{address: senderAddr.ID()}
	senderLeaseOutDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(0, 0, senderLeaseOutDiff, false)); err != nil {
		return txBalanceChanges{}, err
	}
	senderFeeDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(senderFeeDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	// Append receiver diff.
	recipientAddr, err := recipientToAddress(tx.Recipient, td.stor.aliases)
	if err != nil {
		return txBalanceChanges{}, err
	}
	receiverKey := wavesBalanceKey{address: recipientAddr.ID()}
	receiverLeaseInDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), newBalanceDiff(0, receiverLeaseInDiff, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addresses := []proto.WavesAddress{senderAddr, *recipientAddr}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffLeaseWithSig(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return td.createDiffLease(&tx.Lease, info)
}

func (td *transactionDiffer) createDiffLeaseWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return td.createDiffLease(&tx.Lease, info)
}

func (td *transactionDiffer) createDiffLeaseCancel(tx *proto.LeaseCancel, info *differInfo) (txBalanceChanges, error) {
	diff := newTxDiff()
	l, err := td.stor.leases.newestLeasingInfo(tx.LeaseID)
	if err != nil {
		return txBalanceChanges{}, errors.Wrap(err, "no leasing info found for this leaseID")
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderKey := wavesBalanceKey{address: senderAddr.ID()}
	senderLeaseOutDiff := -int64(l.Amount)
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(0, 0, senderLeaseOutDiff, false)); err != nil {
		return txBalanceChanges{}, err
	}
	senderFeeDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), newBalanceDiff(senderFeeDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	// Append receiver diff.
	receiverKey := wavesBalanceKey{address: l.Recipient.ID()}
	receiverLeaseInDiff := -int64(l.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), newBalanceDiff(0, receiverLeaseInDiff, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addresses := []proto.WavesAddress{senderAddr, l.Recipient}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffLeaseCancelWithSig(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	return td.createDiffLeaseCancel(&tx.LeaseCancel, info)
}

func (td *transactionDiffer) createDiffLeaseCancelWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	return td.createDiffLeaseCancel(&tx.LeaseCancel, info)
}

func (td *transactionDiffer) createDiffCreateAlias(tx *proto.CreateAlias, info *differInfo) (txBalanceChanges, error) {
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr.ID()}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addresses := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffCreateAliasWithSig(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	return td.createDiffCreateAlias(&tx.CreateAlias, info)
}

func (td *transactionDiffer) createDiffCreateAliasWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return td.createDiffCreateAlias(&tx.CreateAlias, info)
}

func (td *transactionDiffer) createDiffMassTransferWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to MassTransferWithProofs transaction")
	}
	diff := newTxDiff()
	addresses := make([]proto.WavesAddress, len(tx.Transfers)+1)
	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime {
		updateMinIntermediateBalance = true
	}
	// Append sender fee diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderAddrID := senderAddr.ID()

	addresses[0] = senderAddr
	senderFeeKey := wavesBalanceKey{address: senderAddrID}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	// Append amount diffs.
	senderAmountKey := byteKey(senderAddrID, tx.Asset)
	for i, entry := range tx.Transfers {
		// Sender.
		senderAmountBalanceDiff := -int64(entry.Amount)
		if err := diff.appendBalanceDiff(senderAmountKey, newBalanceDiff(senderAmountBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txBalanceChanges{}, err
		}
		// Recipient.
		recipientAddr, err := recipientToAddress(entry.Recipient, td.stor.aliases)
		if err != nil {
			return txBalanceChanges{}, err
		}
		recipientKey := byteKey(recipientAddr.ID(), tx.Asset)
		recipientBalanceDiff := int64(entry.Amount)
		if err := diff.appendBalanceDiff(recipientKey, newBalanceDiff(recipientBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txBalanceChanges{}, err
		}
		addresses[i+1] = *recipientAddr
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffDataWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to DataWithProofs transaction")
	}
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr.ID()}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addresses := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffSponsorshipWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr.ID()}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addresses := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffSetScriptWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr.ID()}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addresses := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffSetAssetScriptWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}
	diff := newTxDiff()
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr.ID()}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	if info.hasMiner() {
		if err := td.minerPayoutInWaves(diff, tx.Fee, info); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	addresses := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addresses, diff)
	return changes, nil
}

func (td *transactionDiffer) createDiffInvokeScriptWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
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
		return txBalanceChanges{}, err
	}
	senderAddrID := senderAddr.ID()

	senderFeeKey := byteKey(senderAddrID, tx.FeeAsset)
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	scriptAddr, err := recipientToAddress(tx.ScriptRecipient, td.stor.aliases)
	if err != nil {
		return txBalanceChanges{}, err
	}
	scriptAddrID := scriptAddr.ID()

	addresses := []proto.WavesAddress{senderAddr, *scriptAddr}
	changes := newTxBalanceChanges(addresses, diff)
	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.Fee, tx.FeeAsset, info); err != nil {
		return txBalanceChanges{}, err
	}
	// Append payment diffs.
	for _, payment := range tx.Payments {
		senderPaymentKey := byteKey(senderAddrID, payment.Asset)
		senderBalanceDiff := -int64(payment.Amount)
		if err := diff.appendBalanceDiff(senderPaymentKey, newBalanceDiff(senderBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txBalanceChanges{}, err
		}
		receiverKey := byteKey(scriptAddrID, payment.Asset)
		receiverBalanceDiff := int64(payment.Amount)
		if err := diff.appendBalanceDiff(receiverKey, newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txBalanceChanges{}, err
		}
	}
	return changes, nil
}

func (td *transactionDiffer) createDiffInvokeExpressionWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to InvokeExpessionWithProofs transaction")
	}
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderAddrID := senderAddr.ID()

	senderFeeKey := byteKey(senderAddrID, tx.FeeAsset)
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	scriptAddr, err := recipientToAddress(proto.NewRecipientFromAddress(senderAddr), td.stor.aliases)
	if err != nil {
		return txBalanceChanges{}, err
	}

	addresses := []proto.WavesAddress{senderAddr, *scriptAddr}
	changes := newTxBalanceChanges(addresses, diff)
	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.Fee, tx.FeeAsset, info); err != nil {
		return txBalanceChanges{}, err
	}
	return changes, nil
}

func (td *transactionDiffer) createDiffEthereumInvokeScript(tx *proto.EthereumTransaction, info *differInfo) (txBalanceChanges, error) {
	txInvokeScriptKind, ok := tx.TxKind.(*proto.EthereumInvokeScriptTxKind)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert ethereum tx kind to EthereumTransferAssetsErc20TxKind")
	}

	payments := txInvokeScriptKind.DecodedData().Payments
	updateMinIntermediateBalance := false
	if info.blockInfo.Timestamp >= td.settings.CheckTempNegativeAfterTime && len(payments) > 0 {
		updateMinIntermediateBalance = true
	}
	diff := newTxDiff()
	// Append sender diff.
	senderAddress, err := tx.WavesAddressFrom(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, errors.Wrapf(err, "failed to get sender address from ethereum invoke tx")
	}

	senderAddrID := senderAddress.ID()
	assetFee := proto.NewOptionalAssetWaves()
	senderFeeKey := byteKey(senderAddrID, assetFee)
	senderFeeBalanceDiff := -int64(tx.GetFee())
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txBalanceChanges{}, err
	}
	scriptAddr, err := tx.WavesAddressTo(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}
	scriptAddrID := scriptAddr.ID()

	addresses := []proto.WavesAddress{senderAddress, *scriptAddr}
	changes := newTxBalanceChanges(addresses, diff)

	for _, p := range payments {
		optAsset := proto.NewOptionalAsset(p.PresentAssetID, p.AssetID)
		senderPaymentKey := byteKey(senderAddrID, optAsset)
		senderBalanceDiff := -p.Amount
		if err := diff.appendBalanceDiff(senderPaymentKey, newBalanceDiff(senderBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txBalanceChanges{}, err
		}
		receiverKey := byteKey(scriptAddrID, optAsset)
		receiverBalanceDiff := p.Amount
		if err := diff.appendBalanceDiff(receiverKey, newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
			return txBalanceChanges{}, err
		}
	}
	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.GetFee(), proto.NewOptionalAssetWaves(), info); err != nil {
		return txBalanceChanges{}, err
	}
	return changes, nil
}

// TODO make one function for 3 tx types
func (td *transactionDiffer) createFeeDiffInvokeExpressionWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderFeeKey := byteKey(senderAddr.ID(), tx.FeeAsset)
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, true)); err != nil {
		return txBalanceChanges{}, err
	}

	addresses := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addresses, diff)
	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.GetFee(), proto.NewOptionalAssetWaves(), info); err != nil {
		return txBalanceChanges{}, err
	}
	return changes, nil
}

func (td *transactionDiffer) createFeeDiffInvokeScriptWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderFeeKey := byteKey(senderAddr.ID(), tx.FeeAsset)
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, true)); err != nil {
		return txBalanceChanges{}, err
	}
	scriptAddr, err := recipientToAddress(tx.ScriptRecipient, td.stor.aliases)
	if err != nil {
		return txBalanceChanges{}, err
	}
	addresses := []proto.WavesAddress{senderAddr, *scriptAddr}
	changes := newTxBalanceChanges(addresses, diff)
	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.Fee, tx.FeeAsset, info); err != nil {
		return txBalanceChanges{}, err
	}
	return changes, nil
}

func (td *transactionDiffer) createFeeDiffEthereumInvokeScriptWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	diff := newTxDiff()
	// Append sender diff.
	EthSenderAddr, err := tx.From()
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderAddress, err := EthSenderAddr.ToWavesAddress(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}
	wavesAsset := proto.NewOptionalAssetWaves()
	senderFeeKey := byteKey(senderAddress.ID(), wavesAsset)
	senderFeeBalanceDiff := -int64(tx.GetFee())
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, true)); err != nil {
		return txBalanceChanges{}, err
	}
	scriptAddress, err := tx.To().ToWavesAddress(td.settings.AddressSchemeCharacter)
	if err != nil {
		return txBalanceChanges{}, err
	}

	addresses := []proto.WavesAddress{senderAddress, scriptAddress}
	changes := newTxBalanceChanges(addresses, diff)

	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.GetFee(), proto.NewOptionalAssetWaves(), info); err != nil {
		return txBalanceChanges{}, err
	}
	return changes, nil
}

func (td *transactionDiffer) createDiffUpdateAssetInfoWithProofs(transaction proto.Transaction, info *differInfo) (txBalanceChanges, error) {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return txBalanceChanges{}, errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	diff := newTxDiff()
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(td.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txBalanceChanges{}, err
	}
	senderFeeKey := byteKey(senderAddr.ID(), tx.FeeAsset)
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, newBalanceDiff(senderFeeBalanceDiff, 0, 0, false)); err != nil {
		return txBalanceChanges{}, err
	}
	addresses := []proto.WavesAddress{senderAddr}
	changes := newTxBalanceChanges(addresses, diff)
	if err := td.payoutMinerWithSponsorshipHandling(&changes, tx.Fee, tx.FeeAsset, info); err != nil {
		return txBalanceChanges{}, err
	}
	return changes, nil
}
