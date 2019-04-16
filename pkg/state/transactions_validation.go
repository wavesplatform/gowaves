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
			if newBalance < 0 {
				return errors.New("validation failed: negative balance")
			}
			if err := bs.balances.setAccountBalance(delta.key, uint64(newBalance), change.blockID); err != nil {
				return errors.Errorf("failed to set account balance: %v\n", err)
			}
		}
	}
	bs.reset()
	return nil
}

func (bs *changesStorage) reset() {
	bs.deltas = nil
	bs.lastIndex = 0
	bs.wavesKeys = make(map[wavesBalanceKey]int)
	bs.assetKeys = make(map[assetBalanceKey]int)

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
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.GetFee())
	if ok, err := tv.addChanges(senderFeeKey.bytes(), senderFeeBalanceDiff, block); !ok {
		return false, err
	}
	senderAssetKey := balanceKey{address: senderAddr, asset: assetID[:]}
	senderAssetBalanceDiff := int64(tx.GetQuantity())
	if ok, err := tv.addChanges(senderAssetKey.bytes(), senderAssetBalanceDiff, block); !ok {
		return false, err
	}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr}
	minerBalanceDiff := int64(tx.GetFee())
	if ok, err := tv.addChanges(minerKey.bytes(), minerBalanceDiff, block); !ok {
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
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if ok, err := tv.addChanges(senderFeeKey.bytes(), senderFeeBalanceDiff, block); !ok {
		return false, err
	}
	senderAssetKey := balanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := int64(tx.Quantity)
	if ok, err := tv.addChanges(senderAssetKey.bytes(), senderAssetBalanceDiff, block); !ok {
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
	// Update sender.
	senderAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr}
	senderFeeBalanceDiff := -int64(tx.Fee)
	if ok, err := tv.addChanges(senderFeeKey.bytes(), senderFeeBalanceDiff, block); !ok {
		return false, err
	}
	senderAssetKey := balanceKey{address: senderAddr, asset: tx.AssetID[:]}
	senderAssetBalanceDiff := -int64(tx.Amount)
	if ok, err := tv.addChanges(senderAssetKey.bytes(), senderAssetBalanceDiff, block); !ok {
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
	senderPriceKey := balanceKey{address: senderAddr, asset: sellOrder.AssetPair.PriceAsset.ToID()}
	if ok, err := tv.addChanges(senderPriceKey.bytes(), priceDiff, block); !ok {
		return false, err
	}
	senderAmountKey := balanceKey{address: senderAddr, asset: sellOrder.AssetPair.AmountAsset.ToID()}
	if ok, err := tv.addChanges(senderAmountKey.bytes(), -amountDiff, block); !ok {
		return false, err
	}
	senderFeeKey := balanceKey{address: senderAddr}
	senderFeeDiff := -int64(tx.GetSellMatcherFee())
	if ok, err := tv.addChanges(senderFeeKey.bytes(), senderFeeDiff, block); !ok {
		return false, err
	}
	receiverAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, buyOrder.SenderPK)
	if err != nil {
		return false, err
	}
	receiverPriceKey := balanceKey{address: receiverAddr, asset: sellOrder.AssetPair.PriceAsset.ToID()}
	if ok, err := tv.addChanges(receiverPriceKey.bytes(), -priceDiff, block); !ok {
		return false, err
	}
	receiverAmountKey := balanceKey{address: receiverAddr, asset: sellOrder.AssetPair.AmountAsset.ToID()}
	if ok, err := tv.addChanges(receiverAmountKey.bytes(), amountDiff, block); !ok {
		return false, err
	}
	receiverFeeKey := balanceKey{address: receiverAddr}
	receiverFeeDiff := -int64(tx.GetBuyMatcherFee())
	if ok, err := tv.addChanges(receiverFeeKey.bytes(), receiverFeeDiff, block); !ok {
		return false, err
	}
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
	if ok, err := tv.addChanges(matcherKey.bytes(), matcherBalanceDiff, block); !ok {
		return false, err
	}
	// Update miner.
	minerAddr, err := proto.NewAddressFromPublicKey(tv.settings.AddressSchemeCharacter, block.GenPublicKey)
	if err != nil {
		return false, err
	}
	minerKey := balanceKey{address: minerAddr}
	minerBalanceDiff := int64(tx.GetFee())
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
	default:
		return errors.Errorf("transaction type %T is not supported\n", v)
	}
	return nil
}

func (tv *transactionValidator) performTransactions() error {
	return tv.balancesChanges.applyDeltas()
}

func (tv *transactionValidator) reset() {
	tv.balancesChanges.reset()
}
