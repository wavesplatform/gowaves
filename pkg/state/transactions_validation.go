package state

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	priceConstant = 10e7

	wavesBalanceKeySize = 1 + proto.AddressSize
	assetBalanceKeySize = 1 + proto.AddressSize + crypto.DigestSize
)

type balanceDiff struct {
	balance  int64
	leaseIn  int64
	leaseOut int64
	blockID  crypto.Signature
}

func (diff *balanceDiff) spendableBalanceDiff() int64 {
	return diff.balance - diff.leaseOut
}

func (diff *balanceDiff) applyTo(profile *balanceProfile) (*balanceProfile, error) {
	newBalance, err := util.AddInt64(diff.balance, int64(profile.balance))
	if err != nil {
		return nil, errors.Errorf("failed to add balance and balance diff: %v\n", err)
	}
	newLeaseIn, err := util.AddInt64(diff.leaseIn, int64(profile.leaseIn))
	if err != nil {
		return nil, errors.Errorf("failed to add leaseIn and leaseIn diff: %v\n", err)
	}
	newLeaseOut, err := util.AddInt64(diff.leaseOut, int64(profile.leaseOut))
	if err != nil {
		return nil, errors.Errorf("failed to add leaseOut and leaseOut diff: %v\n", err)
	}
	if newBalance-newLeaseOut < 0 {
		return nil, errors.New("negative balance")
	}
	newProfile := &balanceProfile{}
	newProfile.balance = uint64(newBalance)
	newProfile.leaseIn = uint64(newLeaseIn)
	newProfile.leaseOut = uint64(newLeaseOut)
	return newProfile, nil
}

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

type balanceChanges struct {
	// Key in main DB.
	key []byte
	// Cumulative diffs of blocks transactions.
	balanceDiffs []balanceDiff
	// Diff which produces minimal spendable (taking leasing into account) balance value.
	// This is needed to check for negative balances.
	// For blocks when temporary negative balances are possible,
	// this value is set to the cumulative diff of all block's transactions.
	minSpendableBalanceDiff int64
}

func (ch *balanceChanges) update(newDiff balanceDiff, checkTempNegative bool) error {
	last := len(ch.balanceDiffs) - 1
	lastDiff := balanceDiff{}
	if last >= 0 {
		lastDiff = ch.balanceDiffs[last]
	}
	if err := newDiff.add(&lastDiff); err != nil {
		return errors.Errorf("failed to add diffs: %v\n", err)
	}
	if newDiff.blockID != lastDiff.blockID {
		ch.balanceDiffs = append(ch.balanceDiffs, newDiff)
	} else {
		ch.balanceDiffs[last] = newDiff
	}
	if checkTempNegative {
		// Check every tx, minSpendableBalanceDiff will have mimimum diff value among all txs at the end.
		if newDiff.spendableBalanceDiff() < ch.minSpendableBalanceDiff {
			ch.minSpendableBalanceDiff = newDiff.spendableBalanceDiff()
		}
	}
	return nil
}

type byKey []balanceChanges

func (k byKey) Len() int {
	return len(k)
}
func (k byKey) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}
func (k byKey) Less(i, j int) bool {
	return bytes.Compare(k[i].key, k[j].key) == -1
}

type wavesBalanceKey [wavesBalanceKeySize]byte
type assetBalanceKey [assetBalanceKeySize]byte

type changesStorage struct {
	balances  *balances
	deltas    []balanceChanges
	wavesKeys map[wavesBalanceKey]int // waves key --> index in deltas.
	assetKeys map[assetBalanceKey]int // asset key --> index in deltas.
}

func newChangesStorage(balances *balances) (*changesStorage, error) {
	return &changesStorage{
		balances:  balances,
		wavesKeys: make(map[wavesBalanceKey]int),
		assetKeys: make(map[assetBalanceKey]int),
	}, nil
}

func (bs *changesStorage) balanceChanges(key []byte) (*balanceChanges, error) {
	size := len(key)
	if size == wavesBalanceKeySize {
		var wavesKey wavesBalanceKey
		copy(wavesKey[:], key)
		if _, ok := bs.wavesKeys[wavesKey]; !ok {
			bs.wavesKeys[wavesKey] = len(bs.deltas)
			bs.deltas = append(bs.deltas, balanceChanges{key: key})
		}
		return &bs.deltas[bs.wavesKeys[wavesKey]], nil
	} else if size == assetBalanceKeySize {
		var assetKey assetBalanceKey
		copy(assetKey[:], key)
		if _, ok := bs.assetKeys[assetKey]; !ok {
			bs.assetKeys[assetKey] = len(bs.deltas)
			bs.deltas = append(bs.deltas, balanceChanges{key: key})
		}
		return &bs.deltas[bs.assetKeys[assetKey]], nil
	}
	return nil, errors.New("invalid key size")
}

func (bs *changesStorage) applyWavesChange(change *balanceChanges) error {
	var k balanceKey
	if err := k.unmarshal(change.key); err != nil {
		return errors.Errorf("failed to unmarshal balance key: %v\n", err)
	}
	profile, err := bs.balances.wavesBalance(k.address)
	if err != nil {
		return errors.Errorf("failed to retrieve waves balance: %v\n", err)
	}
	// Check for negative balance.
	minBalance, err := util.AddInt64(int64(profile.spendableBalance()), change.minSpendableBalanceDiff)
	if err != nil {
		return errors.Errorf("failed to add balances: %v\n", err)
	}
	if minBalance < 0 {
		return errors.New("validation failed: negative balance")
	}
	for _, diff := range change.balanceDiffs {
		newProfile, err := diff.applyTo(profile)
		if err != nil {
			return errors.Errorf("failed to apply waves balance change: %v\n", err)
		}
		r := &wavesBalanceRecord{*newProfile, diff.blockID}
		if err := bs.balances.setWavesBalance(k.address, r); err != nil {
			return errors.Errorf("failed to set account balance: %v\n", err)
		}
	}
	return nil
}

func (bs *changesStorage) applyAssetChange(change *balanceChanges) error {
	var k balanceKey
	if err := k.unmarshal(change.key); err != nil {
		return errors.Errorf("failed to unmarshal balance key: %v\n", err)
	}
	balance, err := bs.balances.assetBalance(k.address, k.asset)
	if err != nil {
		return errors.Errorf("failed to retrieve asset balance: %v\n", err)
	}
	// Check for negative balance.
	minBalance, err := util.AddInt64(int64(balance), change.minSpendableBalanceDiff)
	if err != nil {
		return errors.Errorf("failed to add balances: %v\n", err)
	}
	if minBalance < 0 {
		return errors.New("validation failed: negative balance")
	}
	for _, diff := range change.balanceDiffs {
		newBalance, err := util.AddInt64(int64(balance), diff.balance)
		if err != nil {
			return errors.Errorf("failed to add balances: %v\n", err)
		}
		if newBalance < 0 {
			return errors.New("validation failed: negative balance")
		}
		r := &assetBalanceRecord{uint64(newBalance), diff.blockID}
		if err := bs.balances.setAssetBalance(k.address, k.asset, r); err != nil {
			return errors.Errorf("failed to set asset balance: %v\n", err)
		}
	}
	return nil
}

// Apply all balance changes (actually move them to DB batch) and reset.
func (bs *changesStorage) applyDeltas() error {
	// Apply and validate balance variations.
	// At first, sort all changes by addresses they do modify.
	// That's *very* important optimization, since levelDB stores data
	// sorted by keys, and the idea is to read in sorted order.
	// We save a lot of time on disk's seek time.
	// TODO: if DB supported MultiGet() operation, this would probably be even faster.
	sort.Sort(byKey(bs.deltas))
	for _, delta := range bs.deltas {
		if len(delta.key) > wavesBalanceKeySize {
			// Is asset change.
			if err := bs.applyAssetChange(&delta); err != nil {
				return err
			}
		} else {
			// Is Waves change, need to take leasing into account.
			if err := bs.applyWavesChange(&delta); err != nil {
				return err
			}
		}
	}
	bs.reset()
	return nil
}

func (bs *changesStorage) reset() {
	bs.deltas = nil
	bs.wavesKeys = make(map[wavesBalanceKey]int)
	bs.assetKeys = make(map[assetBalanceKey]int)

}

type transactionValidator struct {
	genesis     crypto.Signature
	changesStor *changesStorage
	assets      *assets
	leases      *leases
	settings    *settings.BlockchainSettings
}

func newTransactionValidator(
	genesis crypto.Signature,
	balances *balances,
	assets *assets,
	leases *leases,
	settings *settings.BlockchainSettings,
) (*transactionValidator, error) {
	changesStor, err := newChangesStorage(balances)
	if err != nil {
		return nil, errors.Errorf("failed to create balances changes storage: %v\n", err)
	}
	return &transactionValidator{
		genesis:     genesis,
		changesStor: changesStor,
		assets:      assets,
		leases:      leases,
		settings:    settings,
	}, nil
}

func (tv *transactionValidator) checkFromFuture(timestamp uint64) bool {
	return timestamp > tv.settings.TxFromFutureCheckAfterTime
}

func (tv *transactionValidator) checkNegativeBalance(timestamp uint64) bool {
	return timestamp > tv.settings.NegativeBalanceCheckAfterTime
}

func (tv *transactionValidator) checkTxChangesSorted(timestamp uint64) bool {
	return timestamp > tv.settings.TxChangesSortedCheckAfterTime
}

func (tv *transactionValidator) checkTimestamps(txTimestamp, blockTimestamp, prevBlockTimestamp uint64) (bool, error) {
	if txTimestamp < prevBlockTimestamp-tv.settings.MaxTxTimeBackOffset {
		return false, errors.New("early transaction creation time")
	}
	if tv.checkFromFuture(blockTimestamp) && txTimestamp > blockTimestamp+tv.settings.MaxTxTimeForwardOffset {
		return false, errors.New("late transaction creation time")
	}
	return true, nil
}

func (tv *transactionValidator) addChange(key []byte, diff balanceDiff, block *proto.Block) (bool, error) {
	changes, err := tv.changesStor.balanceChanges(key)
	if err != nil {
		return false, errors.Wrap(err, "can not retrieve balance changes")
	}
	checkTempNegative := tv.checkNegativeBalance(block.Timestamp)
	if err := changes.update(diff, checkTempNegative); err != nil {
		return false, errors.Wrap(err, "can not update balance changes")
	}
	return true, nil
}

type balanceChange struct {
	key  []byte
	diff balanceDiff
}

func (tv *transactionValidator) pushChanges(changes []balanceChange, block *proto.Block) error {
	for _, ch := range changes {
		if ok, err := tv.addChange(ch.key, ch.diff, block); !ok {
			return err
		}
	}
	return nil
}

func (tv *transactionValidator) validateGenesis(tx *proto.Genesis, block *proto.Block, initialisation bool) (bool, error) {
	if block.BlockSignature != tv.genesis {
		return false, errors.New("genesis transaction inside of non-genesis block")
	}
	if !initialisation {
		return false, errors.New("genesis transaction in non-initialisation mode")
	}
	key := balanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	if ok, err := tv.addChange(key.bytes(), balanceDiff{balance: receiverBalanceDiff, blockID: block.BlockSignature}, block); !ok {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validatePayment(tx *proto.Payment, block, parent *proto.Block, initialisation bool) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, block.Timestamp, parent.Timestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	changes := make([]balanceChange, 3)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderKey := balanceKey{address: senderAddr}
	senderBalanceDiff := -int64(tx.Amount) - int64(tx.Fee)
	changes[0] = balanceChange{senderKey.bytes(), balanceDiff{balance: senderBalanceDiff, blockID: block.BlockSignature}}
	// Update receiver.
	receiverKey := balanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	changes[1] = balanceChange{receiverKey.bytes(), balanceDiff{balance: receiverBalanceDiff, blockID: block.BlockSignature}}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr}
	minerBalanceDiff := int64(tx.Fee)
	changes[2] = balanceChange{minerKey.bytes(), balanceDiff{balance: minerBalanceDiff, blockID: block.BlockSignature}}
	if err := tv.pushChanges(changes, block); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) checkAsset(asset *proto.OptionalAsset) error {
	if !asset.Present {
		// Waves always valid.
		return nil
	}
	if _, err := tv.assets.newestAssetRecord(asset.ID); err != nil {
		return errors.New("unknown asset")
	}
	return nil
}

func (tv *transactionValidator) validateTransfer(tx *proto.Transfer, block, parent *proto.Block, initialisation bool) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, block.Timestamp, parent.Timestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	if err := tv.checkAsset(&tx.AmountAsset); err != nil {
		return false, err
	}
	if err := tv.checkAsset(&tx.FeeAsset); err != nil {
		return false, err
	}
	changes := make([]balanceChange, 4)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr, asset: tx.FeeAsset.ToID()}
	senderFeeBalanceDiff := -int64(tx.Fee)
	changes[0] = balanceChange{senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff, blockID: block.BlockSignature}}
	senderAmountKey := balanceKey{address: senderAddr, asset: tx.AmountAsset.ToID()}
	senderAmountBalanceDiff := -int64(tx.Amount)
	changes[1] = balanceChange{senderAmountKey.bytes(), balanceDiff{balance: senderAmountBalanceDiff, blockID: block.BlockSignature}}
	// Update receiver.
	if tx.Recipient.Address == nil {
		// TODO implement
		return false, errors.New("alias without address is not supported yet")
	}
	receiverKey := balanceKey{address: *tx.Recipient.Address, asset: tx.AmountAsset.ToID()}
	receiverBalanceDiff := int64(tx.Amount)
	changes[2] = balanceChange{receiverKey.bytes(), balanceDiff{balance: receiverBalanceDiff, blockID: block.BlockSignature}}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr, asset: tx.FeeAsset.ToID()}
	minerBalanceDiff := int64(tx.Fee)
	changes[3] = balanceChange{minerKey.bytes(), balanceDiff{balance: minerBalanceDiff, blockID: block.BlockSignature}}
	if err := tv.pushChanges(changes, block); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateIssue(tx proto.Issue, block, parent *proto.Block, initialisation bool) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.GetTimestamp(), block.Timestamp, parent.Timestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	// Create new asset.
	info := &assetInfo{
		assetConstInfo: assetConstInfo{
			name:        tx.GetName(),
			description: tx.GetDescription(),
			decimals:    int8(tx.GetDecimals()),
		},
		assetHistoryRecord: assetHistoryRecord{
			quantity:   tx.GetQuantity(),
			reissuable: tx.GetReissuable(),
			blockID:    block.BlockSignature,
		},
	}
	assetID, err := crypto.NewDigestFromBytes(tx.GetID())
	if err != nil {
		return false, err
	}
	if err := tv.assets.issueAsset(assetID, info); err != nil {
		return false, errors.Wrap(err, "failed to issue asset")
	}
	changes := make([]balanceChange, 3)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.GetFee())
	changes[0] = balanceChange{senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff, blockID: block.BlockSignature}}
	senderAssetKey := balanceKey{address: senderAddr, asset: assetID[:]}
	senderAssetBalanceDiff := int64(tx.GetQuantity())
	changes[1] = balanceChange{senderAssetKey.bytes(), balanceDiff{balance: senderAssetBalanceDiff, blockID: block.BlockSignature}}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr}
	minerBalanceDiff := int64(tx.GetFee())
	changes[2] = balanceChange{minerKey.bytes(), balanceDiff{balance: minerBalanceDiff, blockID: block.BlockSignature}}
	if err := tv.pushChanges(changes, block); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateReissue(tx *proto.Reissue, block, parent *proto.Block, initialisation bool) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, block.Timestamp, parent.Timestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	// Check if it's "legal" to modify given asset.
	record, err := tv.assets.newestAssetRecord(tx.AssetID)
	if err != nil {
		return false, err
	}
	if (block.Timestamp > tv.settings.InvalidReissueInSameBlockUntilTime) && !record.reissuable {
		return false, errors.New("attempt to reissue asset which is not reissuable")
	}
	// Modify asset.
	change := &assetReissueChange{
		reissuable: tx.Reissuable,
		diff:       tx.Quantity,
		blockID:    block.BlockSignature,
	}
	if err := tv.assets.reissueAsset(tx.AssetID, change); err != nil {
		return false, errors.Wrap(err, "failed to reissue asset")
	}
	changes := make([]balanceChange, 3)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	changes[0] = balanceChange{senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff, blockID: block.BlockSignature}}
	senderAssetKey := balanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	changes[1] = balanceChange{senderAssetKey.bytes(), balanceDiff{balance: senderAssetBalanceDiff, blockID: block.BlockSignature}}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr}
	minerBalanceDiff := int64(tx.Fee)
	changes[2] = balanceChange{minerKey.bytes(), balanceDiff{balance: minerBalanceDiff, blockID: block.BlockSignature}}
	if err := tv.pushChanges(changes, block); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateBurn(tx *proto.Burn, block, parent *proto.Block, initialisation bool) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, block.Timestamp, parent.Timestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	// Modify asset.
	change := &assetBurnChange{
		diff:    tx.Amount,
		blockID: block.BlockSignature,
	}
	if err := tv.assets.burnAsset(tx.AssetID, change); err != nil {
		return false, errors.Wrap(err, "failed to burn asset")
	}
	changes := make([]balanceChange, 3)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	changes[0] = balanceChange{senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff, blockID: block.BlockSignature}}
	senderAssetKey := balanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := -int64(tx.Amount)
	changes[1] = balanceChange{senderAssetKey.bytes(), balanceDiff{balance: senderAssetBalanceDiff, blockID: block.BlockSignature}}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr}
	minerBalanceDiff := int64(tx.Fee)
	changes[2] = balanceChange{minerKey.bytes(), balanceDiff{balance: minerBalanceDiff, blockID: block.BlockSignature}}
	if err := tv.pushChanges(changes, block); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateExchange(tx proto.Exchange, block, parent *proto.Block, initialisation bool) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.GetTimestamp(), block.Timestamp, parent.Timestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return false, err
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return false, err
	}
	// Check assets.
	if err := tv.checkAsset(&sellOrder.AssetPair.AmountAsset); err != nil {
		return false, err
	}
	if err := tv.checkAsset(&sellOrder.AssetPair.PriceAsset); err != nil {
		return false, err
	}
	// Perform exchange.
	var val, amount, price big.Int
	priceConst := big.NewInt(priceConstant)
	amount.SetUint64(tx.GetAmount())
	price.SetUint64(tx.GetPrice())
	val.Mul(&amount, &price)
	val.Quo(&val, priceConst)
	if !val.IsInt64() {
		return false, errors.New("price * amount exceeds MaxInt64")
	}
	priceDiff := val.Int64()
	amountDiff := int64(tx.GetAmount())
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, sellOrder.SenderPK)
	if err != nil {
		return false, err
	}
	changes := make([]balanceChange, 8)
	senderPriceKey := balanceKey{address: senderAddr, asset: sellOrder.AssetPair.PriceAsset.ToID()}
	changes[0] = balanceChange{senderPriceKey.bytes(), balanceDiff{balance: priceDiff, blockID: block.BlockSignature}}
	senderAmountKey := balanceKey{address: senderAddr, asset: sellOrder.AssetPair.AmountAsset.ToID()}
	changes[1] = balanceChange{senderAmountKey.bytes(), balanceDiff{balance: -amountDiff, blockID: block.BlockSignature}}
	senderFeeKey := balanceKey{address: senderAddr}
	senderFeeDiff := -int64(tx.GetSellMatcherFee())
	changes[2] = balanceChange{senderFeeKey.bytes(), balanceDiff{balance: senderFeeDiff, blockID: block.BlockSignature}}
	receiverAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, buyOrder.SenderPK)
	if err != nil {
		return false, err
	}
	receiverPriceKey := balanceKey{address: receiverAddr, asset: sellOrder.AssetPair.PriceAsset.ToID()}
	changes[3] = balanceChange{receiverPriceKey.bytes(), balanceDiff{balance: -priceDiff, blockID: block.BlockSignature}}
	receiverAmountKey := balanceKey{address: receiverAddr, asset: sellOrder.AssetPair.AmountAsset.ToID()}
	changes[4] = balanceChange{receiverAmountKey.bytes(), balanceDiff{balance: amountDiff, blockID: block.BlockSignature}}
	receiverFeeKey := balanceKey{address: receiverAddr}
	receiverFeeDiff := -int64(tx.GetBuyMatcherFee())
	changes[5] = balanceChange{receiverFeeKey.bytes(), balanceDiff{balance: receiverFeeDiff, blockID: block.BlockSignature}}
	// Update matcher.
	matcherAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, buyOrder.MatcherPK)
	if err != nil {
		return false, err
	}
	matcherKey := balanceKey{address: matcherAddr}
	matcherFee, err := util.AddInt64(int64(tx.GetBuyMatcherFee()), int64(tx.GetSellMatcherFee()))
	if err != nil {
		return false, err
	}
	matcherBalanceDiff := matcherFee - int64(tx.GetFee())
	changes[6] = balanceChange{matcherKey.bytes(), balanceDiff{balance: matcherBalanceDiff, blockID: block.BlockSignature}}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr}
	minerBalanceDiff := int64(tx.GetFee())
	changes[7] = balanceChange{minerKey.bytes(), balanceDiff{balance: minerBalanceDiff, blockID: block.BlockSignature}}
	if err := tv.pushChanges(changes, block); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateLease(tx *proto.Lease, block, parent *proto.Block, initialisation bool) (bool, error) {
	return false, nil
}

func (tv *transactionValidator) validateLeaseCancel(tx *proto.LeaseCancel, block, parent *proto.Block, initialisation bool) (bool, error) {
	return false, nil
}

func (tv *transactionValidator) validateTransaction(block, parent *proto.Block, tx proto.Transaction, initialisation bool) error {
	switch v := tx.(type) {
	case *proto.Genesis:
		if ok, err := tv.validateGenesis(v, block, initialisation); !ok {
			return errors.Wrap(err, "genesis validation failed")
		}
	case *proto.Payment:
		if ok, err := tv.validatePayment(v, block, parent, initialisation); !ok {
			return errors.Wrap(err, "payment validation failed")
		}
	case *proto.TransferV1:
		if ok, err := tv.validateTransfer(&v.Transfer, block, parent, initialisation); !ok {
			return errors.Wrap(err, "transferv1 validation failed")
		}
	case *proto.TransferV2:
		if ok, err := tv.validateTransfer(&v.Transfer, block, parent, initialisation); !ok {
			return errors.Wrap(err, "transferv2 validation failed")
		}
	case *proto.IssueV1:
		if ok, err := tv.validateIssue(v, block, parent, initialisation); !ok {
			return errors.Wrap(err, "issuev1 validation failed")
		}
	case *proto.IssueV2:
		if ok, err := tv.validateIssue(v, block, parent, initialisation); !ok {
			return errors.Wrap(err, "issuev2 validation failed")
		}
	case *proto.ReissueV1:
		if ok, err := tv.validateReissue(&v.Reissue, block, parent, initialisation); !ok {
			return errors.Wrap(err, "reissuev1 validation failed")
		}
	case *proto.ReissueV2:
		if ok, err := tv.validateReissue(&v.Reissue, block, parent, initialisation); !ok {
			return errors.Wrap(err, "reissuev2 validation failed")
		}
	case *proto.BurnV1:
		if ok, err := tv.validateBurn(&v.Burn, block, parent, initialisation); !ok {
			return errors.Wrap(err, "burnv1 validation failed")
		}
	case *proto.BurnV2:
		if ok, err := tv.validateBurn(&v.Burn, block, parent, initialisation); !ok {
			return errors.Wrap(err, "burnv2 validation failed")
		}
	case *proto.ExchangeV1:
		if ok, err := tv.validateExchange(v, block, parent, initialisation); !ok {
			return errors.Wrap(err, "exchangev1 validation failed")
		}
	case *proto.ExchangeV2:
		if ok, err := tv.validateExchange(v, block, parent, initialisation); !ok {
			return errors.Wrap(err, "exchange2 validation failed")
		}
	case *proto.LeaseV1:
		if ok, err := tv.validateLease(&v.Lease, block, parent, initialisation); !ok {
			return errors.Wrap(err, "leasev1 validation failed")
		}
	case *proto.LeaseV2:
		if ok, err := tv.validateLease(&v.Lease, block, parent, initialisation); !ok {
			return errors.Wrap(err, "leasev2 validation failed")
		}
	case *proto.LeaseCancelV1:
		if ok, err := tv.validateLeaseCancel(&v.LeaseCancel, block, parent, initialisation); !ok {
			return errors.Wrap(err, "leasecancelv1 validation failed")
		}
	case *proto.LeaseCancelV2:
		if ok, err := tv.validateLeaseCancel(&v.LeaseCancel, block, parent, initialisation); !ok {
			return errors.Wrap(err, "leasecancelv2 validation failed")
		}
	default:
		return errors.Errorf("transaction type %T is not supported\n", v)
	}
	return nil
}

func (tv *transactionValidator) performTransactions() error {
	return tv.changesStor.applyDeltas()
}

func (tv *transactionValidator) reset() {
	tv.changesStor.reset()
}
