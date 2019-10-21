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
	// validatingUtx bool
}

func (i *invokeAddlInfo) hasBlock() bool {
	return i.block != nil
}

type invokeApplier struct {
	state types.SmartState
	sc    *scriptCaller

	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings

	blockDiffer *blockDiffer
	diffStor    *diffStorage
}

func newInvokeApplier(
	state types.SmartState,
	sc *scriptCaller,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
	blockDiffer *blockDiffer,
	diffStor *diffStorage,
) *invokeApplier {
	return &invokeApplier{
		state:       state,
		sc:          sc,
		stor:        stor,
		blockDiffer: blockDiffer,
		diffStor:    diffStor,
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
	if tr.Amount <= 0 {
		return nil, errors.New("transfer amount is <= 0")
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
	// This is needed because we save this diff to storage manually.
	ia.blockDiffer.appendBlockInfoToTxDiff(diff, info.block)
	return diff, nil
}

func (ia *invokeApplier) applyPayment(pmt *payment, updateMinIntermediateBalance bool, info *invokeAddlInfo) error {
	diff, err := ia.newTxDiffFromPayment(pmt, updateMinIntermediateBalance, info)
	if err != nil {
		return err
	}
	// diff must be saved to storage, because further asset scripts must take
	// recent balance changes into account.
	if err := ia.diffStor.saveTxDiff(diff); err != nil {
		return err
	}
	return nil
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
	// Perform fee and payment changes first.
	// Basic differ for InvokeScript creates only fee and payment diff.
	feeAndPaymentDiff, err := ia.blockDiffer.createTransactionDiff(tx, info.block, info.height, info.initialisation)
	if err != nil {
		return err
	}
	if err := ia.diffStor.saveTxDiff(feeAndPaymentDiff); err != nil {
		return err
	}
	// Now call script.
	blockInfo, err := proto.BlockInfoFromHeader(ia.settings.AddressSchemeCharacter, info.block, info.height)
	if err != nil {
		return err
	}
	scriptRes, err := ia.sc.invokeFunction(tx, blockInfo, info.initialisation)
	if err != nil {
		return errors.Wrap(err, "invokeFunction() failed")
	}
	// Check script result.
	if err := scriptRes.Valid(); err != nil {
		return errors.Wrap(err, "invalid script result")
	}
	// Perform data storage writes.
	scriptAddr, err := recipientToAddress(tx.ScriptRecipient, ia.stor.aliases, !info.initialisation)
	if err != nil {
		return errors.Wrap(err, "recipientToAddress() failed")
	}
	if info.hasBlock() {
		// TODO: when UTX transactions are validated, there is no block,
		// and we can not perform state changes.
		for _, entry := range scriptRes.Writes {
			if err := ia.stor.accountsDataStor.appendEntry(*scriptAddr, entry, info.block.BlockSignature); err != nil {
				return err
			}
		}
	}
	// updateMinIntermediateBalance is set to false here, because in Scala implementation
	// only fee and payments are checked for temporary negative balance.
	updateMinIntermediateBalance := false
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
			// Call asset script if transferring smart asset.
			if err := ia.sc.callAssetScript(tx, transfer.Asset.ID, blockInfo, info.initialisation); err != nil {
				return errors.Wrap(err, "asset script failed on transfer set")
			}
			scriptRuns++
		}
		// Perform transfer.
		pmt, err := ia.newPaymentFromScriptTransfer(*scriptAddr, transfer, info)
		if err != nil {
			return err
		}
		if err := ia.applyPayment(pmt, updateMinIntermediateBalance, info); err != nil {
			return errors.Wrap(err, "failed to apply script transfer")
		}
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
