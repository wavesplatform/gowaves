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
	// priceConstant is used for exchange calculations.
	priceConstant = 10e7
	// ngCurrentBlockFeePercentage is percentage of fees miner gets from the current block after activating NG (40%).
	// It is represented as (2 / 5), to make it compatible with Scala implementation.
	ngCurrentBlockFeePercentageDivider  = 5
	ngCurrentBlockFeePercentageDividend = 2
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
// Transaction may produce one or more balance diffs.
// Each address among tx participants may also have one or more diffs within this tx.
// For instance, paying transaction fee in Waves and sending Waves are two separate diffs for same address in Transfer/Payment tx.
type balanceDiff struct {
	allowLeasedTransfer bool
	// Balance change.
	balance int64
	// LeaseIn change.
	leaseIn int64
	// LeaseOut change.
	leaseOut int64
	// blockID when this change takes place.
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

// balanceChanges is a full collection of changes for given key.
// balanceDiffs is slice of per-block cumulative diffs.
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

// newBalanceChanges() constructs new balanceChanges from the first balance diff.
func newBalanceChanges(key []byte, diff balanceDiff) *balanceChanges {
	return &balanceChanges{key, []balanceDiff{diff}, diff}
}

func (ch *balanceChanges) safeCopy() *balanceChanges {
	newChanges := &balanceChanges{}
	newChanges.key = make([]byte, len(ch.key))
	copy(newChanges.key[:], ch.key[:])
	newChanges.balanceDiffs = make([]balanceDiff, len(ch.balanceDiffs))
	copy(newChanges.balanceDiffs[:], ch.balanceDiffs[:])
	newChanges.minBalanceDiff = ch.minBalanceDiff
	return newChanges
}

func (ch *balanceChanges) updateMinBalanceDiff(newDiff balanceDiff) {
	// Check every tx, minBalanceDiff will have minimum diff value among all txs at the end.
	if newDiff.spendableBalanceDiff() < ch.minBalanceDiff.spendableBalanceDiff() {
		ch.minBalanceDiff = newDiff
	}
}

func (ch *balanceChanges) addDiff(newDiff balanceDiff) error {
	if len(ch.balanceDiffs) < 1 {
		return errors.New("trying to addDiff() to empty balanceChanges")
	}
	last := len(ch.balanceDiffs) - 1
	lastDiff := ch.balanceDiffs[last]
	if err := newDiff.add(&lastDiff); err != nil {
		return errors.Errorf("failed to add diffs: %v\n", err)
	}
	if newDiff.blockID != lastDiff.blockID {
		ch.balanceDiffs = append(ch.balanceDiffs, newDiff)
	} else {
		ch.balanceDiffs[last] = newDiff
	}
	ch.updateMinBalanceDiff(newDiff)
	return nil
}

type wavesBalanceKeyFixed [wavesBalanceKeySize]byte
type assetBalanceKeyFixed [assetBalanceKeySize]byte

type changesStorage struct {
	balances  *balances
	changes   []balanceChanges
	wavesKeys map[wavesBalanceKeyFixed]int // waves key --> index in changes.
	assetKeys map[assetBalanceKeyFixed]int // asset key --> index in changes.
}

func newChangesStorage(balances *balances) (*changesStorage, error) {
	return &changesStorage{
		balances:  balances,
		wavesKeys: make(map[wavesBalanceKeyFixed]int),
		assetKeys: make(map[assetBalanceKeyFixed]int),
	}, nil
}

func (bs *changesStorage) setBalanceChanges(changes *balanceChanges) error {
	key := changes.key
	size := len(key)
	if size == wavesBalanceKeySize {
		var wavesKey wavesBalanceKeyFixed
		copy(wavesKey[:], key)
		if index, ok := bs.wavesKeys[wavesKey]; ok {
			bs.changes[index] = *changes
		} else {
			bs.wavesKeys[wavesKey] = len(bs.changes)
			bs.changes = append(bs.changes, *changes)
		}
		return nil
	} else if size == assetBalanceKeySize {
		var assetKey assetBalanceKeyFixed
		copy(assetKey[:], key)
		if index, ok := bs.assetKeys[assetKey]; ok {
			bs.changes[index] = *changes
		} else {
			bs.assetKeys[assetKey] = len(bs.changes)
			bs.changes = append(bs.changes, *changes)
		}
		return nil
	}
	return errors.New("invalid key size")
}

func (bs *changesStorage) balanceChanges(key []byte) (*balanceChanges, error) {
	size := len(key)
	if size == wavesBalanceKeySize {
		var wavesKey wavesBalanceKeyFixed
		copy(wavesKey[:], key)
		index, ok := bs.wavesKeys[wavesKey]
		if !ok {
			return nil, errNotFound
		}
		return bs.changes[index].safeCopy(), nil
	} else if size == assetBalanceKeySize {
		var assetKey assetBalanceKeyFixed
		copy(assetKey[:], key)
		index, ok := bs.assetKeys[assetKey]
		if !ok {
			return nil, errNotFound
		}
		return bs.changes[index].safeCopy(), nil
	}
	return nil, errors.New("invalid key size")
}

// constructBalanceChanges() checks whether changes for given change key already exist, and adds new diff to them in such case.
// Otherwise, it creates fresh changes with the first diff equal to the argument.
func (bs *changesStorage) constructBalanceChanges(key []byte, diff balanceDiff) (*balanceChanges, error) {
	// Changes for this key are already in the stor, retrieve them.
	changes, err := bs.balanceChanges(key)
	if err == errNotFound {
		// Fresh changes with the first diff set.
		return newBalanceChanges(key, diff), nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "can not retrieve balance changes")
	}
	// Add new diff to existing changes.
	if err := changes.addDiff(diff); err != nil {
		return nil, errors.Wrap(err, "can not update balance changes")
	}
	return changes, nil
}

// addBalanceDiff() adds new balance diff to storage, validating it immediately before saving if necessarily.
func (bs *changesStorage) addBalanceDiff(key []byte, diff balanceDiff, validate bool) error {
	changes, err := bs.constructBalanceChanges(key, diff)
	if err != nil {
		return errors.Wrap(err, "failed to construct balance changes for given key and diff")
	}
	if validate {
		// Validate immediately, without waiting for validateTransactions() call.
		if err := bs.validateBalanceChanges(changes, true, false); err != nil {
			return errors.Wrap(err, "changes validation failed")
		}
	}
	// Save changes at the end if validation was successful / if immediate validation was not needed.
	if err := bs.setBalanceChanges(changes); err != nil {
		return errors.Wrap(err, "failed to save changes to changes storage")
	}
	return nil
}

func (bs *changesStorage) saveTxDiff(diff txDiff, validate bool) error {
	for key, balanceDiff := range diff {
		if err := bs.addBalanceDiff([]byte(key), balanceDiff, validate); err != nil {
			return err
		}
	}
	return nil
}

func (bs *changesStorage) validateWavesBalanceChanges(change *balanceChanges, filter, perform bool) error {
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
		return errors.Errorf("minimum balance diff for %s produces invalid result: %v\n", k.address.String(), err)
	}
	for _, diff := range change.balanceDiffs {
		// Check for negative balance.
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

func (bs *changesStorage) validateAssetBalanceChanges(change *balanceChanges, filter, perform bool) error {
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

func (bs *changesStorage) validateBalanceChanges(changes *balanceChanges, filter, perform bool) error {
	if len(changes.key) > wavesBalanceKeySize {
		// Is asset change.
		if err := bs.validateAssetBalanceChanges(changes, filter, perform); err != nil {
			return err
		}
	} else {
		// Is Waves change, need to take leasing into account.
		if err := bs.validateWavesBalanceChanges(changes, filter, perform); err != nil {
			return err
		}
	}
	return nil
}

type changesByKey []balanceChanges

func (k changesByKey) Len() int {
	return len(k)
}
func (k changesByKey) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}
func (k changesByKey) Less(i, j int) bool {
	return bytes.Compare(k[i].key, k[j].key) == -1
}

// Apply all balance changes (actually move them to balances in-memory storage) and reset.
func (bs *changesStorage) validateBalancesChanges(filter, perform bool) error {
	// Apply and validate balance variations.
	// At first, sort all changes by addresses they do modify.
	// LevelDB stores data sorted by keys, and the idea is to read in sorted order.
	// We save a lot of time on disk's seek time for hdd, and some time for ssd too (by reducing amount of reads).
	// TODO: if DB supported MultiGet() operation, this would probably be even faster.
	sort.Sort(changesByKey(bs.changes))
	for _, changes := range bs.changes {
		if err := bs.validateBalanceChanges(&changes, filter, perform); err != nil {
			return err
		}
	}
	bs.reset()
	return nil
}

func (bs *changesStorage) reset() {
	bs.changes = nil
	bs.wavesKeys = make(map[wavesBalanceKeyFixed]int)
	bs.assetKeys = make(map[assetBalanceKeyFixed]int)

}

type txValidationInfo struct {
	perform          bool
	initialisation   bool
	validate         bool
	currentTimestamp uint64
	parentTimestamp  uint64
	minerPK          crypto.PublicKey
	blockID          crypto.Signature
	prevBlockID      crypto.Signature
}

func (i *txValidationInfo) hasMiner() bool {
	return i.minerPK != (crypto.PublicKey{})
}

func (i *txValidationInfo) hasPrevBlock() bool {
	return i.prevBlockID != (crypto.Signature{})
}

type txDiff map[string]balanceDiff

func newTxDiff() txDiff {
	return make(txDiff)
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

type transactionValidator struct {
	genesis     crypto.Signature
	changesStor *changesStorage
	stor        *blockchainEntitiesStorage
	settings    *settings.BlockchainSettings
	// Current block's fee distribution.
	curDistr  feeDistribution
	blockFees map[crypto.Signature]feeDistribution
}

func newTransactionValidator(
	genesis crypto.Signature,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
) (*transactionValidator, error) {
	changesStor, err := newChangesStorage(stor.balances)
	if err != nil {
		return nil, errors.Errorf("failed to create balances changes storage: %v\n", err)
	}
	return &transactionValidator{
		genesis:     genesis,
		changesStor: changesStor,
		stor:        stor,
		settings:    settings,
		curDistr:    newFeeDistribution(),
		blockFees:   make(map[crypto.Signature]feeDistribution),
	}, nil
}

func (tv *transactionValidator) calculateCurrentBlockTxFee(txFee uint64) (uint64, error) {
	ngActivated, err := tv.stor.features.isActivated(int16(settings.NG))
	if err != nil {
		return 0, err
	}
	if ngActivated {
		return txFee / ngCurrentBlockFeePercentageDivider * ngCurrentBlockFeePercentageDividend, nil
	}
	return txFee, nil
}

func (tv *transactionValidator) prevBlockFeeDistr(prevBlock crypto.Signature) (*feeDistribution, error) {
	ngActivated, err := tv.stor.features.isActivated(int16(settings.NG))
	if err != nil {
		return nil, err
	}
	if !ngActivated {
		// If NG is not activated, miner does not get any fees from the previous block.
		return &feeDistribution{}, nil
	}
	ngActivationBlock, err := tv.stor.features.activationBlock(int16(settings.NG))
	if err != nil {
		return nil, err
	}
	if bytes.Compare(prevBlock[:], ngActivationBlock[:]) == 0 {
		// If the last block in current state is the NG activation block,
		// miner does not get any fees from this (last) block, because it was all taken by the last non-NG miner.
		return &feeDistribution{}, nil
	}
	if distr, ok := tv.blockFees[prevBlock]; ok {
		return &distr, nil
	}
	// Load from DB.
	return tv.stor.blocksInfo.feeDistribution(prevBlock)
}

func (tv *transactionValidator) checkFromFuture(timestamp uint64) bool {
	return timestamp > tv.settings.TxFromFutureCheckAfterTime
}

func (tv *transactionValidator) checkTxChangesSorted(timestamp uint64) bool {
	return timestamp > tv.settings.TxChangesSortedCheckAfterTime
}

func (tv *transactionValidator) checkTimestamps(txTimestamp, blockTimestamp, prevBlockTimestamp uint64) error {
	if txTimestamp < prevBlockTimestamp-tv.settings.MaxTxTimeBackOffset {
		return errors.New("early transaction creation time")
	}
	if tv.checkFromFuture(blockTimestamp) && txTimestamp > blockTimestamp+tv.settings.MaxTxTimeForwardOffset {
		return errors.New("late transaction creation time")
	}
	return nil
}

// curBlockBalanceDiff takes balanceDiff and appends additional info.
func (tv *transactionValidator) curBlockBalanceDiff(diff balanceDiff, info *txValidationInfo) balanceDiff {
	allowLeasedTransfer := true
	if info.currentTimestamp >= tv.settings.AllowLeasedBalanceTransferUntilTime {
		allowLeasedTransfer = false
	}
	diff.allowLeasedTransfer = allowLeasedTransfer
	diff.blockID = info.blockID
	return diff
}

// minerPayout adds current fee part of given tx to txDiff.
func (tv *transactionValidator) minerPayout(diff txDiff, fee uint64, info *txValidationInfo, feeAsset []byte) error {
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
	if err != nil {
		return err
	}
	minerKey := byteKey(minerAddr, feeAsset)
	minerBalanceDiff, err := tv.calculateCurrentBlockTxFee(fee)
	if err != nil {
		return err
	}
	if err := diff.appendBalanceDiff(minerKey, tv.curBlockBalanceDiff(balanceDiff{balance: int64(minerBalanceDiff)}, info)); err != nil {
		return err
	}
	// Count fees.
	if feeAsset == nil {
		tv.curDistr.totalWavesFees += fee
		tv.curDistr.currentWavesBlockFees += minerBalanceDiff
	} else {
		assetID, err := crypto.NewDigestFromBytes(feeAsset)
		if err != nil {
			return err
		}
		tv.curDistr.totalFees[assetID] += fee
		tv.curDistr.currentBlockFees[assetID] += minerBalanceDiff
	}
	return nil
}

func (tv *transactionValidator) createDiffGenesis(tx *proto.Genesis, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if info.blockID != tv.genesis {
		return txDiff{}, errors.New("genesis transaction inside of non-genesis block")
	}
	if !info.initialisation {
		return txDiff{}, errors.New("genesis transaction in non-initialisation mode")
	}
	key := wavesBalanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(key.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: receiverBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	return diff, nil
}

func (tv *transactionValidator) createDiffPayment(tx *proto.Payment, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return txDiff{}, errors.Wrap(err, "invalid timestamp")
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderKey := wavesBalanceKey{address: senderAddr}
	senderBalanceDiff := -int64(tx.Amount) - int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	receiverKey := wavesBalanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: receiverBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := tv.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (tv *transactionValidator) checkAsset(asset *proto.OptionalAsset, initialisation bool) error {
	if !asset.Present {
		// Waves always valid.
		return nil
	}
	if _, err := tv.stor.assets.newestAssetRecord(asset.ID, !initialisation); err != nil {
		return errors.New("unknown asset")
	}
	return nil
}

func (tv *transactionValidator) createDiffTransfer(tx *proto.Transfer, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return txDiff{}, errors.Wrap(err, "invalid timestamp")
	}
	if err := tv.checkAsset(&tx.AmountAsset, info.initialisation); err != nil {
		return txDiff{}, err
	}
	if err := tv.checkAsset(&tx.FeeAsset, info.initialisation); err != nil {
		return txDiff{}, err
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := byteKey(senderAddr, tx.FeeAsset.ToID())
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey, tv.curBlockBalanceDiff(balanceDiff{balance: senderFeeBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	senderAmountKey := byteKey(senderAddr, tx.AmountAsset.ToID())
	senderAmountBalanceDiff := -int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderAmountKey, tv.curBlockBalanceDiff(balanceDiff{balance: senderAmountBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	recipientAddr := &proto.Address{}
	if tx.Recipient.Address == nil {
		recipientAddr, err = tv.stor.aliases.newestAddrByAlias(tx.Recipient.Alias.Alias, !info.initialisation)
		if err != nil {
			return txDiff{}, errors.Errorf("invalid alias: %v\n", err)
		}
	} else {
		recipientAddr = tx.Recipient.Address
	}
	receiverKey := byteKey(*recipientAddr, tx.AmountAsset.ToID())
	receiverBalanceDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey, tv.curBlockBalanceDiff(balanceDiff{balance: receiverBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := tv.minerPayout(diff, tx.Fee, info, tx.FeeAsset.ToID()); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (tv *transactionValidator) createDiffIssue(tx *proto.Issue, id []byte, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return txDiff{}, errors.Wrap(err, "invalid timestamp")
	}
	assetID, err := crypto.NewDigestFromBytes(id)
	if err != nil {
		return txDiff{}, err
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
		if err := tv.stor.assets.issueAsset(assetID, asset); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to issue asset")
		}
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderFeeBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: assetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderAssetBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := tv.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (tv *transactionValidator) createDiffReissue(tx *proto.Reissue, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return txDiff{}, errors.Wrap(err, "invalid timestamp")
	}
	// Check if it's "legal" to modify given asset.
	record, err := tv.stor.assets.newestAssetRecord(tx.AssetID, !info.initialisation)
	if err != nil {
		return txDiff{}, err
	}
	if (info.currentTimestamp > tv.settings.InvalidReissueInSameBlockUntilTime) && !record.reissuable {
		return txDiff{}, errors.New("attempt to reissue asset which is not reissuable")
	}
	if info.perform {
		// Modify asset.
		change := &assetReissueChange{
			reissuable: tx.Reissuable,
			diff:       int64(tx.Quantity),
			blockID:    info.blockID,
		}
		if err := tv.stor.assets.reissueAsset(tx.AssetID, change, !info.initialisation); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to reissue asset")
		}
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderFeeBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderAssetBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := tv.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (tv *transactionValidator) createDiffBurn(tx *proto.Burn, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return txDiff{}, errors.Wrap(err, "invalid timestamp")
	}
	if info.perform {
		// Modify asset.
		change := &assetBurnChange{
			diff:    int64(tx.Amount),
			blockID: info.blockID,
		}
		if err := tv.stor.assets.burnAsset(tx.AssetID, change, !info.initialisation); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to burn asset")
		}
	}
	// Append sender diff.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderFeeBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	senderAssetKey := assetBalanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := -int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderAssetBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := tv.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (tv *transactionValidator) createDiffExchange(tx proto.Exchange, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if err := tv.checkTimestamps(tx.GetTimestamp(), info.currentTimestamp, info.parentTimestamp); err != nil {
		return txDiff{}, errors.Wrap(err, "invalid timestamp")
	}
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return txDiff{}, err
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return txDiff{}, err
	}
	// Check assets.
	if err := tv.checkAsset(&sellOrder.AssetPair.AmountAsset, info.initialisation); err != nil {
		return txDiff{}, err
	}
	if err := tv.checkAsset(&sellOrder.AssetPair.PriceAsset, info.initialisation); err != nil {
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
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, sellOrder.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	senderPriceKey := byteKey(senderAddr, sellOrder.AssetPair.PriceAsset.ToID())
	if err := diff.appendBalanceDiff(senderPriceKey, tv.curBlockBalanceDiff(balanceDiff{balance: priceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	senderAmountKey := byteKey(senderAddr, sellOrder.AssetPair.AmountAsset.ToID())
	if err := diff.appendBalanceDiff(senderAmountKey, tv.curBlockBalanceDiff(balanceDiff{balance: -amountDiff}, info)); err != nil {
		return txDiff{}, err
	}
	senderFeeKey := wavesBalanceKey{senderAddr}
	senderFeeDiff := -int64(tx.GetSellMatcherFee())
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderFeeDiff}, info)); err != nil {
		return txDiff{}, err
	}
	receiverAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, buyOrder.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	receiverPriceKey := byteKey(receiverAddr, sellOrder.AssetPair.PriceAsset.ToID())
	if err := diff.appendBalanceDiff(receiverPriceKey, tv.curBlockBalanceDiff(balanceDiff{balance: -priceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	receiverAmountKey := byteKey(receiverAddr, sellOrder.AssetPair.AmountAsset.ToID())
	if err := diff.appendBalanceDiff(receiverAmountKey, tv.curBlockBalanceDiff(balanceDiff{balance: amountDiff}, info)); err != nil {
		return txDiff{}, err
	}
	receiverFeeKey := wavesBalanceKey{receiverAddr}
	receiverFeeDiff := -int64(tx.GetBuyMatcherFee())
	if err := diff.appendBalanceDiff(receiverFeeKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: receiverFeeDiff}, info)); err != nil {
		return txDiff{}, err
	}
	// Update matcher.
	matcherAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, buyOrder.MatcherPK)
	if err != nil {
		return txDiff{}, err
	}
	matcherKey := wavesBalanceKey{matcherAddr}
	matcherFee, err := util.AddInt64(int64(tx.GetBuyMatcherFee()), int64(tx.GetSellMatcherFee()))
	if err != nil {
		return txDiff{}, err
	}
	matcherBalanceDiff := matcherFee - int64(tx.GetFee())
	if err := diff.appendBalanceDiff(matcherKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: matcherBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := tv.minerPayout(diff, tx.GetFee(), info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (tv *transactionValidator) createDiffLease(tx *proto.Lease, id *crypto.Digest, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return txDiff{}, errors.Wrap(err, "invalid timestamp")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	recipientAddr := &proto.Address{}
	if tx.Recipient.Address == nil {
		recipientAddr, err = tv.stor.aliases.newestAddrByAlias(tx.Recipient.Alias.Alias, !info.initialisation)
		if err != nil {
			return txDiff{}, errors.Errorf("invalid alias: %v\n", err)
		}
	} else {
		recipientAddr = tx.Recipient.Address
	}
	if senderAddr == *recipientAddr {
		return txDiff{}, errors.New("trying to lease money to self")
	}
	// Append sender diff.
	senderKey := wavesBalanceKey{address: senderAddr}
	senderLeaseOutDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(senderKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{leaseOut: senderLeaseOutDiff}, info)); err != nil {
		return txDiff{}, err
	}
	senderFeeDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderFeeDiff}, info)); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	receiverKey := wavesBalanceKey{address: *recipientAddr}
	receiverLeaseInDiff := int64(tx.Amount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{leaseIn: receiverLeaseInDiff}, info)); err != nil {
		return txDiff{}, err
	}
	if info.perform {
		// Add leasing to lease state.
		r := &leasingRecord{
			leasing{true, tx.Amount, *recipientAddr, senderAddr},
			info.blockID,
		}
		if err := tv.stor.leases.addLeasing(*id, r); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to add leasing")
		}
	}
	if info.hasMiner() {
		if err := tv.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (tv *transactionValidator) createDiffLeaseCancel(tx *proto.LeaseCancel, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return txDiff{}, errors.Wrap(err, "invalid timestamp")
	}
	l, err := tv.stor.leases.newestLeasingInfo(tx.LeaseID, !info.initialisation)
	if err != nil {
		return txDiff{}, errors.Wrap(err, "no leasing info found for this leaseID")
	}
	if !l.isActive && (info.currentTimestamp > tv.settings.AllowMultipleLeaseCancelUntilTime) {
		return txDiff{}, errors.New("can not cancel lease which has already been cancelled")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	if (l.sender != senderAddr) && (info.currentTimestamp > tv.settings.AllowMultipleLeaseCancelUntilTime) {
		return txDiff{}, errors.New("sender of LeaseCancel is not sender of corresponding Lease")
	}
	if info.perform {
		if err := tv.stor.leases.cancelLeasing(tx.LeaseID, info.blockID, !info.initialisation); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to cancel leasing")
		}
	}
	// Append sender diff.
	senderKey := wavesBalanceKey{address: senderAddr}
	senderLeaseOutDiff := -int64(l.leaseAmount)
	if err := diff.appendBalanceDiff(senderKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{leaseOut: senderLeaseOutDiff}, info)); err != nil {
		return txDiff{}, err
	}
	senderFeeDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderFeeDiff}, info)); err != nil {
		return txDiff{}, err
	}
	// Append receiver diff.
	receiverKey := wavesBalanceKey{address: l.recipient}
	receiverLeaseInDiff := -int64(l.leaseAmount)
	if err := diff.appendBalanceDiff(receiverKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{leaseIn: receiverLeaseInDiff}, info)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := tv.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (tv *transactionValidator) createDiffCreateAlias(tx *proto.CreateAlias, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	if err := tv.checkTimestamps(tx.Timestamp, info.currentTimestamp, info.parentTimestamp); err != nil {
		return txDiff{}, errors.Wrap(err, "invalid timestamp")
	}
	// Check if alias already taken.
	if _, err := tv.stor.aliases.newestAddrByAlias(tx.Alias.Alias, !info.initialisation); err == nil {
		return txDiff{}, errors.New("alias is already taken")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return txDiff{}, err
	}
	if info.perform {
		// Save alias to aliases storage.
		r := &aliasRecord{
			addr:    senderAddr,
			blockID: info.blockID,
		}
		if err := tv.stor.aliases.createAlias(tx.Alias.Alias, r); err != nil {
			return txDiff{}, err
		}
	}
	// Append sender diff.
	senderFeeKey := wavesBalanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if err := diff.appendBalanceDiff(senderFeeKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: senderFeeBalanceDiff}, info)); err != nil {
		return txDiff{}, err
	}
	if info.hasMiner() {
		if err := tv.minerPayout(diff, tx.Fee, info, nil); err != nil {
			return txDiff{}, errors.Wrap(err, "failed to append miner payout")
		}
	}
	return diff, nil
}

func (tv *transactionValidator) createTxDiff(tx proto.Transaction, info *txValidationInfo) (txDiff, error) {
	switch v := tx.(type) {
	case *proto.Genesis:
		diff, err := tv.createDiffGenesis(v, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "genesis initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.Payment:
		diff, err := tv.createDiffPayment(v, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "payment initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.TransferV1:
		diff, err := tv.createDiffTransfer(&v.Transfer, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "transferv1 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.TransferV2:
		diff, err := tv.createDiffTransfer(&v.Transfer, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "transferv2 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.IssueV1:
		diff, err := tv.createDiffIssue(&v.Issue, tx.GetID(), info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "issuev1 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.IssueV2:
		diff, err := tv.createDiffIssue(&v.Issue, tx.GetID(), info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "issuev2 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.ReissueV1:
		diff, err := tv.createDiffReissue(&v.Reissue, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "reissuev1 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.ReissueV2:
		diff, err := tv.createDiffReissue(&v.Reissue, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "reissuev2 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.BurnV1:
		diff, err := tv.createDiffBurn(&v.Burn, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "burnv1 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.BurnV2:
		diff, err := tv.createDiffBurn(&v.Burn, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "burnv2 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.ExchangeV1:
		diff, err := tv.createDiffExchange(v, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "exchangev1 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.ExchangeV2:
		diff, err := tv.createDiffExchange(v, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "exchange2 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.LeaseV1:
		diff, err := tv.createDiffLease(&v.Lease, v.ID, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "leasev1 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.LeaseV2:
		diff, err := tv.createDiffLease(&v.Lease, v.ID, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "leasev2 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.LeaseCancelV1:
		diff, err := tv.createDiffLeaseCancel(&v.LeaseCancel, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "leasecancelv1 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.LeaseCancelV2:
		diff, err := tv.createDiffLeaseCancel(&v.LeaseCancel, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "leasecancelv2 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.CreateAliasV1:
		diff, err := tv.createDiffCreateAlias(&v.CreateAlias, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "createaliasv1 initial checking/diff creation failed")
		}
		return diff, nil
	case *proto.CreateAliasV2:
		diff, err := tv.createDiffCreateAlias(&v.CreateAlias, info)
		if err != nil {
			return txDiff{}, errors.Wrap(err, "createaliasv2 initial checking/diff creation failed")
		}
		return diff, nil
	default:
		return txDiff{}, errors.Errorf("transaction type %T is not supported\n", v)
	}
}

func (tv *transactionValidator) txDiffFromFees(addr proto.Address, distr *feeDistribution, info *txValidationInfo) (txDiff, error) {
	diff := newTxDiff()
	wavesKey := wavesBalanceKey{addr}
	wavesDiff := distr.totalWavesFees - distr.currentWavesBlockFees
	if err := diff.appendBalanceDiff(wavesKey.bytes(), tv.curBlockBalanceDiff(balanceDiff{balance: int64(wavesDiff)}, info)); err != nil {
		return txDiff{}, err
	}
	for asset, totalFee := range distr.totalFees {
		curFee, ok := distr.currentBlockFees[asset]
		if !ok {
			return txDiff{}, errors.New("current fee for asset is not found")
		}
		assetKey := byteKey(addr, asset[:])
		assetDiff := totalFee - curFee
		if err := diff.appendBalanceDiff(assetKey, tv.curBlockBalanceDiff(balanceDiff{balance: int64(assetDiff)}, info)); err != nil {
			return txDiff{}, err
		}
	}
	return diff, nil
}

func (tv *transactionValidator) createPrevBlockMinerFeeDiff(info *txValidationInfo) (txDiff, error) {
	feeDistr, err := tv.prevBlockFeeDistr(info.prevBlockID)
	if err != nil {
		return txDiff{}, err
	}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, info.minerPK)
	if err != nil {
		return txDiff{}, err
	}
	diff, err := tv.txDiffFromFees(minerAddr, feeDistr, info)
	if err != nil {
		return txDiff{}, err
	}
	return diff, nil
}

func (tv *transactionValidator) createTransactionsDiffs(transactions []proto.Transaction, info *txValidationInfo) ([]txDiff, error) {
	diffs := make([]txDiff, len(transactions))
	for i, tx := range transactions {
		diff, err := tv.createTxDiff(tx, info)
		if err != nil {
			return nil, err
		}
		diffs[i] = diff
	}
	return diffs, nil
}

type blockDiff struct {
	minerDiff txDiff
	txDiffs   []txDiff
}

func (tv *transactionValidator) createBlockDiff(blockTxs []proto.Transaction, info *txValidationInfo) (blockDiff, error) {
	var diff blockDiff
	if info.hasPrevBlock() {
		minerDiff, err := tv.createPrevBlockMinerFeeDiff(info)
		if err != nil {
			return blockDiff{}, err
		}
		diff.minerDiff = minerDiff
	}
	txDiffs, err := tv.createTransactionsDiffs(blockTxs, info)
	if err != nil {
		return blockDiff{}, err
	}
	diff.txDiffs = txDiffs
	// Save fee distribution.
	tv.blockFees[info.blockID] = tv.curDistr
	// Reset current block fee distribution.
	tv.curDistr = newFeeDistribution()
	return diff, nil
}

func (tv *transactionValidator) saveTransactionsDiffs(diffs []txDiff, validate bool) error {
	for _, diff := range diffs {
		if err := tv.changesStor.saveTxDiff(diff, validate); err != nil {
			return err
		}
	}
	return nil
}

func (tv *transactionValidator) saveBlockDiff(diff blockDiff, validate bool) error {
	if err := tv.changesStor.saveTxDiff(diff.minerDiff, validate); err != nil {
		return err
	}
	if err := tv.saveTransactionsDiffs(diff.txDiffs, validate); err != nil {
		return err
	}
	return nil
}

func (tv *transactionValidator) validateBalanceChanges(initialisation, perform bool) error {
	if err := tv.changesStor.validateBalancesChanges(!initialisation, perform); err != nil {
		return err
	}
	for blockID, distr := range tv.blockFees {
		if err := tv.stor.blocksInfo.saveFeeDistribution(blockID, &distr); err != nil {
			return err
		}
	}
	return nil
}

func (tv *transactionValidator) reset() {
	tv.changesStor.reset()
	tv.curDistr = newFeeDistribution()
	tv.blockFees = make(map[crypto.Signature]feeDistribution)
}
