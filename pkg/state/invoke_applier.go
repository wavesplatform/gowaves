package state

import (
	"fmt"
	"math"
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

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

func (ia *invokeApplier) newTxDiffFromPayment(pmt *payment, updateMinIntermediateBalance bool, info *fallibleValidationParams) (txDiff, error) {
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
	return diff, nil
}

func (ia *invokeApplier) newTxDiffFromScriptTransfer(scriptAddr *proto.Address, action *proto.TransferScriptAction, info *fallibleValidationParams) (txDiff, error) {
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

func (ia *invokeApplier) countIssuedAssets(actions []proto.ScriptAction) (uint64, error) {
	issuedAssetsCount := uint64(0)
	for _, action := range actions {
		switch a := action.(type) {
		case *proto.IssueScriptAction:
			assetParams := assetParams{a.Quantity, a.Decimals, a.Reissuable}
			nft, err := isNFT(ia.stor.features, assetParams)
			if err != nil {
				return 0, err
			}
			if !nft {
				issuedAssetsCount += 1
			}
		}
	}
	return issuedAssetsCount, nil
}

func (ia *invokeApplier) countActionScriptRuns(actions []proto.ScriptAction, initialisation bool) uint64 {
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
		isSmartAsset := ia.stor.scriptsStorage.newestIsSmartAsset(assetID, initialisation)
		if isSmartAsset {
			scriptRuns++
		}
	}
	return scriptRuns
}

func errorForSmartAsset(res ast.Result, asset crypto.Digest) error {
	var text string
	if res.Throw {
		text = fmt.Sprintf("Transaction is not allowed by token-script id %s: throw from asset script.", asset.String())
	} else {
		text = fmt.Sprintf("Transaction is not allowed by token-script id %s.", asset.String())
	}
	return errors.New(text)
}

type addlInvokeInfo struct {
	*fallibleValidationParams

	scriptAddr           *proto.Address
	scriptPK             crypto.PublicKey
	scriptRuns           uint64
	failedChanges        txBalanceChanges
	actions              []proto.ScriptAction
	paymentSmartAssets   []crypto.Digest
	disableSelfTransfers bool
}

func (ia *invokeApplier) fallibleValidation(tx *proto.InvokeScriptWithProofs, info *addlInvokeInfo) (proto.TxFailureReason, txBalanceChanges, error) {
	// Check smart asset scripts on payments.
	for _, smartAsset := range info.paymentSmartAssets {
		r, err := ia.sc.callAssetScript(tx, smartAsset, info.blockInfo, info.initialisation, info.acceptFailed)
		if err != nil {
			return proto.DAppError, info.failedChanges, errors.Errorf("failed to call asset %s script on payment: %v", smartAsset.String(), err)
		}
		if r.Failed() {
			return proto.SmartAssetOnPaymentFailure, info.failedChanges, errorForSmartAsset(r, smartAsset)
		}
	}
	// Resolve all aliases.
	// It has to be done before validation because we validate addresses, not aliases.
	if err := ia.resolveAliases(info.actions, info.initialisation); err != nil {
		return proto.DAppError, info.failedChanges, errors.New("ScriptResult; failed to resolve aliases")
	}
	// Validate produced actions.
	restrictions := proto.ActionsValidationRestrictions{DisableSelfTransfers: info.disableSelfTransfers, ScriptAddress: *info.scriptAddr}
	if err := proto.ValidateActions(info.actions, restrictions); err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	// Check full transaction fee (with actions and payments scripts).
	issuedAssetsCount, err := ia.countIssuedAssets(info.actions)
	if err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	if err := ia.checkFullFee(tx, info.scriptRuns, issuedAssetsCount); err != nil {
		return proto.InsufficientActionsFee, info.failedChanges, err
	}
	// Add feeAndPaymentChanges to stor before performing actions.
	differInfo := &differInfo{info.initialisation, info.blockInfo}
	feeAndPaymentChanges, err := ia.blockDiffer.createTransactionDiff(tx, info.block, differInfo)
	if err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	totalChanges := feeAndPaymentChanges
	if err := ia.saveIntermediateDiff(totalChanges.diff); err != nil {
		return proto.DAppError, info.failedChanges, err
	}
	// Perform actions.
	for _, action := range info.actions {
		switch a := action.(type) {
		case *proto.DataEntryScriptAction:
			// Perform data storage writes.
			ia.stor.accountsDataStor.appendEntryUncertain(*info.scriptAddr, a.Entry)
		case *proto.TransferScriptAction:
			// Perform transfers.
			addr := a.Recipient.Address
			totalChanges.appendAddr(*addr)
			assetExists := ia.stor.assets.newestAssetExists(a.Asset, !info.initialisation)
			if !assetExists {
				return proto.DAppError, info.failedChanges, errors.New("invalid asset in transfer")
			}
			isSmartAsset := ia.stor.scriptsStorage.newestIsSmartAsset(a.Asset.ID, !info.initialisation)
			if isSmartAsset {
				fullTr, err := proto.NewFullScriptTransfer(a, tx)
				if err != nil {
					return proto.DAppError, info.failedChanges, errors.Wrap(err, "failed to convert transfer to full script transfer")
				}
				// Call asset script if transferring smart asset.
				res, err := ia.sc.callAssetScriptWithScriptTransfer(fullTr, a.Asset.ID, info.blockInfo, info.initialisation, info.acceptFailed)
				if err != nil {
					return proto.DAppError, info.failedChanges, errors.Wrap(err, "failed to call asset script on transfer set")
				}
				if res.Failed() {
					return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset(res, a.Asset.ID)
				}
			}
			txDiff, err := ia.newTxDiffFromScriptTransfer(info.scriptAddr, a, info.fallibleValidationParams)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}
		case *proto.IssueScriptAction:
			// Create asset's info.
			assetInfo := &assetInfo{
				assetConstInfo: assetConstInfo{
					issuer:   info.scriptPK,
					decimals: int8(a.Decimals),
				},
				assetChangeableInfo: assetChangeableInfo{
					quantity:    *big.NewInt(a.Quantity),
					name:        a.Name,
					description: a.Description,
					reissuable:  a.Reissuable,
				},
			}
			ia.stor.assets.issueAssetUncertain(a.ID, assetInfo)
			// Currently asset script is always empty.
			// TODO: if this script is ever set, don't forget to
			// also save complexity for it here using saveComplexityForAsset().
			ia.stor.scriptsStorage.setAssetScriptUncertain(a.ID, proto.Script{}, info.scriptPK)
			txDiff, err := ia.newTxDiffFromScriptIssue(info.scriptAddr, a)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}
		case *proto.ReissueScriptAction:
			// Check validity of reissue.
			assetInfo, err := ia.stor.assets.newestAssetInfo(a.AssetID, !info.initialisation)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if assetInfo.issuer != info.scriptPK {
				return proto.DAppError, info.failedChanges, errs.NewAssetIssuedByOtherAddress("asset was issued by other address")
			}
			if !assetInfo.reissuable {
				return proto.DAppError, info.failedChanges, errors.New("attempt to reissue asset which is not reissuable")
			}
			if math.MaxInt64-a.Quantity < assetInfo.quantity.Int64() && info.block.Timestamp >= ia.settings.ReissueBugWindowTimeEnd {
				return proto.DAppError, info.failedChanges, errors.New("asset total value overflow")
			}
			ok, res, err := ia.validateActionSmartAsset(a.AssetID, a, info.scriptPK, info.blockInfo, *tx.ID, tx.Timestamp, info.initialisation, info.acceptFailed)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if !ok {
				return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset(res, a.AssetID)
			}
			// Update asset's info.
			change := &assetReissueChange{
				reissuable: a.Reissuable,
				diff:       a.Quantity,
			}
			if err := ia.stor.assets.reissueAssetUncertain(a.AssetID, change, !info.initialisation); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			txDiff, err := ia.newTxDiffFromScriptReissue(info.scriptAddr, a)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}
		case *proto.BurnScriptAction:
			// Check burn.
			assetInfo, err := ia.stor.assets.newestAssetInfo(a.AssetID, !info.initialisation)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			burnAnyTokensEnabled, err := ia.stor.features.isActivated(int16(settings.BurnAnyTokens))
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if !burnAnyTokensEnabled && assetInfo.issuer != info.scriptPK {
				return proto.DAppError, info.failedChanges, errors.New("asset was issued by other address")
			}
			ok, res, err := ia.validateActionSmartAsset(a.AssetID, a, info.scriptPK, info.blockInfo, *tx.ID, tx.Timestamp, info.initialisation, info.acceptFailed)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if !ok {
				return proto.SmartAssetOnActionFailure, info.failedChanges, errorForSmartAsset(res, a.AssetID)
			}
			// Update asset's info
			// Modify asset.
			change := &assetBurnChange{
				diff: int64(a.Quantity),
			}
			if err := ia.stor.assets.burnAssetUncertain(a.AssetID, change, !info.initialisation); err != nil {
				return proto.DAppError, info.failedChanges, errors.Wrap(err, "failed to burn asset")
			}
			txDiff, err := ia.newTxDiffFromScriptBurn(info.scriptAddr, a)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// diff must be saved to storage, because further asset scripts must take
			// recent balance changes into account.
			if err := ia.saveIntermediateDiff(txDiff); err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			// Append intermediate diff to common diff.
			for key, balanceDiff := range txDiff {
				if err := totalChanges.diff.appendBalanceDiffStr(key, balanceDiff); err != nil {
					return proto.DAppError, info.failedChanges, err
				}
			}
		case *proto.SponsorshipScriptAction:
			assetInfo, err := ia.stor.assets.newestAssetInfo(a.AssetID, !info.initialisation)
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			sponsorshipActivated, err := ia.stor.features.isActivated(int16(settings.FeeSponsorship))
			if err != nil {
				return proto.DAppError, info.failedChanges, err
			}
			if !sponsorshipActivated {
				return proto.DAppError, info.failedChanges, errors.New("sponsorship has not been activated yet")
			}
			if assetInfo.issuer != info.scriptPK {
				return proto.DAppError, info.failedChanges, errors.Errorf("asset %s was not issued by this DApp", a.AssetID.String())
			}
			isSmart := ia.stor.scriptsStorage.newestIsSmartAsset(a.AssetID, !info.initialisation)
			if isSmart {
				return proto.DAppError, info.failedChanges, errors.Errorf("can not sponsor smart asset %s", a.AssetID.String())
			}
			ia.stor.sponsoredAssets.sponsorAssetUncertain(a.AssetID, uint64(a.MinFee))
		default:
			return proto.DAppError, info.failedChanges, errors.Errorf("unsupported script action '%T'", a)
		}
	}
	if info.acceptFailed {
		// Validate total balance changes.
		if err := ia.diffApplier.validateTxDiff(totalChanges.diff, ia.invokeDiffStor.diffStorage, !info.initialisation); err != nil {
			// Total balance changes lead to negative balance, hence invoke has failed.
			// TODO: use different code for negative balances after it is introduced; use better error text here (addr + amount + asset).
			return proto.DAppError, info.failedChanges, err
		}
	}
	// If we are here, invoke succeeded.
	ia.blockDiffer.appendBlockInfoToTxDiff(totalChanges.diff, info.block)
	return 0, totalChanges, nil
}

// For InvokeScript transactions there is no performer function.
// Instead, here (in applyInvokeScript) we perform both balance and state changes
// along with fee validation which is normally done in checker function.
// We can not check fee in checker because before function invocation, we don't have Actions
// and can not evaluate how many smart assets (= script runs) will be involved, while this directly
// affects minimum allowed fee.
// That is why invoke transaction is applied to state in a different way - here, unlike other
// transaction types.
func (ia *invokeApplier) applyInvokeScript(tx *proto.InvokeScriptWithProofs, info *fallibleValidationParams) (*applicationResult, error) {
	// In defer we should clean all the temp changes invoke does to state.
	defer func() {
		ia.invokeDiffStor.invokeDiffsStor.reset()
		ia.stor.dropUncertain()
	}()

	// Check sender script, if any.
	if info.senderScripted {
		if err := ia.sc.callAccountScriptWithTx(tx, info.blockInfo, info.initialisation); err != nil {
			// Never accept invokes with failed script on transaction sender.
			return nil, err
		}
	}
	// Basic checks against state.
	paymentSmartAssets, err := ia.txHandler.checkTx(tx, info.checkerInfo)
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
	disableSelfTransfers := info.acceptFailed && script.Version >= 4
	if disableSelfTransfers && len(tx.Payments) > 0 {
		sender, err := proto.NewAddressFromPublicKey(ia.settings.AddressSchemeCharacter, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply script invocation")
		}
		if sender == *scriptAddr {
			return nil, errors.New("paying to DApp itself is forbidden since RIDE V4")
		}
	}
	// Basic differ for InvokeScript creates only fee and payment diff.
	// Create changes for both failed and successful scenarios.
	differInfo := &differInfo{info.initialisation, info.blockInfo}
	failedChanges, err := ia.blockDiffer.createFailedTransactionDiff(tx, info.block, differInfo)
	if err != nil {
		return nil, err
	}
	if !info.checkScripts {
		// Special mode when we don't check any fallible scripts.
		res := &invocationResult{failed: true, changes: failedChanges}
		return ia.handleInvocationResult(tx, info, res)
	}
	// Call script function.
	ok, scriptActions, err := ia.sc.invokeFunction(script, tx, info.blockInfo, *scriptAddr, info.initialisation)
	if !ok {
		// When ok is false, it means that we could not even start invocation.
		// We just return error in such case.
		return nil, errors.Wrap(err, "invokeFunction() failed")
	} else if err != nil {
		// If ok is true, but error is not nil, it means that invocation has failed.
		if !info.acceptFailed {
			return nil, errors.Wrap(err, "invokeFunction() failed")
		}
		res := &invocationResult{failed: true, code: proto.DAppError, text: err.Error(), actions: scriptActions, changes: failedChanges}
		return ia.handleInvocationResult(tx, info, res)
	}
	actionScriptRuns := ia.countActionScriptRuns(scriptActions, info.initialisation)
	scriptRuns := uint64(len(paymentSmartAssets)) + actionScriptRuns
	var res invocationResult
	code, changes, err := ia.fallibleValidation(tx, &addlInvokeInfo{
		fallibleValidationParams: info,
		scriptAddr:               scriptAddr,
		scriptPK:                 scriptPK,
		scriptRuns:               scriptRuns,
		failedChanges:            failedChanges,
		actions:                  scriptActions,
		paymentSmartAssets:       paymentSmartAssets,
		disableSelfTransfers:     disableSelfTransfers,
	})
	if err != nil {
		// If fallibleValidation fails, we should save transaction to blockchain when acceptFailed is true.
		if !info.acceptFailed {
			return nil, err
		}
		res = invocationResult{
			failed:     true,
			code:       code,
			text:       err.Error(),
			scriptRuns: scriptRuns,
			actions:    scriptActions,
			changes:    changes,
		}
	} else {
		res = invocationResult{
			failed:     false,
			scriptRuns: scriptRuns,
			actions:    scriptActions,
			changes:    changes,
		}
	}
	return ia.handleInvocationResult(tx, info, &res)
}

type invocationResult struct {
	failed bool
	code   proto.TxFailureReason
	text   string

	scriptRuns uint64
	actions    []proto.ScriptAction
	changes    txBalanceChanges
}

func toScriptResult(ir *invocationResult) (*proto.ScriptResult, error) {
	errorMsg := proto.ScriptErrorMessage{}
	if ir.failed {
		errorMsg = proto.ScriptErrorMessage{Code: ir.code, Text: ir.text}
	}
	return proto.NewScriptResult(ir.actions, errorMsg)
}

func (ia *invokeApplier) handleInvocationResult(tx *proto.InvokeScriptWithProofs, info *fallibleValidationParams, res *invocationResult) (*applicationResult, error) {
	if !res.failed && !info.validatingUtx {
		// Commit actions state changes.
		// TODO: when UTX transactions are validated, there is no block,
		// and we can not perform state changes.
		if err := ia.stor.commitUncertain(info.block.BlockID()); err != nil {
			return nil, err
		}
	}
	if ia.buildApiData && !info.validatingUtx {
		// Save invoke result for extended API.
		res, err := toScriptResult(res)
		if err != nil {
			return nil, errors.Wrap(err, "failed to save script result")
		}
		if err := ia.stor.invokeResults.saveResult(*tx.ID, res, info.block.BlockID()); err != nil {
			return nil, errors.Wrap(err, "failed to save script result")
		}
	}
	// Total scripts invoked = scriptRuns + invocation itself.
	totalScriptsInvoked := res.scriptRuns + 1
	return &applicationResult{
		totalScriptsRuns: totalScriptsInvoked,
		changes:          res.changes,
		status:           !res.failed,
	}, nil
}

func (ia *invokeApplier) checkFullFee(tx *proto.InvokeScriptWithProofs, scriptRuns, issuedAssetsCount uint64) error {
	sponsorshipActivated, err := ia.stor.features.isActivated(int16(settings.FeeSponsorship))
	if err != nil {
		return err
	}
	if !sponsorshipActivated {
		// Minimum fee is not checked before sponsorship activation.
		return nil
	}
	minIssueFee := feeConstants[proto.IssueTransaction] * FeeUnit * issuedAssetsCount
	minWavesFee := scriptExtraFee*scriptRuns + feeConstants[proto.InvokeScriptTransaction]*FeeUnit + minIssueFee
	wavesFee := tx.Fee
	if tx.FeeAsset.Present {
		wavesFee, err = ia.stor.sponsoredAssets.sponsoredAssetToWaves(tx.FeeAsset.ID, tx.Fee)
		if err != nil {
			return errs.Extend(err, "failed to convert fee asset to waves")
		}
	}
	if wavesFee < minWavesFee {
		feeAssetStr := tx.FeeAsset.String()
		return errors.Errorf("Fee in %s for InvokeScriptTransaction (%d in %s) with %d total scripts invoked does not exceed minimal value of %d WAVES", feeAssetStr, tx.Fee, feeAssetStr, scriptRuns, minWavesFee)
	}
	return nil
}

func (ia *invokeApplier) validateActionSmartAsset(asset crypto.Digest, action proto.ScriptAction, callerPK crypto.PublicKey,
	blockInfo *proto.BlockInfo, txID crypto.Digest, txTimestamp uint64, initialisation, acceptFailed bool) (bool, ast.Result, error) {
	isSmartAsset := ia.stor.scriptsStorage.newestIsSmartAsset(asset, !initialisation)
	if !isSmartAsset {
		return true, ast.Result{}, nil
	}
	obj, err := ast.NewVariablesFromScriptAction(ia.settings.AddressSchemeCharacter, action, callerPK, txID, txTimestamp)
	if err != nil {
		return false, ast.Result{}, err
	}
	res, err := ia.sc.callAssetScriptCommon(obj, asset, blockInfo, initialisation, acceptFailed)
	if err != nil {
		return false, ast.Result{}, err
	}
	ok := !res.Failed()
	return ok, res, nil
}
