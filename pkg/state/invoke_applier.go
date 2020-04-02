package state

import (
	"math"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
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
	hitSource          []byte

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

	buildApiData bool
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
	buildApiData bool,
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
		buildApiData:   buildApiData,
	}
}

type payment struct {
	sender   proto.Address
	receiver proto.Address
	amount   uint64
	asset    proto.OptionalAsset
}

func (ia *invokeApplier) newPaymentFromTransferScriptAction(scriptAddr *proto.Address, action *proto.TransferScriptAction) (*payment, error) {
	if action.Recipient.Address == nil {
		return nil, errors.New("transfer has unresolved aliases")
	}
	if action.Amount < 0 {
		return nil, errors.New("negative transfer amount")
	}
	return &payment{
		sender:   *scriptAddr,
		receiver: *action.Recipient.Address,
		amount:   uint64(action.Amount),
		asset:    action.Asset,
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

func (ia *invokeApplier) newTxDiffFromScriptTransfer(scriptAddr *proto.Address, action *proto.TransferScriptAction, info *invokeAddlInfo) (txDiff, error) {
	pmt, err := ia.newPaymentFromTransferScriptAction(scriptAddr, action)
	if err != nil {
		return txDiff{}, err
	}
	// updateMinIntermediateBalance is set to false here, because in Scala implementation
	// only fee and payments are checked for temporary negative balance.
	return ia.newTxDiffFromPayment(pmt, false, info)
}

func (ia *invokeApplier) newTxDiffFromScriptIssue(scriptAddr *proto.Address, action *proto.IssueScriptAction) (txDiff, error) {
	diff := newTxDiff()
	senderAssetKey := assetBalanceKey{address: *scriptAddr, asset: action.ID[:]}
	senderAssetBalanceDiff := int64(action.Quantity)
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
		return nil, err
	}
	return diff, nil
}

func (ia *invokeApplier) newTxDiffFromScriptReissue(scriptAddr *proto.Address, action *proto.ReissueScriptAction) (txDiff, error) {
	diff := newTxDiff()
	senderAssetKey := assetBalanceKey{address: *scriptAddr, asset: action.AssetID[:]}
	senderAssetBalanceDiff := action.Quantity
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
		return nil, err
	}
	return diff, nil
}

func (ia *invokeApplier) newTxDiffFromScriptBurn(scriptAddr *proto.Address, action *proto.BurnScriptAction) (txDiff, error) {
	diff := newTxDiff()
	senderAssetKey := assetBalanceKey{address: *scriptAddr, asset: action.AssetID[:]}
	senderAssetBalanceDiff := -action.Quantity
	if err := diff.appendBalanceDiff(senderAssetKey.bytes(), newBalanceDiff(senderAssetBalanceDiff, 0, 0, false)); err != nil {
		return nil, err
	}
	return diff, nil
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

func (ia *invokeApplier) createTxDiff(tx *proto.InvokeScriptWithProofs, info *invokeAddlInfo) (txBalanceChanges, error) {
	if info.validatingUtx {
		return ia.txHandler.createDiffTx(tx, &differInfo{
			initialisation: false,
			blockInfo:      &proto.BlockInfo{Timestamp: info.block.Timestamp},
		})
	}
	return ia.blockDiffer.createTransactionDiff(tx, info.block, info.height, info.hitSource, info.initialisation)
}

func (ia *invokeApplier) resolveAliases(actions []proto.ScriptAction, initialisation bool) error {
	for i, a := range actions {
		tr, ok := a.(proto.TransferScriptAction)
		if !ok {
			continue
		}
		addr, err := recipientToAddress(tr.Recipient, ia.stor.aliases, !initialisation)
		if err != nil {
			return err
		}
		tr.Recipient = proto.NewRecipientFromAddress(*addr)
		actions[i] = tr
	}
	return nil
}

// For InvokeScript transactions there is no performer function.
// Instead, here (in applyInvokeScriptWithProofs) we perform both balance and state changes
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
func (ia *invokeApplier) applyInvokeScriptWithProofs(tx *proto.InvokeScriptWithProofs, info *invokeAddlInfo) (txBalanceChanges, error) {
	// At first, clear invoke diff storage from any previous diffs.
	ia.invokeDiffStor.invokeDiffsStor.reset()
	if !info.validatingUtx && !info.hasBlock() {
		return txBalanceChanges{}, errors.New("no block is provided and not validating UTX")
	}
	// Call script function.
	blockInfo, err := proto.BlockInfoFromHeader(ia.settings.AddressSchemeCharacter, info.block, info.height, info.hitSource)
	if err != nil {
		return txBalanceChanges{}, err
	}
	scriptAddr, err := recipientToAddress(tx.ScriptRecipient, ia.stor.aliases, !info.initialisation)
	if err != nil {
		return txBalanceChanges{}, errors.Wrap(err, "recipientToAddress() failed")
	}
	multiPaymentActivated, err := ia.stor.features.isActivated(int16(settings.MultiPaymentInvokeScript))
	if err != nil {
		return txBalanceChanges{}, errors.Wrap(err, "failed to apply script invocation")
	}
	script, err := ia.stor.scriptsStorage.newestScriptByAddr(*scriptAddr, !info.initialisation)
	if err != nil {
		return txBalanceChanges{}, errors.Wrapf(err, "failed to instantiate script on address '%s'", scriptAddr.String())
	}
	scriptPK, err := ia.stor.scriptsStorage.newestScriptPKByAddr(*scriptAddr, !info.initialisation)
	if err != nil {
		return txBalanceChanges{}, errors.Wrapf(err, "failed to get script's public key on address '%s'", scriptAddr.String())
	}
	// Check that the script's library supports multiple payments.
	// We don't have to check feature activation because we done it before.
	if len(tx.Payments) == 2 && script.Version < 4 {
		return txBalanceChanges{}, errors.Errorf("multiple payments is not allowed for RIDE library version %d", script.Version)
	}
	// Refuse payments to DApp itself since activation of MultiPaymentInvokeScript (RIDE V4) and for DApps with StdLib V4
	disableSelfTransfers := multiPaymentActivated && script.Version >= 4
	if disableSelfTransfers && len(tx.Payments) > 0 {
		sender, err := proto.NewAddressFromPublicKey(ia.settings.AddressSchemeCharacter, tx.SenderPK)
		if err != nil {
			return txBalanceChanges{}, errors.Wrapf(err, "failed to apply script invocation")
		}
		if sender == *scriptAddr {
			return txBalanceChanges{}, errors.New("paying to DApp itself is forbidden since RIDE V4")
		}
	}
	scriptActions, err := ia.sc.invokeFunction(script, tx, blockInfo, *scriptAddr, info.initialisation)
	if err != nil {
		return txBalanceChanges{}, errors.Wrap(err, "invokeFunction() failed")
	}
	// Resolve all aliases in .
	// It have to be done before validation because we validate addresses, not aliases.
	if err := ia.resolveAliases(scriptActions, info.initialisation); err != nil {
		return txBalanceChanges{}, errors.New("ScriptResult; failed to resolve aliases")
	}
	// Check script result
	restrictions := proto.ActionsValidationRestrictions{DisableSelfTransfers: disableSelfTransfers, ScriptAddress: *scriptAddr}
	if err := proto.ValidateActions(scriptActions, restrictions); err != nil {
		return txBalanceChanges{}, errors.Wrap(err, "invalid script result")
	}
	if ia.buildApiData {
		// Save invoke result for extended API.
		res, err := proto.NewScriptResult(scriptActions)
		if err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to save script result")
		}
		if err := ia.stor.invokeResults.saveResult(*tx.ID, res, info.block.BlockID()); err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to save script result")
		}
	}
	// Perform fee and payment changes first.
	// Basic differ for InvokeScript creates only fee and payment diff.
	feeAndPaymentChanges, err := ia.createTxDiff(tx, info)
	if err != nil {
		return txBalanceChanges{}, err
	}
	totalChanges := feeAndPaymentChanges
	commonDiff := totalChanges.diff
	if err := ia.saveIntermediateDiff(commonDiff); err != nil {
		return txBalanceChanges{}, err
	}

	scriptRuns := info.previousScriptRuns
	for _, action := range scriptActions {
		switch a := action.(type) {
		case *proto.DataEntryScriptAction:
			// Perform data storage writes.
			if !info.validatingUtx {
				// TODO: when UTX transactions are validated, there is no block,
				// and we can not perform state changes.
				if err := ia.stor.accountsDataStor.appendEntry(*scriptAddr, a.Entry, info.block.BlockID()); err != nil {
					return txBalanceChanges{}, err
				}
			}

		case *proto.TransferScriptAction:
			// Perform transfers.
			addr := a.Recipient.Address
			totalChanges.appendAddr(*addr)
			assetExists := ia.stor.assets.newestAssetExists(a.Asset, !info.initialisation)
			if !assetExists {
				return txBalanceChanges{}, errors.New("invalid asset in transfer")
			}
			isSmartAsset, err := ia.stor.scriptsStorage.newestIsSmartAsset(a.Asset.ID, !info.initialisation)
			if err != nil {
				return txBalanceChanges{}, err
			}
			if isSmartAsset {
				fullTr, err := proto.NewFullScriptTransfer(a, tx)
				if err != nil {
					return txBalanceChanges{}, errors.Wrap(err, "failed to convert transfer to full script transfer")
				}
				// Call asset script if transferring smart asset.
				if err := ia.sc.callAssetScriptWithScriptTransfer(fullTr, a.Asset.ID, blockInfo, info.initialisation); err != nil {
					return txBalanceChanges{}, errors.Wrap(err, "asset script failed on transfer set")
				}
				scriptRuns++
			}
			// Perform transfer.
			txDiff, err := ia.newTxDiffFromScriptTransfer(scriptAddr, a, info)
			if err != nil {
				return txBalanceChanges{}, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return txBalanceChanges{}, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := commonDiff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return txBalanceChanges{}, err
				}
			}

		case *proto.IssueScriptAction:
			// Create asset's info
			assetInfo := &assetInfo{
				assetConstInfo: assetConstInfo{
					issuer:   scriptPK,
					decimals: int8(a.Decimals),
				},
				assetChangeableInfo: assetChangeableInfo{
					quantity:    *big.NewInt(int64(a.Quantity)),
					name:        a.Name,
					description: a.Description,
					reissuable:  a.Reissuable,
				},
			}
			if !info.validatingUtx {
				if err := ia.stor.assets.issueAsset(a.ID, assetInfo, info.block.ID); err != nil {
					return txBalanceChanges{}, err
				}
			}

			txDiff, err := ia.newTxDiffFromScriptIssue(scriptAddr, a)
			if err != nil {
				return txBalanceChanges{}, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return txBalanceChanges{}, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := commonDiff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return txBalanceChanges{}, err
				}
			}

		case *proto.ReissueScriptAction:
			// Check validity of reissue
			assetInfo, err := ia.stor.assets.newestAssetInfo(a.AssetID, !info.initialisation)
			if err != nil {
				return txBalanceChanges{}, err
			}
			if assetInfo.issuer != scriptPK {
				return txBalanceChanges{}, errors.New("asset was issued by other address")
			}
			if !assetInfo.reissuable {
				return txBalanceChanges{}, errors.New("attempt to reissue asset which is not reissuable")
			}
			if math.MaxInt64-a.Quantity < assetInfo.quantity.Int64() && info.block.Timestamp >= ia.settings.ReissueBugWindowTimeEnd {
				return txBalanceChanges{}, errors.New("asset total value overflow")
			}
			ok, err := ia.validateActionSmartAsset(a.AssetID, a, scriptPK, blockInfo, *tx.ID, tx.Timestamp, info.initialisation)
			if err != nil {
				return txBalanceChanges{}, err
			}
			if ok {
				scriptRuns++
			}
			// Update asset's info
			if !info.validatingUtx {
				change := &assetReissueChange{
					reissuable: a.Reissuable,
					diff:       a.Quantity,
				}
				if err := ia.stor.assets.reissueAsset(a.AssetID, change, info.block.ID, !info.initialisation); err != nil {
					return txBalanceChanges{}, err
				}
			}
			txDiff, err := ia.newTxDiffFromScriptReissue(scriptAddr, a)
			if err != nil {
				return txBalanceChanges{}, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return txBalanceChanges{}, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := commonDiff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return txBalanceChanges{}, err
				}
			}
		case *proto.BurnScriptAction:
			// Check burn
			assetInfo, err := ia.stor.assets.newestAssetInfo(a.AssetID, !info.initialisation)
			if err != nil {
				return txBalanceChanges{}, err
			}
			burnAnyTokensEnabled, err := ia.stor.features.isActivated(int16(settings.BurnAnyTokens))
			if err != nil {
				return txBalanceChanges{}, err
			}
			if !burnAnyTokensEnabled && assetInfo.issuer != scriptPK {
				return txBalanceChanges{}, errors.New("asset was issued by other address")
			}
			ok, err := ia.validateActionSmartAsset(a.AssetID, a, scriptPK, blockInfo, *tx.ID, tx.Timestamp, info.initialisation)
			if err != nil {
				return txBalanceChanges{}, err
			}
			if ok {
				scriptRuns++
			}
			// Update asset's info
			// Modify asset.
			if !info.validatingUtx {
				change := &assetBurnChange{
					diff: int64(a.Quantity),
				}
				if err := ia.stor.assets.burnAsset(a.AssetID, change, info.block.ID, !info.initialisation); err != nil {
					return txBalanceChanges{}, errors.Wrap(err, "failed to burn asset")
				}
			}
			txDiff, err := ia.newTxDiffFromScriptBurn(scriptAddr, a)
			if err != nil {
				return txBalanceChanges{}, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return txBalanceChanges{}, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := commonDiff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return txBalanceChanges{}, err
				}
			}
		default:
			return txBalanceChanges{}, errors.Errorf("unsupported script action '%T'", a)
		}
	}
	// Remove diffs from invoke stor.
	ia.invokeDiffStor.invokeDiffsStor.reset()
	// Add these diffs as a common diff to main stor.
	if err := ia.saveDiff(commonDiff, info); err != nil {
		return txBalanceChanges{}, err
	}
	// Check transaction fee.
	sponsorshipActivated, err := ia.stor.features.isActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return txBalanceChanges{}, err
	}
	if !sponsorshipActivated {
		// Minimum fee is not checked before sponsorship activation.
		return totalChanges, nil
	}
	minWavesFee := scriptExtraFee*scriptRuns + feeConstants[proto.InvokeScriptTransaction]*FeeUnit
	wavesFee := tx.Fee
	if tx.FeeAsset.Present {
		wavesFee, err = ia.stor.sponsoredAssets.sponsoredAssetToWaves(tx.FeeAsset.ID, tx.Fee)
		if err != nil {
			return txBalanceChanges{}, errors.Wrap(err, "failed to convert fee asset to waves")
		}
	}
	if wavesFee < minWavesFee {
		return txBalanceChanges{}, errors.Errorf("tx fee %d is less than minimum value of %d\n", wavesFee, minWavesFee)
	}
	return totalChanges, nil
}

func (ia *invokeApplier) validateActionSmartAsset(asset crypto.Digest, action proto.ScriptAction, callerPK crypto.PublicKey,
	blockInfo *proto.BlockInfo, txID crypto.Digest, txTimestamp uint64, initialisation bool) (bool, error) {
	isSmartAsset, err := ia.stor.scriptsStorage.newestIsSmartAsset(asset, !initialisation)
	if err != nil {
		return false, err
	}
	if isSmartAsset {
		obj, err := ast.NewVariablesFromScriptAction(ia.settings.AddressSchemeCharacter, action, callerPK, txID, txTimestamp)
		if err != nil {
			return false, err
		}
		if err := ia.sc.callAssetScriptCommon(obj, asset, blockInfo, initialisation); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
