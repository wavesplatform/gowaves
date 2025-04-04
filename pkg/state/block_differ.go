package state

import (
	"math"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state/internal"
)

type blockDiff struct {
	minerDiff txDiff
	txDiffs   []txDiff
}

type blockDiffer struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings

	curDistr    feeDistribution
	prevDistr   feeDistribution
	prevBlockID proto.BlockID

	handler *transactionHandler
}

func newBlockDiffer(handler *transactionHandler, stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*blockDiffer, error) {
	return &blockDiffer{
		stor:      stor,
		settings:  settings,
		curDistr:  newFeeDistribution(),
		prevDistr: newFeeDistribution(),
		handler:   handler,
	}, nil
}

// prevBlockFeeDistr returns fee distribution for the previous block.
// If the previous block is the same as the previous block in the blockDiffer,
// it returns the distribution from the blockDiffer.
// This method does not modify the blockDiffer itself and the state.
func (d *blockDiffer) prevBlockFeeDistr(prevBlock proto.BlockID) (*feeDistribution, error) {
	ngActivated, err := d.stor.features.newestIsActivatedForNBlocks(int16(settings.NG), 2)
	if err != nil {
		return nil, err
	}
	if !ngActivated {
		// If NG is not activated, miner does not get any fees from the previous block.
		// If the last block in current state is the NG activation block,
		// miner does not get any fees from this (last) block, because it was all taken by the last non-NG miner.
		return &feeDistribution{}, nil
	}
	if prevBlock == d.prevBlockID {
		// We already have distribution for this block.
		return &d.prevDistr, nil
	}
	// Load from DB.
	return d.stor.blocksInfo.feeDistribution(prevBlock)
}

// appendBlockInfoToBalanceDiff appends block ID and allowLeasedTransfer flag to balanceDiff.
// This method does not modify blockDiffer itself, but modifies balanceDiff.
func (d *blockDiffer) appendBlockInfoToBalanceDiff(diff *balanceDiff, block *proto.BlockHeader) {
	allowLeasedTransfer := block.Timestamp < d.settings.AllowLeasedBalanceTransferUntilTime
	diff.allowLeasedTransfer = allowLeasedTransfer
	diff.blockID = block.BlockID()
}

// appendBlockInfoToTxDiff appends block ID and allowLeasedTransfer flag to all balanceDiffs in txDiff.
// This method does not modify blockDiffer itself, but modifies txDiff.
func (d *blockDiffer) appendBlockInfoToTxDiff(diff txDiff, block *proto.BlockHeader) {
	for key := range diff {
		balanceDiff := diff[key]
		d.appendBlockInfoToBalanceDiff(&balanceDiff, block)
		diff[key] = balanceDiff
	}
}

func txDiffFromFees(addr proto.AddressID, distr *feeDistribution) (txDiff, error) {
	diff := newTxDiff()
	wavesKey := wavesBalanceKey{addr}
	wavesDiff := distr.totalWavesFees - distr.currentWavesBlockFees
	if wavesDiff != 0 {
		err := diff.appendBalanceDiff(wavesKey.bytes(), balanceDiff{balance: internal.NewIntChange(int64(wavesDiff))})
		if err != nil {
			return txDiff{}, err
		}
	}
	for asset, totalFee := range distr.totalFees {
		curFee, ok := distr.currentBlockFees[asset]
		if !ok {
			return txDiff{}, errors.New("current fee for asset is not found")
		}
		assetKey := byteKey(addr, *proto.NewOptionalAssetFromDigest(asset))
		assetDiff := totalFee - curFee
		err := diff.appendBalanceDiff(assetKey, balanceDiff{balance: internal.NewIntChange(int64(assetDiff))})
		if err != nil {
			return txDiff{}, err
		}
	}
	return diff, nil
}

// createPrevBlockMinerFeeDiff creates txDiff for the miner of the previous block.
// This method does not modify the state.
func (d *blockDiffer) createPrevBlockMinerFeeDiff(prevBlockID proto.BlockID, minerPK crypto.PublicKey) (txDiff, proto.WavesAddress, error) {
	feeDistr, err := d.prevBlockFeeDistr(prevBlockID)
	if err != nil {
		return txDiff{}, proto.WavesAddress{}, err
	}
	minerAddr, err := proto.NewAddressFromPublicKey(d.settings.AddressSchemeCharacter, minerPK)
	if err != nil {
		return txDiff{}, proto.WavesAddress{}, err
	}
	diff, err := txDiffFromFees(minerAddr.ID(), feeDistr)
	if err != nil {
		return txDiff{}, minerAddr, err
	}
	return diff, minerAddr, nil
}

func (d *blockDiffer) createFailedTransactionDiff(tx proto.Transaction, block *proto.BlockHeader, differInfo *differInfo) (txBalanceChanges, error) {
	var txChanges txBalanceChanges
	var err error
	switch tx.GetTypeInfo().Type {
	case proto.InvokeScriptTransaction:
		txChanges, err = d.handler.td.createFeeDiffInvokeScriptWithProofs(tx, differInfo)
		if err != nil {
			return txBalanceChanges{}, err
		}
	case proto.ExchangeTransaction:
		txChanges, err = d.handler.td.createFeeDiffExchange(tx, differInfo)
		if err != nil {
			return txBalanceChanges{}, err
		}
	case proto.EthereumMetamaskTransaction:
		txChanges, err = d.handler.td.createFeeDiffEthereumInvokeScriptWithProofs(tx, differInfo)
		if err != nil {
			return txBalanceChanges{}, err
		}
	case proto.InvokeExpressionTransaction:
		txChanges, err = d.handler.td.createFeeDiffInvokeExpressionWithProofs(tx, differInfo)
		if err != nil {
			return txBalanceChanges{}, err
		}
	default:
		return txBalanceChanges{}, errors.New("only Exchange and Invoke transactions may fail")
	}
	d.appendBlockInfoToTxDiff(txChanges.diff, block)
	return txChanges, nil
}

func (d *blockDiffer) createTransactionDiff(tx proto.Transaction, block *proto.BlockHeader, differInfo *differInfo) (txBalanceChanges, error) {
	txChanges, err := d.handler.createDiffTx(tx, differInfo)
	if err != nil {
		return txBalanceChanges{}, err
	}
	d.appendBlockInfoToTxDiff(txChanges.diff, block)
	return txChanges, nil
}

func (d *blockDiffer) countMinerFee(tx proto.Transaction) error {
	if err := d.handler.minerFeeTx(tx, &d.curDistr); err != nil {
		return err
	}
	return nil
}

// doMinerPayoutBeforeNG calculates miner payout for the block before NG activation.
// This method does not modify the state.
// All changes are appended to the passed txDiff.
func (d *blockDiffer) doMinerPayoutBeforeNG(
	diff txDiff,
	blockTimestamp uint64,
	minerAddr proto.WavesAddress,
	transactions []proto.Transaction,
) error {
	ngActivated, fErr := d.stor.features.newestIsActivatedForNBlocks(int16(settings.NG), 1)
	if fErr != nil {
		return fErr
	}
	if ngActivated { // no-op after NG activation
		return nil
	}
	updateMinIntermediateBalance := blockTimestamp >= d.settings.CheckTempNegativeAfterTime
	for i, tx := range transactions {
		var (
			fee      = tx.GetFee()
			feeAsset = tx.GetFeeAsset()
		)
		minerKey := byteKey(minerAddr.ID(), feeAsset)
		minerBalanceDiff := calculateCurrentBlockTxFee(fee, ngActivated)
		nd := newMinerFeeForcedBalanceDiff(int64(minerBalanceDiff), updateMinIntermediateBalance)
		if err := diff.appendBalanceDiff(minerKey, nd); err != nil {
			return errors.Wrapf(err, "failed to append balance diff for miner on %d-th transaction", i+1)
		}
	}
	return nil
}

func (d *blockDiffer) saveCurFeeDistr(block *proto.BlockHeader) error {
	// Save fee distribution to DB.
	if err := d.stor.blocksInfo.saveFeeDistribution(block.BlockID(), &d.curDistr); err != nil {
		return err
	}
	// Update fee distribution.
	d.prevDistr = d.curDistr
	d.prevBlockID = block.BlockID()
	d.curDistr = newFeeDistribution()
	return nil
}

// createMinerAndRewardDiff creates txDiff for the miner of the block and adds block reward.
// This method MUST NOT modify the state.
func (d *blockDiffer) createMinerAndRewardDiff(
	blockHeader *proto.BlockHeader,
	hasParent bool,
	transactions []proto.Transaction,
) (txDiff, error) {
	var err error
	var minerDiff txDiff
	var minerAddr proto.WavesAddress
	if hasParent {
		minerDiff, minerAddr, err = d.createPrevBlockMinerFeeDiff(blockHeader.Parent, blockHeader.GeneratorPublicKey)
		if err != nil {
			return txDiff{}, err
		}
		if mpErr := d.doMinerPayoutBeforeNG(minerDiff, blockHeader.Timestamp, minerAddr, transactions); mpErr != nil {
			return txDiff{}, errors.Wrap(mpErr, "failed to count miner payout")
		}
		d.appendBlockInfoToTxDiff(minerDiff, blockHeader)
	}
	err = d.addBlockReward(minerDiff, minerAddr.ID(), blockHeader)
	if err != nil {
		return txDiff{}, err
	}
	return minerDiff, nil
}

// addBlockReward adds block reward to the miner's balance.
// This method does not modify the state.
// All changes are applied to the passed txDiff.
func (d *blockDiffer) addBlockReward(diff txDiff, addr proto.AddressID, block *proto.BlockHeader) error {
	activated, err := d.stor.features.newestIsActivated(int16(settings.BlockReward))
	if err != nil {
		return err
	}
	if !activated {
		// Monetary policy is not working yet.
		return nil
	}
	reward, err := d.stor.monetaryPolicy.reward()
	if err != nil {
		return err
	}
	if reward > math.MaxInt64 {
		return errors.New("reward overflows int64")
	}
	c := newRewardsCalculator(d.settings, d.stor.features)
	if err := c.applyToDiff(diff, addr, d.stor.hs.stateDB.rw.addingBlockHeight(), reward); err != nil {
		return err
	}
	d.appendBlockInfoToTxDiff(diff, block)
	return nil
}

func (d *blockDiffer) reset() {
	d.curDistr = newFeeDistribution()
	d.prevDistr = newFeeDistribution()
	d.prevBlockID = proto.BlockID{}
}
