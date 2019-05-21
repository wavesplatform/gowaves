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
)

func byteKey(addr proto.Address, assetID []byte) []byte {
	if assetID == nil {
		k := wavesBalanceKey{addr}
		return k.bytes()
	}
	k := assetBalanceKey{addr, assetID}
	return k.bytes()
}

type balanceDiff struct {
	// Exception for Exchange transactions which can result in temporary negative balance.
	allowTempNegative   bool
	allowLeasedTransfer bool
	balance             int64
	leaseIn             int64
	leaseOut            int64
	blockID             crypto.Signature
}

func (diff *balanceDiff) spendableBalanceDiff() int64 {
	return diff.balance - diff.leaseOut
}

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
	// minBalanceDiff is diff which produces minimal spendable (taking leasing into account) balance value.
	// This is needed to check for negative balances.
	// For blocks when temporary negative balances are possible, this value is ignored.
	minBalanceDiff balanceDiff
}

func (ch *balanceChanges) updateMinBalanceDiff(newDiff balanceDiff, checkTempNegative bool) {
	allowNegForDiff := newDiff.allowTempNegative
	if checkTempNegative && !allowNegForDiff {
		// Check every tx, minBalanceDiff will have mimimum diff value among all txs at the end.
		if newDiff.spendableBalanceDiff() < ch.minBalanceDiff.spendableBalanceDiff() {
			ch.minBalanceDiff = newDiff
		}
	}
}

// Similar to update() but doesn't work with block IDs.
func (ch *balanceChanges) updateDirectly(newDiff balanceDiff, checkTempNegative bool) error {
	last := len(ch.balanceDiffs) - 1
	lastDiff := balanceDiff{}
	if last >= 0 {
		lastDiff = ch.balanceDiffs[last]
	}
	if err := newDiff.add(&lastDiff); err != nil {
		return errors.Errorf("failed to add diffs: %v\n", err)
	}
	if last >= 0 {
		ch.balanceDiffs[last] = newDiff
	} else {
		ch.balanceDiffs = append(ch.balanceDiffs, newDiff)
	}
	ch.updateMinBalanceDiff(newDiff, checkTempNegative)
	return nil
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
	} else if last >= 0 {
		ch.balanceDiffs[last] = newDiff
	} else {
		return errors.New("empty balance diffs slice and can not append the first diff")
	}
	ch.updateMinBalanceDiff(newDiff, checkTempNegative)
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

type wavesBalanceKeyFixed [wavesBalanceKeySize]byte
type assetBalanceKeyFixed [assetBalanceKeySize]byte

type changesStorage struct {
	balances  *balances
	deltas    []balanceChanges
	wavesKeys map[wavesBalanceKeyFixed]int // waves key --> index in deltas.
	assetKeys map[assetBalanceKeyFixed]int // asset key --> index in deltas.
}

func newChangesStorage(balances *balances) (*changesStorage, error) {
	return &changesStorage{
		balances:  balances,
		wavesKeys: make(map[wavesBalanceKeyFixed]int),
		assetKeys: make(map[assetBalanceKeyFixed]int),
	}, nil
}

func (bs *changesStorage) balanceChanges(key []byte) (*balanceChanges, error) {
	size := len(key)
	if size == wavesBalanceKeySize {
		var wavesKey wavesBalanceKeyFixed
		copy(wavesKey[:], key)
		if _, ok := bs.wavesKeys[wavesKey]; !ok {
			bs.wavesKeys[wavesKey] = len(bs.deltas)
			bs.deltas = append(bs.deltas, balanceChanges{key: key})
		}
		return &bs.deltas[bs.wavesKeys[wavesKey]], nil
	} else if size == assetBalanceKeySize {
		var assetKey assetBalanceKeyFixed
		copy(assetKey[:], key)
		if _, ok := bs.assetKeys[assetKey]; !ok {
			bs.assetKeys[assetKey] = len(bs.deltas)
			bs.deltas = append(bs.deltas, balanceChanges{key: key})
		}
		return &bs.deltas[bs.assetKeys[assetKey]], nil
	}
	return nil, errors.New("invalid key size")
}

func (bs *changesStorage) validateWavesChange(change *balanceChanges, filter, perform bool) error {
	var k wavesBalanceKey
	if err := k.unmarshal(change.key); err != nil {
		return errors.Errorf("failed to unmarshal waves balance key: %v\n", err)
	}
	profile, err := bs.balances.wavesBalance(k.address, filter)
	if err != nil {
		return errors.Errorf("failed to retrieve waves balance: %v\n", err)
	}
	// Check for negative balance.
	if _, err := change.minBalanceDiff.applyTo(profile); err != nil {
		return errors.Errorf("minimum balance diff produces invalid result: %v\n", err)
	}
	for _, diff := range change.balanceDiffs {
		newProfile, err := diff.applyTo(profile)
		if err != nil {
			return errors.Errorf("failed to apply waves balance change: %v\n", err)
		}
		if !perform {
			continue
		}
		r := &wavesBalanceRecord{*newProfile, diff.blockID}
		if err := bs.balances.setWavesBalance(k.address, r); err != nil {
			return errors.Errorf("failed to set account balance: %v\n", err)
		}
	}
	return nil
}

func (bs *changesStorage) validateAssetChange(change *balanceChanges, filter, perform bool) error {
	var k assetBalanceKey
	if err := k.unmarshal(change.key); err != nil {
		return errors.Errorf("failed to unmarshal asset balance key: %v\n", err)
	}
	balance, err := bs.balances.assetBalance(k.address, k.asset, filter)
	if err != nil {
		return errors.Errorf("failed to retrieve asset balance: %v\n", err)
	}
	// Check for negative balance.
	minBalance, err := util.AddInt64(int64(balance), change.minBalanceDiff.balance)
	if err != nil {
		return errors.Errorf("failed to add balances: %v\n", err)
	}
	if minBalance < 0 {
		return errors.New("validation failed: negative asset balance")
	}
	for _, diff := range change.balanceDiffs {
		newBalance, err := util.AddInt64(int64(balance), diff.balance)
		if err != nil {
			return errors.Errorf("failed to add balances: %v\n", err)
		}
		if newBalance < 0 {
			return errors.New("validation failed: negative asset balance")
		}
		if !perform {
			continue
		}
		r := &assetBalanceRecord{uint64(newBalance), diff.blockID}
		if err := bs.balances.setAssetBalance(k.address, k.asset, r); err != nil {
			return errors.Errorf("failed to set asset balance: %v\n", err)
		}
	}
	return nil
}

func (bs *changesStorage) validateDelta(delta *balanceChanges, filter, perform bool) error {
	if len(delta.key) > wavesBalanceKeySize {
		// Is asset change.
		if err := bs.validateAssetChange(delta, filter, perform); err != nil {
			return err
		}
	} else {
		// Is Waves change, need to take leasing into account.
		if err := bs.validateWavesChange(delta, filter, perform); err != nil {
			return err
		}
	}
	return nil
}

// Apply all balance changes (actually move them to DB batch) and reset.
func (bs *changesStorage) validateDeltas(filter, perform bool) error {
	// Apply and validate balance variations.
	// At first, sort all changes by addresses they do modify.
	// That's *very* important optimization, since levelDB stores data
	// sorted by keys, and the idea is to read in sorted order.
	// We save a lot of time on disk's seek time.
	// TODO: if DB supported MultiGet() operation, this would probably be even faster.
	sort.Sort(byKey(bs.deltas))
	for _, delta := range bs.deltas {
		if err := bs.validateDelta(&delta, filter, perform); err != nil {
			return err
		}
	}
	bs.reset()
	return nil
}

func (bs *changesStorage) reset() {
	bs.deltas = nil
	bs.wavesKeys = make(map[wavesBalanceKeyFixed]int)
	bs.assetKeys = make(map[assetBalanceKeyFixed]int)

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

func (tv *transactionValidator) addChange(key []byte, diff balanceDiff, currentTimestamp uint64, validate bool) (bool, error) {
	changes, err := tv.changesStor.balanceChanges(key)
	if err != nil {
		return false, errors.Wrap(err, "can not retrieve balance changes")
	}
	changesCopy := *changes
	changesCopy.minBalanceDiff.allowLeasedTransfer = diff.allowLeasedTransfer
	checkTempNegative := tv.checkNegativeBalance(currentTimestamp)
	if diff.blockID != (crypto.Signature{}) {
		if err := changesCopy.update(diff, checkTempNegative); err != nil {
			return false, errors.Wrap(err, "can not update balance changes")
		}
	} else {
		if err := changesCopy.updateDirectly(diff, checkTempNegative); err != nil {
			return false, errors.Wrap(err, "can not update balance changes")
		}
	}
	if validate {
		// Validate immediately, without waiting for validateTransactions() call.
		if err := tv.changesStor.validateDelta(&changesCopy, true, false); err != nil {
			return false, errors.Wrap(err, "changes validation failed")
		}
	}
	// Save changes at the end if validation was successful.
	*changes = changesCopy
	return true, nil
}

type balanceChange struct {
	key  []byte
	diff balanceDiff
}

type txValidationInfo struct {
	perform          bool
	initialisation   bool
	validate         bool
	currentTimestamp uint64
	parentTimestamp  uint64
	minerPK          crypto.PublicKey
	blockID          crypto.Signature
}

func (i *txValidationInfo) hasMiner() bool {
	return i.minerPK != (crypto.PublicKey{})
}

func (tv *transactionValidator) pushChanges(changes []balanceChange, info *txValidationInfo) error {
	for _, ch := range changes {
		allowLeasedTransfer := true
		if info.currentTimestamp >= tv.settings.AllowLeasedBalanceTransferUntilTime {
			allowLeasedTransfer = false
		}
		ch.diff.allowLeasedTransfer = allowLeasedTransfer
		ch.diff.blockID = info.blockID
		if ok, err := tv.addChange(ch.key, ch.diff, info.currentTimestamp, info.validate); !ok {
			return err
		}
	}
	return nil
}

func (tv *transactionValidator) validateGenesis(tx *proto.Genesis, info *txValidationInfo) (bool, error) {
	if info.blockID != tv.genesis {
		return false, errors.New("genesis transaction inside of non-genesis block")
	}
	if !info.initialisation {
		return false, errors.New("genesis transaction in non-initialisation mode")
	}
	changes := make([]balanceChange, 1)
	key := wavesBalanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	changes[0] = balanceChange{key.bytes(), balanceDiff{balance: receiverBalanceDiff}}
	if err := tv.pushChanges(changes, info); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validatePayment(tx *proto.Payment, info *txValidationInfo) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	changes := make([]balanceChange, 2)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderKey := wavesBalanceKey{address: senderAddr}
	senderBalanceDiff := -int64(tx.Amount) - int64(tx.Fee)
	changes[0] = balanceChange{senderKey.bytes(), balanceDiff{balance: senderBalanceDiff}}
	// Update receiver.
	receiverKey := wavesBalanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	changes[1] = balanceChange{receiverKey.bytes(), balanceDiff{balance: receiverBalanceDiff}}
	if info.hasMiner() {
		// Update miner.
		minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
		if err != nil {
			return false, err
		}
		minerKey := wavesBalanceKey{address: minerAddr}
		minerBalanceDiff := int64(tx.Fee)
		changes = append(changes, balanceChange{minerKey.bytes(), balanceDiff{balance: minerBalanceDiff}})
	}
	if err := tv.pushChanges(changes, info); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) checkAsset(asset *proto.OptionalAsset, initialisation bool) error {
	if !asset.Present {
		// Waves always valid.
		return nil
	}
	if _, err := tv.assets.newestAssetRecord(asset.ID, !initialisation); err != nil {
		return errors.New("unknown asset")
	}
	return nil
}

func (tv *transactionValidator) validateTransfer(tx *proto.Transfer, info *txValidationInfo) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	if err := tv.checkAsset(&tx.AmountAsset, info.initialisation); err != nil {
		return false, err
	}
	if err := tv.checkAsset(&tx.FeeAsset, info.initialisation); err != nil {
		return false, err
	}
	changes := make([]balanceChange, 3)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := byteKey(senderAddr, tx.FeeAsset.ToID())
	senderFeeBalanceDiff := -int64(tx.Fee)
	changes[0] = balanceChange{senderFeeKey, balanceDiff{balance: senderFeeBalanceDiff}}
	senderAmountKey := byteKey(senderAddr, tx.AmountAsset.ToID())
	senderAmountBalanceDiff := -int64(tx.Amount)
	changes[1] = balanceChange{senderAmountKey, balanceDiff{balance: senderAmountBalanceDiff}}
	// Update receiver.
	if tx.Recipient.Address == nil {
		// TODO support aliases.
		return false, errors.New("alias without address is not supported yet")
	}
	receiverKey := byteKey(*tx.Recipient.Address, tx.AmountAsset.ToID())
	receiverBalanceDiff := int64(tx.Amount)
	changes[2] = balanceChange{receiverKey, balanceDiff{balance: receiverBalanceDiff}}
	if info.hasMiner() {
		// Update miner.
		minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
		if err != nil {
			return false, err
		}
		minerKey := byteKey(minerAddr, tx.FeeAsset.ToID())
		minerBalanceDiff := int64(tx.Fee)
		changes = append(changes, balanceChange{minerKey, balanceDiff{balance: minerBalanceDiff}})
	}
	if err := tv.pushChanges(changes, info); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateIssue(tx *proto.Issue, id []byte, info *txValidationInfo) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	assetID, err := crypto.NewDigestFromBytes(id)
	if err != nil {
		return false, err
	}
	if info.perform {
		// Create new asset.
		asset := &assetInfo{
			assetConstInfo: assetConstInfo{
				name:        tx.Name,
				description: tx.Description,
				decimals:    int8(tx.Decimals),
			},
			assetHistoryRecord: assetHistoryRecord{
				quantity:   *big.NewInt(int64(tx.Quantity)),
				reissuable: tx.Reissuable,
				blockID:    info.blockID,
			},
		}
		if err := tv.assets.issueAsset(assetID, asset); err != nil {
			return false, errors.Wrap(err, "failed to issue asset")
		}
	}
	changes := make([]balanceChange, 2)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	changes[0] = balanceChange{senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff}}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: assetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	changes[1] = balanceChange{senderAssetKey.bytes(), balanceDiff{balance: senderAssetBalanceDiff}}
	if info.hasMiner() {
		// Update miner.
		minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
		if err != nil {
			return false, err
		}
		minerKey := byteKey(minerAddr, nil)
		minerBalanceDiff := int64(tx.Fee)
		changes = append(changes, balanceChange{minerKey, balanceDiff{balance: minerBalanceDiff}})
	}
	if err := tv.pushChanges(changes, info); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateReissue(tx *proto.Reissue, info *txValidationInfo) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	// Check if it's "legal" to modify given asset.
	record, err := tv.assets.newestAssetRecord(tx.AssetID, !info.initialisation)
	if err != nil {
		return false, err
	}
	if (info.currentTimestamp > tv.settings.InvalidReissueInSameBlockUntilTime) && !record.reissuable {
		return false, errors.New("attempt to reissue asset which is not reissuable")
	}
	if info.perform {
		// Modify asset.
		change := &assetReissueChange{
			reissuable: tx.Reissuable,
			diff:       int64(tx.Quantity),
			blockID:    info.blockID,
		}
		if err := tv.assets.reissueAsset(tx.AssetID, change, !info.initialisation); err != nil {
			return false, errors.Wrap(err, "failed to reissue asset")
		}
	}
	changes := make([]balanceChange, 2)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	changes[0] = balanceChange{senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff}}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	changes[1] = balanceChange{senderAssetKey.bytes(), balanceDiff{balance: senderAssetBalanceDiff}}
	if info.hasMiner() {
		// Update miner.
		minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
		if err != nil {
			return false, err
		}
		minerKey := byteKey(minerAddr, nil)
		minerBalanceDiff := int64(tx.Fee)
		changes = append(changes, balanceChange{minerKey, balanceDiff{balance: minerBalanceDiff}})
	}
	if err := tv.pushChanges(changes, info); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateBurn(tx *proto.Burn, info *txValidationInfo) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	if info.perform {
		// Modify asset.
		change := &assetBurnChange{
			diff:    int64(tx.Amount),
			blockID: info.blockID,
		}
		if err := tv.assets.burnAsset(tx.AssetID, change, !info.initialisation); err != nil {
			return false, errors.Wrap(err, "failed to burn asset")
		}
	}
	changes := make([]balanceChange, 2)
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	changes[0] = balanceChange{senderFeeKey.bytes(), balanceDiff{balance: senderFeeBalanceDiff}}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := -int64(tx.Amount)
	changes[1] = balanceChange{senderAssetKey.bytes(), balanceDiff{balance: senderAssetBalanceDiff}}
	if info.hasMiner() {
		// Update miner.
		minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
		if err != nil {
			return false, err
		}
		minerKey := byteKey(minerAddr, nil)
		minerBalanceDiff := int64(tx.Fee)
		changes = append(changes, balanceChange{minerKey, balanceDiff{balance: minerBalanceDiff}})
	}
	if err := tv.pushChanges(changes, info); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateExchange(tx proto.Exchange, info *txValidationInfo) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.GetTimestamp(), info.currentTimestamp, info.parentTimestamp); !ok {
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
	if err := tv.checkAsset(&sellOrder.AssetPair.AmountAsset, info.initialisation); err != nil {
		return false, err
	}
	if err := tv.checkAsset(&sellOrder.AssetPair.PriceAsset, info.initialisation); err != nil {
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
	changes := make([]balanceChange, 7)
	senderPriceKey := byteKey(senderAddr, sellOrder.AssetPair.PriceAsset.ToID())
	changes[0] = balanceChange{senderPriceKey, balanceDiff{balance: priceDiff}}
	senderAmountKey := byteKey(senderAddr, sellOrder.AssetPair.AmountAsset.ToID())
	changes[1] = balanceChange{senderAmountKey, balanceDiff{allowTempNegative: true, balance: -amountDiff}}
	senderFeeKey := wavesBalanceKey{senderAddr}
	senderFeeDiff := -int64(tx.GetSellMatcherFee())
	changes[2] = balanceChange{senderFeeKey.bytes(), balanceDiff{balance: senderFeeDiff}}
	receiverAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, buyOrder.SenderPK)
	if err != nil {
		return false, err
	}
	receiverPriceKey := byteKey(receiverAddr, sellOrder.AssetPair.PriceAsset.ToID())
	changes[3] = balanceChange{receiverPriceKey, balanceDiff{allowTempNegative: true, balance: -priceDiff}}
	receiverAmountKey := byteKey(receiverAddr, sellOrder.AssetPair.AmountAsset.ToID())
	changes[4] = balanceChange{receiverAmountKey, balanceDiff{balance: amountDiff}}
	receiverFeeKey := wavesBalanceKey{receiverAddr}
	receiverFeeDiff := -int64(tx.GetBuyMatcherFee())
	changes[5] = balanceChange{receiverFeeKey.bytes(), balanceDiff{balance: receiverFeeDiff}}
	// Update matcher.
	matcherAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, buyOrder.MatcherPK)
	if err != nil {
		return false, err
	}
	matcherKey := wavesBalanceKey{matcherAddr}
	matcherFee, err := util.AddInt64(int64(tx.GetBuyMatcherFee()), int64(tx.GetSellMatcherFee()))
	if err != nil {
		return false, err
	}
	matcherBalanceDiff := matcherFee - int64(tx.GetFee())
	changes[6] = balanceChange{matcherKey.bytes(), balanceDiff{balance: matcherBalanceDiff}}
	if info.hasMiner() {
		// Update miner.
		minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
		if err != nil {
			return false, err
		}
		minerKey := byteKey(minerAddr, nil)
		minerBalanceDiff := int64(tx.GetFee())
		changes = append(changes, balanceChange{minerKey, balanceDiff{balance: minerBalanceDiff}})
	}
	if err := tv.pushChanges(changes, info); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateLease(tx *proto.Lease, id *crypto.Digest, info *txValidationInfo) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	if senderAddr == *tx.Recipient.Address {
		return false, errors.New("trying to lease money to self")
	}
	changes := make([]balanceChange, 3)
	// Update sender.
	senderKey := wavesBalanceKey{address: senderAddr}
	senderLeaseOutDiff := int64(tx.Amount)
	changes[0] = balanceChange{senderKey.bytes(), balanceDiff{leaseOut: senderLeaseOutDiff}}
	senderFeeDiff := -int64(tx.Fee)
	changes[1] = balanceChange{senderKey.bytes(), balanceDiff{balance: senderFeeDiff}}
	// Update receiver.
	if tx.Recipient.Address == nil {
		// TODO support aliases.
		return false, errors.New("alias without address is not supported yet")
	}
	receiverKey := wavesBalanceKey{address: *tx.Recipient.Address}
	receiverLeaseInDiff := int64(tx.Amount)
	changes[2] = balanceChange{receiverKey.bytes(), balanceDiff{leaseIn: receiverLeaseInDiff}}
	if info.hasMiner() {
		// Update miner.
		minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
		if err != nil {
			return false, err
		}
		minerKey := wavesBalanceKey{address: minerAddr}
		minerBalanceDiff := int64(tx.Fee)
		changes = append(changes, balanceChange{minerKey.bytes(), balanceDiff{balance: minerBalanceDiff}})
	}
	if err := tv.pushChanges(changes, info); err != nil {
		return false, err
	}
	if info.perform {
		// Add leasing to lease state.
		r := &leasingRecord{
			leasing{true, tx.Amount, *tx.Recipient.Address, senderAddr},
			info.blockID,
		}
		if err := tv.leases.addLeasing(*id, r); err != nil {
			return false, errors.Wrap(err, "failed to add leasing")
		}
	}
	return true, nil
}

func (tv *transactionValidator) validateLeaseCancel(tx *proto.LeaseCancel, info *txValidationInfo) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	l, err := tv.leases.newestLeasingInfo(tx.LeaseID, !info.initialisation)
	if err != nil {
		return false, err
	}
	if !l.isActive && (info.currentTimestamp > tv.settings.AllowMultipleLeaseCancelUntilTime) {
		return false, errors.New("can not cancel lease which has already been cancelled")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	if (l.sender != senderAddr) && (info.currentTimestamp > tv.settings.AllowMultipleLeaseCancelUntilTime) {
		return false, errors.New("sender of LeaseCancel is not sender of corresponding Lease")
	}
	if info.perform {
		if err := tv.leases.cancelLeasing(tx.LeaseID, info.blockID, !info.initialisation); err != nil {
			return false, errors.Wrap(err, "failed to cancel leasing")
		}
	}
	changes := make([]balanceChange, 3)
	// Update sender.
	senderKey := wavesBalanceKey{address: senderAddr}
	senderLeaseOutDiff := -int64(l.leaseAmount)
	changes[0] = balanceChange{senderKey.bytes(), balanceDiff{leaseOut: senderLeaseOutDiff}}
	senderFeeDiff := -int64(tx.Fee)
	changes[1] = balanceChange{senderKey.bytes(), balanceDiff{balance: senderFeeDiff}}
	// Update receiver.
	receiverKey := wavesBalanceKey{address: l.recipient}
	receiverLeaseInDiff := -int64(l.leaseAmount)
	changes[2] = balanceChange{receiverKey.bytes(), balanceDiff{leaseIn: receiverLeaseInDiff}}
	if info.hasMiner() {
		// Update miner.
		minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
		if err != nil {
			return false, err
		}
		minerKey := wavesBalanceKey{address: minerAddr}
		minerBalanceDiff := int64(tx.Fee)
		changes = append(changes, balanceChange{minerKey.bytes(), balanceDiff{balance: minerBalanceDiff}})
	}
	if err := tv.pushChanges(changes, info); err != nil {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) addTxForValidation(tx proto.Transaction, info *txValidationInfo) error {
	switch v := tx.(type) {
	case *proto.Genesis:
		if ok, err := tv.validateGenesis(v, info); !ok {
			return errors.Wrap(err, "genesis validation failed")
		}
	case *proto.Payment:
		if ok, err := tv.validatePayment(v, info); !ok {
			return errors.Wrap(err, "payment validation failed")
		}
	case *proto.TransferV1:
		if ok, err := tv.validateTransfer(&v.Transfer, info); !ok {
			return errors.Wrap(err, "transferv1 validation failed")
		}
	case *proto.TransferV2:
		if ok, err := tv.validateTransfer(&v.Transfer, info); !ok {
			return errors.Wrap(err, "transferv2 validation failed")
		}
	case *proto.IssueV1:
		if ok, err := tv.validateIssue(&v.Issue, tx.GetID(), info); !ok {
			return errors.Wrap(err, "issuev1 validation failed")
		}
	case *proto.IssueV2:
		if ok, err := tv.validateIssue(&v.Issue, tx.GetID(), info); !ok {
			return errors.Wrap(err, "issuev2 validation failed")
		}
	case *proto.ReissueV1:
		if ok, err := tv.validateReissue(&v.Reissue, info); !ok {
			return errors.Wrap(err, "reissuev1 validation failed")
		}
	case *proto.ReissueV2:
		if ok, err := tv.validateReissue(&v.Reissue, info); !ok {
			return errors.Wrap(err, "reissuev2 validation failed")
		}
	case *proto.BurnV1:
		if ok, err := tv.validateBurn(&v.Burn, info); !ok {
			return errors.Wrap(err, "burnv1 validation failed")
		}
	case *proto.BurnV2:
		if ok, err := tv.validateBurn(&v.Burn, info); !ok {
			return errors.Wrap(err, "burnv2 validation failed")
		}
	case *proto.ExchangeV1:
		if ok, err := tv.validateExchange(v, info); !ok {
			return errors.Wrap(err, "exchangev1 validation failed")
		}
	case *proto.ExchangeV2:
		if ok, err := tv.validateExchange(v, info); !ok {
			return errors.Wrap(err, "exchange2 validation failed")
		}
	case *proto.LeaseV1:
		if ok, err := tv.validateLease(&v.Lease, v.ID, info); !ok {
			return errors.Wrap(err, "leasev1 validation failed")
		}
	case *proto.LeaseV2:
		if ok, err := tv.validateLease(&v.Lease, v.ID, info); !ok {
			return errors.Wrap(err, "leasev2 validation failed")
		}
	case *proto.LeaseCancelV1:
		if ok, err := tv.validateLeaseCancel(&v.LeaseCancel, info); !ok {
			return errors.Wrap(err, "leasecancelv1 validation failed")
		}
	case *proto.LeaseCancelV2:
		if ok, err := tv.validateLeaseCancel(&v.LeaseCancel, info); !ok {
			return errors.Wrap(err, "leasecancelv2 validation failed")
		}
	default:
		return errors.Errorf("transaction type %T is not supported\n", v)
	}
	return nil
}

func (tv *transactionValidator) validateTransactions(initialisation, perform bool) error {
	return tv.changesStor.validateDeltas(!initialisation, perform)
}

func (tv *transactionValidator) reset() {
	tv.changesStor.reset()
}
