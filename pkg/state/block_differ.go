package state

import (
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

func (d *blockDiffer) appendBlockInfoToBalanceDiff(diff *balanceDiff, block *proto.BlockHeader) {
	allowLeasedTransfer := true
	if block.Timestamp >= d.settings.AllowLeasedBalanceTransferUntilTime {
		allowLeasedTransfer = false
	}
	diff.allowLeasedTransfer = allowLeasedTransfer
	diff.blockID = block.BlockID()
}

func (d *blockDiffer) appendBlockInfoToTxDiff(diff txDiff, block *proto.BlockHeader) {
	for key := range diff {
		balanceDiff := diff[key]
		d.appendBlockInfoToBalanceDiff(&balanceDiff, block)
		diff[key] = balanceDiff
	}
}

func (d *blockDiffer) txDiffFromFees(addr proto.AddressID, distr *feeDistribution) (txDiff, error) {
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
		assetKey := byteKey(addr, *proto.NewOptionalAssetFromDigest(asset))
		assetDiff := totalFee - curFee
		if err := diff.appendBalanceDiff(assetKey, balanceDiff{balance: int64(assetDiff)}); err != nil {
			return txDiff{}, err
		}
	}
	return diff, nil
}

func (d *blockDiffer) createPrevBlockMinerFeeDiff(prevBlockID proto.BlockID, minerPK crypto.PublicKey) (txDiff, proto.WavesAddress, error) {
	feeDistr, err := d.prevBlockFeeDistr(prevBlockID)
	if err != nil {
		return txDiff{}, proto.WavesAddress{}, err
	}
	minerAddr, err := proto.NewAddressFromPublicKey(d.settings.AddressSchemeCharacter, minerPK)
	if err != nil {
		return txDiff{}, proto.WavesAddress{}, err
	}
	diff, err := d.txDiffFromFees(minerAddr.ID(), feeDistr)
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

func (d *blockDiffer) createMinerDiff(block *proto.BlockHeader, hasParent bool) (txDiff, error) {
	var err error
	var minerDiff txDiff
	var minerAddr proto.WavesAddress
	if hasParent {
		minerDiff, minerAddr, err = d.createPrevBlockMinerFeeDiff(block.Parent, block.GeneratorPublicKey)
		if err != nil {
			return txDiff{}, err
		}
		d.appendBlockInfoToTxDiff(minerDiff, block)
	}
	err = d.addBlockReward(minerDiff, minerAddr.ID(), block)
	if err != nil {
		return txDiff{}, err
	}
	return minerDiff, nil
}

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
	d.prevBlockID = proto.BlockID{}
}
