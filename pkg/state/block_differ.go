package state

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
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
	prevBlockID crypto.Signature

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

func (d *blockDiffer) prevBlockFeeDistr(prevBlock crypto.Signature) (*feeDistribution, error) {
	ngActivated, err := d.stor.features.isActivated(int16(settings.NG))
	if err != nil {
		return nil, err
	}
	if !ngActivated {
		// If NG is not activated, miner does not get any fees from the previous block.
		return &feeDistribution{}, nil
	}
	ngActivationBlock, err := d.stor.features.activationBlock(int16(settings.NG))
	if err != nil {
		return nil, err
	}
	if bytes.Compare(prevBlock[:], ngActivationBlock[:]) == 0 {
		// If the last block in current state is the NG activation block,
		// miner does not get any fees from this (last) block, because it was all taken by the last non-NG miner.
		return &feeDistribution{}, nil
	}
	if bytes.Compare(prevBlock[:], d.prevBlockID[:]) == 0 {
		// We already have distribution for this block.
		return &d.prevDistr, nil
	}
	// Load from DB.
	return d.stor.blocksInfo.feeDistribution(prevBlock)
}

func (d *blockDiffer) appendBlockInfoToTxDiff(diff txDiff, block *proto.Block) {
	for key := range diff {
		balanceDiff := diff[key]
		allowLeasedTransfer := true
		if block.Timestamp >= d.settings.AllowLeasedBalanceTransferUntilTime {
			allowLeasedTransfer = false
		}
		balanceDiff.allowLeasedTransfer = allowLeasedTransfer
		balanceDiff.blockID = block.BlockSignature
		diff[key] = balanceDiff
	}
}

func (d *blockDiffer) txDiffFromFees(addr proto.Address, distr *feeDistribution) (txDiff, error) {
	diff := newTxDiff()
	wavesKey := wavesBalanceKey{addr}
	wavesDiff := distr.totalWavesFees - distr.currentWavesBlockFees
	if err := diff.appendBalanceDiff(wavesKey.bytes(), balanceDiff{balance: int64(wavesDiff)}); err != nil {
		return txDiff{}, err
	}
	for asset, totalFee := range distr.totalFees {
		curFee, ok := distr.currentBlockFees[asset]
		if !ok {
			return txDiff{}, errors.New("current fee for asset is not found")
		}
		assetKey := byteKey(addr, asset[:])
		assetDiff := totalFee - curFee
		if err := diff.appendBalanceDiff(assetKey, balanceDiff{balance: int64(assetDiff)}); err != nil {
			return txDiff{}, err
		}
	}
	return diff, nil
}

func (d *blockDiffer) createPrevBlockMinerFeeDiff(prevBlockID crypto.Signature, minerPK crypto.PublicKey) (txDiff, error) {
	feeDistr, err := d.prevBlockFeeDistr(prevBlockID)
	if err != nil {
		return txDiff{}, err
	}
	minerAddr, err := proto.NewAddressFromPublicKey(d.settings.AddressSchemeCharacter, minerPK)
	if err != nil {
		return txDiff{}, err
	}
	diff, err := d.txDiffFromFees(minerAddr, feeDistr)
	if err != nil {
		return txDiff{}, err
	}
	return diff, nil
}

func (d *blockDiffer) createTransactionsDiffs(transactions []proto.Transaction, block *proto.Block, initialisation bool) ([]txDiff, error) {
	d.curDistr = newFeeDistribution()
	diffs := make([]txDiff, len(transactions))
	for i, tx := range transactions {
		differInfo := &differInfo{initialisation, block.GenPublicKey, block.Timestamp}
		diff, err := d.handler.createDiffTx(tx, differInfo)
		if err != nil {
			return nil, err
		}
		diffs[i] = diff
		d.appendBlockInfoToTxDiff(diffs[i], block)
		ngActivated, err := d.stor.features.isActivated(int16(settings.NG))
		if err != nil {
			return nil, err
		}
		if err := d.handler.minerFeeTx(tx, &d.curDistr, ngActivated); err != nil {
			return nil, err
		}
	}
	d.prevDistr = d.curDistr
	d.prevBlockID = block.BlockSignature
	return diffs, nil
}

func (d *blockDiffer) createBlockDiff(blockTxs []proto.Transaction, block *proto.Block, initialisation, hasParent bool) (blockDiff, error) {
	var diff blockDiff
	if hasParent {
		minerDiff, err := d.createPrevBlockMinerFeeDiff(block.Parent, block.GenPublicKey)
		if err != nil {
			return blockDiff{}, err
		}
		diff.minerDiff = minerDiff
		d.appendBlockInfoToTxDiff(diff.minerDiff, block)
	}
	txDiffs, err := d.createTransactionsDiffs(blockTxs, block, initialisation)
	if err != nil {
		return blockDiff{}, err
	}
	diff.txDiffs = txDiffs
	return diff, nil
}

func (d *blockDiffer) reset() {
	d.curDistr = newFeeDistribution()
	d.prevDistr = newFeeDistribution()
}
