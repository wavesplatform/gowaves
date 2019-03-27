package state

import (
	"bytes"
	"sort"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	wavesBalanceKeySize = 1 + proto.AddressSize
	assetBalanceKeySize = 1 + proto.AddressSize + crypto.DigestSize
)

type change struct {
	diff    int64
	blockID crypto.Signature
}

type balanceChanges struct {
	// Key in main DB.
	key []byte
	// Cumulative diffs of blocks transactions.
	balanceDiffs []change
	// Diff which produces minimal balance value.
	// This is needed to check for negative balances.
	// For blocks when temporary negative balances are possible,
	// this value is set to the cumulative diff of all block's transactions.
	minBalanceDiff int64
}

func newBalanceChanges(key []byte) balanceChanges {
	return balanceChanges{key: key}
}

func (ch *balanceChanges) update(balanceDiff int64, blockID crypto.Signature, checkTempNegative bool) error {
	last := len(ch.balanceDiffs) - 1
	var lastID crypto.Signature
	prevDiff := int64(0)
	if last >= 0 {
		prevDiff = ch.balanceDiffs[last].diff
		lastID = ch.balanceDiffs[last].blockID
	}
	newDiff, err := util.AddInt64(prevDiff, balanceDiff)
	if err != nil {
		return errors.Errorf("failed to add balances: %v\n", err)
	}
	newChange := change{blockID: blockID, diff: newDiff}
	if blockID != lastID {
		ch.balanceDiffs = append(ch.balanceDiffs, newChange)
		// Check cumulative diff for previous block.
		if prevDiff < ch.minBalanceDiff {
			ch.minBalanceDiff = prevDiff
		}
	} else {
		ch.balanceDiffs[last] = newChange
	}
	if checkTempNegative {
		// Check every tx.
		if newDiff < ch.minBalanceDiff {
			ch.minBalanceDiff = newDiff
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
	lastIndex int
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
			bs.wavesKeys[wavesKey] = bs.lastIndex
			bs.deltas = append(bs.deltas, newBalanceChanges(key))
			bs.lastIndex++
		}
		return &bs.deltas[bs.wavesKeys[wavesKey]], nil
	} else if size == assetBalanceKeySize {
		var assetKey assetBalanceKey
		copy(assetKey[:], key)
		if _, ok := bs.assetKeys[assetKey]; !ok {
			bs.assetKeys[assetKey] = bs.lastIndex
			bs.deltas = append(bs.deltas, newBalanceChanges(key))
			bs.lastIndex++
		}
		return &bs.deltas[bs.assetKeys[assetKey]], nil
	}
	return nil, errors.New("invalid key size")
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
		balance, err := bs.balances.accountBalance(delta.key)
		if err != nil {
			return errors.Errorf("failed to retrieve account balance: %v\n", err)
		}
		// Check for negative balance.
		minBalance, err := util.AddInt64(int64(balance), delta.minBalanceDiff)
		if err != nil {
			return errors.Errorf("failed to add balances: %v\n", err)
		}
		if minBalance < 0 {
			return errors.New("validation failed: negative balance")
		}
		for _, change := range delta.balanceDiffs {
			newBalance, err := util.AddInt64(int64(balance), change.diff)
			if err != nil {
				return errors.Errorf("failed to add balances: %v\n", err)
			}
			if err := bs.balances.setAccountBalance(delta.key, uint64(newBalance), change.blockID); err != nil {
				return errors.Errorf("failed to set account balance: %v\n", err)
			}
		}
	}
	// Reset (free memory).
	bs.deltas = nil
	bs.wavesKeys = make(map[wavesBalanceKey]int)
	bs.assetKeys = make(map[assetBalanceKey]int)
	return nil
}

type transactionValidator struct {
	genesis         crypto.Signature
	balancesChanges *changesStorage
	assets          *assets
	settings        *settings.BlockchainSettings
}

func newTransactionValidator(
	genesis crypto.Signature,
	balances *balances,
	assets *assets,
	settings *settings.BlockchainSettings,
) (*transactionValidator, error) {
	balancesChanges, err := newChangesStorage(balances)
	if err != nil {
		return nil, errors.Errorf("failed to create balances changes storage: %v\n", err)
	}
	return &transactionValidator{
		genesis:         genesis,
		balancesChanges: balancesChanges,
		assets:          assets,
		settings:        settings,
	}, nil
}

func (tv *transactionValidator) isSupported(tx proto.Transaction) bool {
	switch v := tx.(type) {
	case *proto.Genesis:
		return true
	case *proto.Payment:
		return true
	case *proto.TransferV1:
		if v.FeeAsset.Present || v.AmountAsset.Present {
			// Only Waves for now.
			return false
		}
		if v.Recipient.Address == nil {
			// Aliases without specified address are not supported yet.
			return false
		}
		return true
	case *proto.TransferV2:
		if v.FeeAsset.Present || v.AmountAsset.Present {
			// Only Waves for now.
			return false
		}
		if v.Recipient.Address == nil {
			// Aliases without specified address are not supported yet.
			return false
		}
		return true
	default:
		// Other types of transactions are not supported.
		return false
	}
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

func (tv *transactionValidator) addChanges(key []byte, diff int64, block *proto.Block) (bool, error) {
	changes, err := tv.balancesChanges.balanceChanges(key)
	if err != nil {
		return false, errors.Wrap(err, "can not retrieve balance changes")
	}
	checkTempNegative := tv.checkNegativeBalance(block.Timestamp)
	if err := changes.update(diff, block.BlockSignature, checkTempNegative); err != nil {
		return false, errors.Wrap(err, "can not update balance changes")
	}
	return true, nil
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
	if ok, err := tv.addChanges(key.bytes(), receiverBalanceDiff, block); !ok {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validatePayment(tx *proto.Payment, block, parent *proto.Block, initialisation bool) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, block.Timestamp, parent.Timestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderKey := balanceKey{address: senderAddr}
	senderBalanceDiff := -int64(tx.Amount) - int64(tx.Fee)
	if ok, err := tv.addChanges(senderKey.bytes(), senderBalanceDiff, block); !ok {
		return false, err
	}
	// Update receiver.
	receiverKey := balanceKey{address: tx.Recipient}
	receiverBalanceDiff := int64(tx.Amount)
	if ok, err := tv.addChanges(receiverKey.bytes(), receiverBalanceDiff, block); !ok {
		return false, err
	}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr}
	minerBalanceDiff := int64(tx.Fee)
	if ok, err := tv.addChanges(minerKey.bytes(), minerBalanceDiff, block); !ok {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateTransferV1(tx *proto.TransferV1, block, parent *proto.Block, initialisation bool) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, block.Timestamp, parent.Timestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr, asset: tx.FeeAsset.ToID()}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if ok, err := tv.addChanges(senderFeeKey.bytes(), senderFeeBalanceDiff, block); !ok {
		return false, err
	}
	senderAmountKey := balanceKey{address: senderAddr, asset: tx.AmountAsset.ToID()}
	senderAmountBalanceDiff := -int64(tx.Amount)
	if ok, err := tv.addChanges(senderAmountKey.bytes(), senderAmountBalanceDiff, block); !ok {
		return false, err
	}
	// Update receiver.
	if tx.Recipient.Address == nil {
		// TODO implement
		return false, errors.New("alias without address is not supported yet")
	}
	receiverKey := balanceKey{address: *tx.Recipient.Address, asset: tx.AmountAsset.ToID()}
	receiverBalanceDiff := int64(tx.Amount)
	if ok, err := tv.addChanges(receiverKey.bytes(), receiverBalanceDiff, block); !ok {
		return false, err
	}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr, asset: tx.FeeAsset.ToID()}
	minerBalanceDiff := int64(tx.Fee)
	if ok, err := tv.addChanges(minerKey.bytes(), minerBalanceDiff, block); !ok {
		return false, err
	}
	return true, nil
}

func (tv *transactionValidator) validateTransferV2(tx *proto.TransferV2, block, parent *proto.Block, initialisation bool) (bool, error) {
	if ok, err := tv.checkTimestamps(tx.Timestamp, block.Timestamp, parent.Timestamp); !ok {
		return false, errors.Wrap(err, "invalid timestamp")
	}
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr, asset: tx.FeeAsset.ToID()}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if ok, err := tv.addChanges(senderFeeKey.bytes(), senderFeeBalanceDiff, block); !ok {
		return false, err
	}
	senderAmountKey := balanceKey{address: senderAddr, asset: tx.AmountAsset.ToID()}
	senderAmountBalanceDiff := -int64(tx.Amount)
	if ok, err := tv.addChanges(senderAmountKey.bytes(), senderAmountBalanceDiff, block); !ok {
		return false, err
	}
	// Update receiver.
	if tx.Recipient.Address == nil {
		// TODO implement
		return false, errors.New("alias without address is not supported yet")
	}
	receiverKey := balanceKey{address: *tx.Recipient.Address, asset: tx.AmountAsset.ToID()}
	receiverBalanceDiff := int64(tx.Amount)
	if ok, err := tv.addChanges(receiverKey.bytes(), receiverBalanceDiff, block); !ok {
		return false, err
	}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr, asset: tx.FeeAsset.ToID()}
	minerBalanceDiff := int64(tx.Fee)
	if ok, err := tv.addChanges(minerKey.bytes(), minerBalanceDiff, block); !ok {
		return false, err
	}
	return true, nil
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
		if ok, err := tv.validateTransferV1(v, block, parent, initialisation); !ok {
			return errors.Wrap(err, "transferv1 validation failed")
		}
	case *proto.TransferV2:
		if ok, err := tv.validateTransferV2(v, block, parent, initialisation); !ok {
			return errors.Wrap(err, "transferv2 validation failed")
		}
	default:
		return errors.Errorf("transaction type %T is not supported\n", v)
	}
	return nil
}

func (tv *transactionValidator) performTransactions() error {
	return tv.balancesChanges.applyDeltas()
}
