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

func (ia *invokeApplier) createTxDiff(tx *proto.InvokeScriptWithProofs, info *invokeAddlInfo, status bool) (txBalanceChanges, error) {
	if info.validatingUtx {
		return ia.txHandler.createDiffTx(tx, &differInfo{
			initialisation: false,
			blockInfo:      &proto.BlockInfo{Timestamp: info.block.Timestamp},
		})
	}
	return ia.blockDiffer.createTransactionDiff(tx, info.block, info.height, info.hitSource, info.initialisation, status)
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

type invokeApplicationInfo struct {
	totalScriptsInvoked uint64
	addresses           []proto.Address
	status              bool
}

func (ia *invokeApplier) countActionScriptRuns(actions []proto.ScriptAction, initialisation bool) (uint64, error) {
	scriptRuns := uint64(0)
	for _, action := range actions {
		var assetID crypto.Digest
		switch a := action.(type) {
		case *proto.TransferScriptAction:
			assetID = a.Asset.ID
		case *proto.ReissueScriptAction:
			assetID = a.AssetID
		case *proto.BurnScriptAction:
			assetID = a.AssetID
		default:
			continue
		}
		isSmartAsset, err := ia.stor.scriptsStorage.newestIsSmartAsset(assetID, initialisation)
		if err != nil {
			return 0, err
		}
		if isSmartAsset {
			scriptRuns++
		}
	}
	return scriptRuns, nil
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
func (ia *invokeApplier) applyInvokeScriptWithProofs(tx *proto.InvokeScriptWithProofs, info *invokeAddlInfo, acceptFailed bool) (*invokeApplicationInfo, error) {
	// At first, clear invoke diff storage from any previous diffs.
	ia.invokeDiffStor.invokeDiffsStor.reset()
	if !info.validatingUtx && !info.hasBlock() {
		return nil, errors.New("no block is provided and not validating UTX")
	}
	// Call script function.
	blockInfo, err := proto.BlockInfoFromHeader(ia.settings.AddressSchemeCharacter, info.block, info.height, info.hitSource)
	if err != nil {
		return nil, err
	}
	scriptAddr, err := recipientToAddress(tx.ScriptRecipient, ia.stor.aliases, !info.initialisation)
	if err != nil {
		return nil, errors.Wrap(err, "recipientToAddress() failed")
	}
	script, err := ia.stor.scriptsStorage.newestScriptByAddr(*scriptAddr, !info.initialisation)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate script on address '%s'", scriptAddr.String())
	}
	scriptPK, err := ia.stor.scriptsStorage.newestScriptPKByAddr(*scriptAddr, !info.initialisation)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get script's public key on address '%s'", scriptAddr.String())
	}
	// Check that the script's library supports multiple payments.
	// We don't have to check feature activation because we done it before.
	if len(tx.Payments) == 2 && script.Version < 4 {
		return nil, errors.Errorf("multiple payments is not allowed for RIDE library version %d", script.Version)
	}
	// Refuse payments to DApp itself since activation of BlockV5 (acceptFailed) and for DApps with StdLib V4.
	disableSelfTransfers := acceptFailed && script.Version >= 4
	if disableSelfTransfers && len(tx.Payments) > 0 {
		sender, err := proto.NewAddressFromPublicKey(ia.settings.AddressSchemeCharacter, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply script invocation")
		}
		if sender == *scriptAddr {
			return nil, errors.New("paying to DApp itself is forbidden since RIDE V4")
		}
	}
	invokeSucceed := true
	scriptActions, err := ia.sc.invokeFunction(script, tx, blockInfo, *scriptAddr, info.initialisation)
	if err != nil {
		if acceptFailed {
			invokeSucceed = false
		} else {
			return nil, errors.Wrap(err, "invokeFunction() failed")
		}
	}
	// Resolve all aliases.
	// It has to be done before validation because we validate addresses, not aliases.
	if err := ia.resolveAliases(scriptActions, info.initialisation); err != nil {
		return nil, errors.New("ScriptResult; failed to resolve aliases")
	}
	// Check script result.
	restrictions := proto.ActionsValidationRestrictions{DisableSelfTransfers: disableSelfTransfers, ScriptAddress: *scriptAddr}
	if err := proto.ValidateActions(scriptActions, restrictions); err != nil {
		return nil, errors.Wrap(err, "invalid script result")
	}
	if ia.buildApiData {
		// Save invoke result for extended API.
		// TODO: add saving of failure status to script result.
		res, err := proto.NewScriptResult(scriptActions, proto.ScriptErrorMessage{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to save script result")
		}
		if err := ia.stor.invokeResults.saveResult(*tx.ID, res, info.block.BlockID()); err != nil {
			return nil, errors.Wrap(err, "failed to save script result")
		}
	}
	// Perform fee and payment changes first.
	// Basic differ for InvokeScript creates only fee and payment diff.
	totalChanges, err := ia.createTxDiff(tx, info, invokeSucceed)
	if err != nil {
		return nil, err
	}
	commonDiff := totalChanges.diff
	issuedAssetsCount := uint64(0)
	actionScriptRuns, err := ia.countActionScriptRuns(scriptActions, info.initialisation)
	if err != nil {
		return nil, err
	}
	scriptRuns := info.previousScriptRuns + actionScriptRuns
	if invokeSucceed {
		if err := ia.saveIntermediateDiff(commonDiff); err != nil {
			return nil, err
		}
		for _, action := range scriptActions {
			switch a := action.(type) {
			case *proto.DataEntryScriptAction:
				// Perform data storage writes.
				if !info.validatingUtx {
					// TODO: when UTX transactions are validated, there is no block,
					// and we can not perform state changes.
					if err := ia.stor.accountsDataStor.appendEntryUncertain(*scriptAddr, a.Entry); err != nil {
						return nil, err
					}
				}

			case *proto.TransferScriptAction:
				// Perform transfers.
				addr := a.Recipient.Address
				totalChanges.appendAddr(*addr)
				assetExists := ia.stor.assets.newestAssetExists(a.Asset, !info.initialisation)
				if !assetExists {
					return nil, errors.New("invalid asset in transfer")
				}
				isSmartAsset, err := ia.stor.scriptsStorage.newestIsSmartAsset(a.Asset.ID, !info.initialisation)
				if err != nil {
					return nil, err
				}
				if isSmartAsset {
					fullTr, err := proto.NewFullScriptTransfer(a, tx)
					if err != nil {
						return nil, errors.Wrap(err, "failed to convert transfer to full script transfer")
					}
					// Call asset script if transferring smart asset.
					ok, err := ia.sc.callAssetScriptWithScriptTransfer(fullTr, a.Asset.ID, blockInfo, info.initialisation, acceptFailed)
					if err != nil {
						return nil, errors.Wrap(err, "asset script failed on transfer set")
					}
					if !ok {
						invokeSucceed = false
						break
					}
				}
				txDiff, err := ia.newTxDiffFromScriptTransfer(scriptAddr, a, info)
				if err != nil {
					return nil, err
				}
				// diff must be saved to storage, because further asset scripts must take
				// recent balance changes into account.
				if err := ia.saveIntermediateDiff(txDiff); err != nil {
					return nil, err
				}
				// Append intermediate diff to common diff.
				for key, balanceDiff := range txDiff {
					if err := commonDiff.appendBalanceDiffStr(key, balanceDiff); err != nil {
						return nil, err
					}
				}

			case *proto.IssueScriptAction:
				// Create asset's info.
				assetInfo := &assetInfo{
					assetConstInfo: assetConstInfo{
						issuer:   scriptPK,
						decimals: int8(a.Decimals),
					},
					assetChangeableInfo: assetChangeableInfo{
						quantity:    *big.NewInt(a.Quantity),
						name:        a.Name,
						description: a.Description,
						reissuable:  a.Reissuable,
					},
				}
				assetParams := assetParams{a.Quantity, a.Decimals, a.Reissuable}
				nft, err := isNFT(ia.stor.features, assetParams)
				if err != nil {
					return nil, err
				}
				if !nft {
					issuedAssetsCount += 1
				}
				if !info.validatingUtx {
					if err := ia.stor.assets.issueAssetUncertain(a.ID, assetInfo); err != nil {
						return nil, err
					}
					// Currently asset script is always empty.
					// TODO: if this script is ever set, don't forget to
					// also save complexity for it using saveComplexityForAsset().
					if err := ia.stor.scriptsStorage.setAssetScriptUncertain(a.ID, proto.Script{}, scriptPK); err != nil {
						return nil, err
					}
				}

				txDiff, err := ia.newTxDiffFromScriptIssue(scriptAddr, a)
				if err != nil {
					return nil, err
				}
				// diff must be saved to storage, because further asset scripts must take
				// recent balance changes into account.
				if err := ia.saveIntermediateDiff(txDiff); err != nil {
					return nil, err
				}
				// Append intermediate diff to common diff.
				for key, balanceDiff := range txDiff {
					if err := commonDiff.appendBalanceDiffStr(key, balanceDiff); err != nil {
						return nil, err
					}
				}

			case *proto.ReissueScriptAction:
				// Check validity of reissue.
				assetInfo, err := ia.stor.assets.newestAssetInfo(a.AssetID, !info.initialisation)
				if err != nil {
					return nil, err
				}
				if assetInfo.issuer != scriptPK {
					return nil, errors.New("asset was issued by other address")
				}
				if !assetInfo.reissuable {
					return nil, errors.New("attempt to reissue asset which is not reissuable")
				}
				if math.MaxInt64-a.Quantity < assetInfo.quantity.Int64() && info.block.Timestamp >= ia.settings.ReissueBugWindowTimeEnd {
					return nil, errors.New("asset total value overflow")
				}
				ok, err := ia.validateActionSmartAsset(a.AssetID, a, scriptPK, blockInfo, *tx.ID, tx.Timestamp, info.initialisation, acceptFailed)
				if err != nil {
					return nil, err
				}
				if !ok {
					invokeSucceed = false
					break
				}
				// Update asset's info.
				if !info.validatingUtx {
					change := &assetReissueChange{
						reissuable: a.Reissuable,
						diff:       a.Quantity,
					}
					if err := ia.stor.assets.reissueAssetUncertain(a.AssetID, change, !info.initialisation); err != nil {
						return nil, err
					}
				}
				txDiff, err := ia.newTxDiffFromScriptReissue(scriptAddr, a)
				if err != nil {
					return nil, err
				}
				// diff must be saved to storage, because further asset scripts must take
				// recent balance changes into account.
				if err := ia.saveIntermediateDiff(txDiff); err != nil {
					return nil, err
				}
				// Append intermediate diff to common diff.
				for key, balanceDiff := range txDiff {
					if err := commonDiff.appendBalanceDiffStr(key, balanceDiff); err != nil {
						return nil, err
					}
				}
			case *proto.BurnScriptAction:
				// Check burn.
				assetInfo, err := ia.stor.assets.newestAssetInfo(a.AssetID, !info.initialisation)
				if err != nil {
					return nil, err
				}
				burnAnyTokensEnabled, err := ia.stor.features.isActivated(int16(settings.BurnAnyTokens))
				if err != nil {
					return nil, err
				}
				if !burnAnyTokensEnabled && assetInfo.issuer != scriptPK {
					return nil, errors.New("asset was issued by other address")
				}
				ok, err := ia.validateActionSmartAsset(a.AssetID, a, scriptPK, blockInfo, *tx.ID, tx.Timestamp, info.initialisation, acceptFailed)
				if err != nil {
					return nil, err
				}
				if !ok {
					invokeSucceed = false
					break
				}
				// Update asset's info
				// Modify asset.
				if !info.validatingUtx {
					change := &assetBurnChange{
						diff: int64(a.Quantity),
					}
					if err := ia.stor.assets.burnAssetUncertain(a.AssetID, change, !info.initialisation); err != nil {
						return nil, errors.Wrap(err, "failed to burn asset")
					}
				}
				txDiff, err := ia.newTxDiffFromScriptBurn(scriptAddr, a)
				if err != nil {
					return nil, err
				}
				// diff must be saved to storage, because further asset scripts must take
				// recent balance changes into account.
				if err := ia.saveIntermediateDiff(txDiff); err != nil {
					return nil, err
				}
				// Append intermediate diff to common diff.
				for key, balanceDiff := range txDiff {
					if err := commonDiff.appendBalanceDiffStr(key, balanceDiff); err != nil {
						return nil, err
					}
				}
			default:
				return nil, errors.Errorf("unsupported script action '%T'", a)
			}
		}
	}
	// Check full transaction fee (with actions).
	feeOk, err := ia.checkFullFee(tx, scriptRuns, issuedAssetsCount)
	if err != nil {
		return nil, err
	}
	if !feeOk {
		invokeSucceed = false
	}
	// Remove diffs from invoke stor.
	ia.invokeDiffStor.invokeDiffsStor.reset()
	// Add these diffs as a common diff to main stor.
	if !invokeSucceed {
		ia.stor.dropUncertain()
		accountsChanges, err := ia.createTxDiff(tx, info, invokeSucceed)
		if err != nil {
			return nil, err
		}
		if err := ia.saveDiff(accountsChanges.diff, info); err != nil {
			return nil, err
		}
	} else {
		if err := ia.stor.commitUncertain(info.block.BlockID()); err != nil {
			return nil, err
		}
		ia.stor.dropUncertain()
		if err := ia.saveDiff(commonDiff, info); err != nil {
			return nil, err
		}
	}
	// Total scripts invoked = scriptRuns + invocation itself.
	totalScriptsInvoked := scriptRuns + 1
	res := &invokeApplicationInfo{
		totalScriptsInvoked: totalScriptsInvoked,
		addresses:           totalChanges.addresses(),
		status:              invokeSucceed,
	}
	return res, nil
}

func (ia *invokeApplier) checkFullFee(tx *proto.InvokeScriptWithProofs, scriptRuns, issuedAssetsCount uint64) (bool, error) {
	sponsorshipActivated, err := ia.stor.features.isActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return false, err
	}
	if !sponsorshipActivated {
		// Minimum fee is not checked before sponsorship activation.
		return true, nil
	}
	minIssueFee := feeConstants[proto.IssueTransaction] * FeeUnit * issuedAssetsCount
	minWavesFee := scriptExtraFee*scriptRuns + feeConstants[proto.InvokeScriptTransaction]*FeeUnit + minIssueFee
	wavesFee := tx.Fee
	if tx.FeeAsset.Present {
		wavesFee, err = ia.stor.sponsoredAssets.sponsoredAssetToWaves(tx.FeeAsset.ID, tx.Fee)
		if err != nil {
			return false, errors.Wrap(err, "failed to convert fee asset to waves")
		}
	}
	if wavesFee < minWavesFee {
		return false, nil
	}
	return true, nil
}

func (ia *invokeApplier) validateActionSmartAsset(asset crypto.Digest, action proto.ScriptAction, callerPK crypto.PublicKey,
	blockInfo *proto.BlockInfo, txID crypto.Digest, txTimestamp uint64, initialisation, acceptFailed bool) (bool, error) {
	isSmartAsset, err := ia.stor.scriptsStorage.newestIsSmartAsset(asset, !initialisation)
	if err != nil {
		return false, err
	}
	if isSmartAsset {
		obj, err := ast.NewVariablesFromScriptAction(ia.settings.AddressSchemeCharacter, action, callerPK, txID, txTimestamp)
		if err != nil {
			return false, err
		}
		ok, err := ia.sc.callAssetScriptCommon(obj, asset, blockInfo, initialisation, acceptFailed)
		if err != nil {
			return false, err
		}
		return ok, nil
	}
	return true, nil
}
