package state

import (
	"bytes"
	"math"

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
	if bytes.Equal(prevBlock[:], ngActivationBlock[:]) {
		// If the last block in current state is the NG activation block,
		// miner does not get any fees from this (last) block, because it was all taken by the last non-NG miner.
		return &feeDistribution{}, nil
	}
	if bytes.Equal(prevBlock[:], d.prevBlockID[:]) {
		// We already have distribution for this block.
		return &d.prevDistr, nil
	}
	// Load from DB.
	return d.stor.blocksInfo.feeDistribution(prevBlock)
}

func (d *blockDiffer) appendBlockInfoToBalanceDiff(diff *balanceDiff, block *proto.BlockHeader) {
	allowLeasedTransfer := true
	if block.Timestamp >= d.settings.AllowLeasedBalanceTransferUntilTime {
		allowLeasedTransfer = false
	}
	diff.allowLeasedTransfer = allowLeasedTransfer
	diff.blockID = block.BlockSignature
}

func (d *blockDiffer) appendBlockInfoToTxDiff(diff txDiff, block *proto.BlockHeader) {
	for key := range diff {
		balanceDiff := diff[key]
		d.appendBlockInfoToBalanceDiff(&balanceDiff, block)
		diff[key] = balanceDiff
	}
}

func (d *blockDiffer) txDiffFromFees(addr proto.Address, distr *feeDistribution) (txDiff, error) {
	diff := newTxDiff()
	wavesKey := wavesBalanceKey{addr}
	wavesDiff := distr.totalWavesFees - distr.currentWavesBlockFees
	if wavesDiff != 0 {
		if err := diff.appendBalanceDiff(wavesKey.bytes(), balanceDiff{balance: int64(wavesDiff)}); err != nil {
			return txDiff{}, err
		}
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

func (d *blockDiffer) createPrevBlockMinerFeeDiff(prevBlockID crypto.Signature, minerPK crypto.PublicKey) (txDiff, proto.Address, error) {
	feeDistr, err := d.prevBlockFeeDistr(prevBlockID)
	if err != nil {
		return txDiff{}, proto.Address{}, err
	}
	minerAddr, err := proto.NewAddressFromPublicKey(d.settings.AddressSchemeCharacter, minerPK)
	if err != nil {
		return txDiff{}, proto.Address{}, err
	}
	diff, err := d.txDiffFromFees(minerAddr, feeDistr)
	if err != nil {
		return txDiff{}, minerAddr, err
	}
	return diff, minerAddr, nil
}

func (d *blockDiffer) createTransactionDiff(tx proto.Transaction, block *proto.BlockHeader, height uint64, initialisation bool) (txDiff, error) {
	blockInfo, err := proto.BlockInfoFromHeader(d.settings.AddressSchemeCharacter, block, height)
	if err != nil {
		return txDiff{}, err
	}
	differInfo := &differInfo{initialisation, blockInfo}
	diff, err := d.handler.createDiffTx(tx, differInfo)
	if err != nil {
		return txDiff{}, err
	}
	d.appendBlockInfoToTxDiff(diff, block)
	return diff, nil
}

func (d *blockDiffer) countMinerFee(tx proto.Transaction) error {
	if err := d.handler.minerFeeTx(tx, &d.curDistr); err != nil {
		return err
	}
	return nil
}

func (d *blockDiffer) saveCurFeeDistr(block *proto.BlockHeader) error {
	// Save fee distribution to DB.
	if err := d.stor.blocksInfo.saveFeeDistribution(block.BlockSignature, &d.curDistr); err != nil {
		return err
	}
	// Update fee distribution.
	d.prevDistr = d.curDistr
	d.prevBlockID = block.BlockSignature
	d.curDistr = newFeeDistribution()
	return nil
}

func (d *blockDiffer) createMinerDiff(block *proto.BlockHeader, hasParent bool, height uint64) (txDiff, error) {
	var err error
	var minerDiff txDiff
	var minerAddr proto.Address
	if hasParent {
		minerDiff, minerAddr, err = d.createPrevBlockMinerFeeDiff(block.Parent, block.GenPublicKey)
		if err != nil {
			return txDiff{}, err
		}
		d.appendBlockInfoToTxDiff(minerDiff, block)
	}
	err = d.addBlockReward(minerDiff, minerAddr, block, height)
	if err != nil {
		return txDiff{}, err
	}
	return minerDiff, nil
}

func (d *blockDiffer) addBlockReward(diff txDiff, addr proto.Address, block *proto.BlockHeader, height uint64) error {
	// We use isOneBlockBeforeActivation() here as workaround, because in existing blockchain
	// reward was charged at 000 block, but block v4 appeared one block after.
	oneBeforeActivation := d.stor.features.isOneBlockBeforeActivation(int16(settings.BlockReward), height)
	activated, err := d.stor.features.isActivated(int16(settings.BlockReward))
	if err != nil {
		return err
	}
	if !activated && !oneBeforeActivation {
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
	wavesKey := wavesBalanceKey{addr}
	err = diff.appendBalanceDiff(wavesKey.bytes(), balanceDiff{balance: int64(reward)})
	if err != nil {
		return err
	}
	d.appendBlockInfoToTxDiff(diff, block)
	return nil
}

func (d *blockDiffer) reset() {
	d.curDistr = newFeeDistribution()
	d.prevDistr = newFeeDistribution()
	d.prevBlockID = crypto.Signature{}
}
