package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type invokeAddlInfo struct {
	// Number of scripts invoked *before* main function invocation.
	// This includes tx sender script and smart assets scripts from script payments.
	previousScriptRuns uint64
	initialisation     bool
	block              *proto.BlockHeader
	height             uint64

	// When validatingUtx flag is true, it means that we should validate balance diffs
	// before saving them to storage.
	validatingUtx bool
}

func (i *invokeAddlInfo) hasBlock() bool {
	return i.block != nil
}

type invokeApplier struct {
	state types.SmartState
	sc    *scriptCaller

	txHandler *transactionHandler

	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings

	blockDiffer    *blockDiffer
	invokeDiffStor *diffStorageWrapped
	diffApplier    *diffApplier
}

func newInvokeApplier(
	state types.SmartState,
	sc *scriptCaller,
	txHandler *transactionHandler,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
	blockDiffer *blockDiffer,
	diffStor *diffStorageWrapped,
	diffApplier *diffApplier,
) *invokeApplier {
	return &invokeApplier{
		state:          state,
		sc:             sc,
		txHandler:      txHandler,
		stor:           stor,
		settings:       settings,
		blockDiffer:    blockDiffer,
		invokeDiffStor: diffStor,
		diffApplier:    diffApplier,
	}
}

type payment struct {
	sender   proto.Address
	receiver proto.Address
	amount   uint64
	asset    proto.OptionalAsset
}

func (ia *invokeApplier) newPaymentFromScriptTransfer(scriptAddr proto.Address, tr proto.ScriptResultTransfer, info *invokeAddlInfo) (*payment, error) {
	receiver, err := recipientToAddress(tr.Recipient, ia.stor.aliases, !info.initialisation)
	if err != nil {
		return nil, errors.Wrap(err, "recipientToAddress() failed")
	}
	if tr.Amount < 0 {
		return nil, errors.New("transfer amount is < 0")
	}
	return &payment{
		sender:   scriptAddr,
		receiver: *receiver,
		amount:   uint64(tr.Amount),
		asset:    tr.Asset,
	}, nil
}

func (ia *invokeApplier) newTxDiffFromPayment(pmt *payment, updateMinIntermediateBalance bool, info *invokeAddlInfo) (txDiff, error) {
	diff := newTxDiff()
	senderKey := byteKey(pmt.sender, pmt.asset.ToID())
	senderBalanceDiff := -int64(pmt.amount)
	if err := diff.appendBalanceDiff(senderKey, newBalanceDiff(senderBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	receiverKey := byteKey(pmt.receiver, pmt.asset.ToID())
	receiverBalanceDiff := int64(pmt.amount)
	if err := diff.appendBalanceDiff(receiverKey, newBalanceDiff(receiverBalanceDiff, 0, 0, updateMinIntermediateBalance)); err != nil {
		return txDiff{}, err
	}
	if !info.validatingUtx {
		// This is needed because we save this diff to storage manually.
		ia.blockDiffer.appendBlockInfoToTxDiff(diff, info.block)
	}
	return diff, nil
}

func (ia *invokeApplier) newTxDiffFromScriptTransfer(scriptAddr proto.Address, tr proto.ScriptResultTransfer, info *invokeAddlInfo) (txDiff, error) {
	pmt, err := ia.newPaymentFromScriptTransfer(scriptAddr, tr, info)
	if err != nil {
		return txDiff{}, err
	}
	// updateMinIntermediateBalance is set to false here, because in Scala implementation
	// only fee and payments are checked for temporary negative balance.
	return ia.newTxDiffFromPayment(pmt, false, info)
}

func (ia *invokeApplier) saveIntermediateDiff(diff txDiff) error {
	return ia.invokeDiffStor.saveTxDiff(diff)
}

func (ia *invokeApplier) saveDiff(diff txDiff, info *invokeAddlInfo) error {
	if !info.validatingUtx {
		ia.blockDiffer.appendBlockInfoToTxDiff(diff, info.block)
		return ia.invokeDiffStor.diffStorage.saveTxDiff(diff)
	}
	// For UTX, we must validate changes before we save them.
	changes, err := ia.invokeDiffStor.diffStorage.changesByTxDiff(diff)
	if err != nil {
		return err
	}
	if err := ia.diffApplier.validateBalancesChanges(changes, true); err != nil {
		return err
	}
	if err := ia.invokeDiffStor.diffStorage.saveBalanceChanges(changes); err != nil {
		return err
	}
	return nil
}

func (ia *invokeApplier) createTxDiff(tx *proto.InvokeScriptV1, info *invokeAddlInfo) (txDiff, error) {
	if info.validatingUtx {
		return ia.txHandler.createDiffTx(tx, &differInfo{
			initialisation: false,
			blockInfo:      &proto.BlockInfo{Timestamp: info.block.Timestamp},
		})
	}
	return ia.blockDiffer.createTransactionDiff(tx, info.block, info.height, info.initialisation)
}

// For InvokeScript transactions there is no performer function.
// Instead, here (in applyInvokeScriptV1) we perform both balance and state changes
// along with fee validation which is normally done in checker function.
// This is due to InvokeScript specifics: WriteSet (state) changes have to be applied before
// TransferSet (balances) changes, and performer is always called *after* differ,
// since differ depends on state and should not normally take into account state changes from same
// transaction (InvokeScript is exception from this rule).
// Also, we can not check fee in checker because before function invocation, we don't have TransferSet
// and can not evaluate how many smart assets (= script runs) will be involved, while this directly
// affects minimum allowed fee.
// That is why invoke transaction is applied to state in a different way - here, unlike other
// transaction types.
func (ia *invokeApplier) applyInvokeScriptV1(tx *proto.InvokeScriptV1, info *invokeAddlInfo) error {
	// At first, clear invoke diff storage from any previus diffs.
	ia.invokeDiffStor.invokeDiffsStor.reset()
	if !info.validatingUtx && !info.hasBlock() {
		return errors.New("no block is provided and not validating UTX")
	}
	// Call script function.
	blockInfo, err := proto.BlockInfoFromHeader(ia.settings.AddressSchemeCharacter, info.block, info.height)
	if err != nil {
		return err
	}
	scriptAddr, err := recipientToAddress(tx.ScriptRecipient, ia.stor.aliases, !info.initialisation)
	if err != nil {
		return errors.Wrap(err, "recipientToAddress() failed")
	}
	scriptRes, err := ia.sc.invokeFunction(tx, blockInfo, info.initialisation)
	if err != nil {
		return errors.Wrap(err, "invokeFunction() failed")
	}
	// Check script result.
	if err := scriptRes.Valid(); err != nil {
		return errors.Wrap(err, "invalid script result")
	}
	// Perform fee and payment changes first.
	// Basic differ for InvokeScript creates only fee and payment diff.
	feeAndPaymentDiff, err := ia.createTxDiff(tx, info)
	if err != nil {
		return err
	}
	commonDiff := feeAndPaymentDiff
	if err := ia.saveIntermediateDiff(feeAndPaymentDiff); err != nil {
		return err
	}
	// Perform data storage writes.
	if !info.validatingUtx {
		// TODO: when UTX transactions are validated, there is no block,
		// and we can not perform state changes.
		for _, entry := range scriptRes.Writes {
			if err := ia.stor.accountsDataStor.appendEntry(*scriptAddr, entry, info.block.BlockSignature); err != nil {
				return err
			}
		}
	}
	// Perform transfers.
	scriptRuns := info.previousScriptRuns
	for _, transfer := range scriptRes.Transfers {
		assetExists := ia.stor.assets.newestAssetExists(transfer.Asset, !info.initialisation)
		if !assetExists {
			return errors.New("invalid asset in transfer")
		}
		isSmartAsset, err := ia.stor.scriptsStorage.newestIsSmartAsset(transfer.Asset.ID, !info.initialisation)
		if err != nil {
			return err
		}
		if isSmartAsset {
			fullTr, err := proto.NewFullScriptTransfer(ia.settings.AddressSchemeCharacter, &transfer, tx)
			if err != nil {
				return errors.Wrap(err, "failed to convert transfer to full script transfer")
			}
			// Call asset script if transferring smart asset.
			if err := ia.sc.callAssetScriptWithScriptTransfer(fullTr, transfer.Asset.ID, blockInfo, info.initialisation); err != nil {
				return errors.Wrap(err, "asset script failed on transfer set")
			}
			scriptRuns++
		}
		// Perform transfer.
		txDiff, err := ia.newTxDiffFromScriptTransfer(*scriptAddr, transfer, info)
		if err != nil {
			return err
		}
		// diff must be saved to storage, because further asset scripts must take
		// recent balance changes into account.
		if err := ia.saveIntermediateDiff(txDiff); err != nil {
			return err
		}
		// Append intermediate diff to common diff.
		for key, balanceDiff := range txDiff {
			if err := commonDiff.appendBalanceDiffStr(key, balanceDiff); err != nil {
				return err
			}
		}
	}
	// Remove diffs from invoke stor.
	ia.invokeDiffStor.invokeDiffsStor.reset()
	// Add these diffs as a common diff to main stor.
	if err := ia.saveDiff(commonDiff, info); err != nil {
		return err
	}
	// Check transaction fee.
	sponsorshipActivated, err := ia.stor.features.isActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return err
	}
	if !sponsorshipActivated {
		// Minimum fee is not checked before sponsorship activation.
		return nil
	}
	minWavesFee := scriptExtraFee*scriptRuns + feeConstants[proto.InvokeScriptTransaction]*FeeUnit
	wavesFee := tx.Fee
	if tx.FeeAsset.Present {
		wavesFee, err = ia.stor.sponsoredAssets.sponsoredAssetToWaves(tx.FeeAsset.ID, tx.Fee)
		if err != nil {
			return errors.Wrap(err, "failed to convert fee asset to waves")
		}
	}
	if wavesFee < minWavesFee {
		return errors.Errorf("tx fee %d is less than minimum value of %d\n", wavesFee, minWavesFee)
	}
	return nil
}
