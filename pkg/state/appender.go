package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type txAppender struct {
	state types.SmartState
	sc    *scriptCaller
	ia    *invokeApplier

	rw *blockReadWriter

	atx      *addressTransactions
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings

	// TransactionHandler is handler for any operations on transactions.
	txHandler *transactionHandler
	// Block differ is used to create diffs from blocks.
	blockDiffer *blockDiffer
	// Storage for diffs of incoming transactions (from added blocks or UTX).
	// It will be used for validation and applying diffs to existing balances.
	diffStor *diffStorage
	// diffStorInvoke is storage for partial diffs generated by Invoke transactions.
	// It is used to calculate balances that take into account intermediate invoke changes for RIDE.
	diffStorInvoke *diffStorageWrapped
	// Ids of all transactions whose diffs are currently in diffStor.
	// This is needed to check that transaction ids are unique.
	recentTxIds map[string]struct{}
	// diffApplier is used to both validate and apply balance diffs.
	diffApplier *diffApplier

	// totalScriptsRuns counts script runs for UTX validation.
	// It is increased every time ValidateNextTx() is called with transaction
	// that involved calling scripts.
	totalScriptsRuns uint64

	// buildApiData flag indicates that additional data for API is built when
	// appending transactions.
	buildApiData bool
}

func newTxAppender(
	state types.SmartState,
	rw *blockReadWriter,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
	stateDB *stateDB,
	atx *addressTransactions,
) (*txAppender, error) {
	sc, err := newScriptCaller(state, stor, settings)
	if err != nil {
		return nil, err
	}
	genesis := settings.Genesis
	txHandler, err := newTransactionHandler(genesis.BlockID(), stor, settings)
	if err != nil {
		return nil, err
	}
	blockDiffer, err := newBlockDiffer(txHandler, stor, settings)
	if err != nil {
		return nil, err
	}
	diffStor, err := newDiffStorage()
	if err != nil {
		return nil, err
	}
	diffStorInvoke, err := newDiffStorageWrapped(diffStor)
	if err != nil {
		return nil, err
	}
	diffApplier, err := newDiffApplier(stor.balances)
	if err != nil {
		return nil, err
	}
	buildApiData, err := stateDB.stateStoresApiData()
	if err != nil {
		return nil, err
	}
	ia := newInvokeApplier(state, sc, txHandler, stor, settings, blockDiffer, diffStorInvoke, diffApplier, buildApiData)
	return &txAppender{
		state:          state,
		sc:             sc,
		ia:             ia,
		rw:             rw,
		atx:            atx,
		stor:           stor,
		settings:       settings,
		txHandler:      txHandler,
		blockDiffer:    blockDiffer,
		recentTxIds:    make(map[string]struct{}),
		diffStor:       diffStor,
		diffStorInvoke: diffStorInvoke,
		diffApplier:    diffApplier,
		buildApiData:   buildApiData,
	}, nil
}

func (a *txAppender) checkDuplicateTxIdsImpl(id []byte, recentIds map[string]struct{}) error {
	// Check recent.
	if _, ok := recentIds[string(id)]; ok {
		return errors.Errorf("transaction with ID %v already in state", id)
	}
	// Check DB.
	if _, _, err := a.rw.readTransaction(id); err == nil {
		return errors.Errorf("transaction with ID %v already in state", id)
	}
	return nil
}

func (a *txAppender) checkDuplicateTxIds(tx proto.Transaction, recentIds map[string]struct{}, timestamp uint64) error {
	if tx.GetTypeInfo().Type == proto.PaymentTransaction {
		// Payment transactions are deprecated.
		return nil
	}
	if tx.GetTypeInfo().Type == proto.CreateAliasTransaction {
		if (timestamp >= a.settings.StolenAliasesWindowTimeStart) && (timestamp <= a.settings.StolenAliasesWindowTimeEnd) {
			// At this period alias transactions might have duplicate IDs due to bugs in historical blockchain.
			return nil
		}
	}
	txID, err := tx.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	return a.checkDuplicateTxIdsImpl(txID, recentIds)
}

type appendBlockParams struct {
	transactions   []proto.Transaction
	chans          *verifierChans
	block, parent  *proto.BlockHeader
	height         uint64
	initialisation bool
}

func (a *txAppender) hasAccountVerifyScript(tx proto.Transaction, initialisation bool) (bool, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return false, err
	}
	return a.stor.scriptsStorage.newestAccountHasVerifier(senderAddr, !initialisation)
}

func (a *txAppender) orderIsScripted(order proto.Order, initialisation bool) (bool, error) {
	return a.txHandler.tc.orderScriptedAccount(order, initialisation)
}

// For UTX validation, this returns the last stable block, which is in fact
// current block.
// For appendBlock(), this returns block that is currently being added.
func (a *txAppender) currentBlock() (*proto.BlockHeader, error) {
	curBlockHeight, err := a.state.AddingBlockHeight()
	if err != nil {
		return nil, err
	}
	curHeader, err := a.state.NewestHeaderByHeight(curBlockHeight)
	if err != nil {
		return nil, err
	}
	return curHeader, nil
}

func (a *txAppender) currentBlockInfo() (*proto.BlockInfo, error) {
	height, err := a.state.AddingBlockHeight()
	if err != nil {
		return nil, err
	}
	curHeader, err := a.currentBlock()
	if err != nil {
		return nil, err
	}
	hs, err := a.state.BlockVRF(curHeader, height)
	if err != nil {
		return nil, err
	}
	return proto.BlockInfoFromHeader(a.settings.AddressSchemeCharacter, curHeader, height, hs)
}

func (a *txAppender) checkProtobufVersion(tx proto.Transaction, blockV5Activated bool) error {
	if !proto.IsProtobufTx(tx) {
		return nil
	}
	if !blockV5Activated {
		return errors.Errorf("bad transaction version %v before blockV5 activation", tx.GetVersion())
	}
	return nil
}

func (a *txAppender) checkTxFees(tx proto.Transaction, info *checkerInfo, blockInfo *proto.BlockInfo) error {
	if tx.GetTypeInfo().Type == proto.LeaseCancelTransaction {
		return nil
	}
	feeChanges, err := a.blockDiffer.createTransactionFeeDiff(tx, blockInfo, info.initialisation)
	if err != nil {
		return err
	}
	changes, err := a.diffStor.changesByTxDiff(feeChanges.diff)
	if err != nil {
		return err
	}
	return a.diffApplier.validateBalancesChanges(changes, true)
}

// This functions is used for script validation of transaction that can't fail
func (a *txAppender) checkTransactionScripts(tx proto.Transaction, accountScripted bool, checkerInfo *checkerInfo, blockInfo *proto.BlockInfo) (uint64, error) {
	scriptsRuns := uint64(0)
	if accountScripted {
		// Check script.
		_, err := a.sc.callAccountScriptWithTx(tx, blockInfo, checkerInfo.initialisation, false)
		if err != nil {
			return 0, err
		}
		scriptsRuns++
	}
	// Check against state.
	txSmartAssets, err := a.txHandler.checkTx(tx, checkerInfo)
	if err != nil {
		return 0, err
	}
	ride4DAppsActivated, err := a.stor.features.isActivated(int16(settings.Ride4DApps))
	if err != nil {
		return 0, err
	}
	for _, smartAsset := range txSmartAssets {
		// Check smart asset's script.
		_, err := a.sc.callAssetScript(tx, smartAsset, blockInfo, checkerInfo.initialisation, false)
		if err != nil {
			return 0, err
		}
		if tx.GetTypeInfo().Type == proto.SetAssetScriptTransaction && !ride4DAppsActivated {
			// Exception: don't count before Ride4DApps activation.
			continue
		}
		scriptsRuns++
	}
	return scriptsRuns, nil
}

func (a *txAppender) checkExchangeTransactionScripts(tx proto.Transaction, accountScripted bool, checkerInfo *checkerInfo, blockInfo *proto.BlockInfo, acceptFailed bool) (uint64, bool, error) {
	exchange, ok := tx.(proto.Exchange)
	if !ok {
		return 0, false, errors.New("failed to convert transaction to Exchange")
	}
	scriptsRuns := uint64(0)
	// Check script on account
	if accountScripted {
		// Check script.
		ok, err := a.sc.callAccountScriptWithTx(tx, blockInfo, checkerInfo.initialisation, acceptFailed)
		if err != nil {
			return 0, false, err
		}
		scriptsRuns++
		if !ok {
			return scriptsRuns, ok, nil
		}
	}
	// Validate transaction, orders and extract smart assets
	txSmartAssets, err := a.txHandler.checkTx(tx, checkerInfo)
	if err != nil {
		return 0, false, err
	}
	// Smart account trading.
	smartAccountTradingActivated, err := a.stor.features.isActivated(int16(settings.SmartAccountTrading))
	if err != nil {
		return 0, false, err
	}
	// Check smart assets' scripts
	for _, smartAsset := range txSmartAssets {
		ok, err := a.sc.callAssetScript(tx, smartAsset, blockInfo, checkerInfo.initialisation, acceptFailed)
		if err != nil {
			return 0, false, err
		}
		scriptsRuns++
		if !ok {
			return scriptsRuns, ok, nil
		}
	}
	if !smartAccountTradingActivated {
		// Following checks are not required because functionality is not yet activated.
		return scriptsRuns, true, nil
	}
	o1 := exchange.GetOrder1()
	o2 := exchange.GetOrder2()
	o1Scripted, err := a.orderIsScripted(o1, checkerInfo.initialisation)
	if err != nil {
		return 0, false, err
	}
	o2Scripted, err := a.orderIsScripted(o2, checkerInfo.initialisation)
	if err != nil {
		return 0, false, err
	}
	if o1Scripted {
		ok, err := a.sc.callAccountScriptWithOrder(o1, blockInfo, checkerInfo.initialisation, acceptFailed)
		if err != nil {
			return 0, false, errors.Wrap(err, "failed to call script on first order")
		}
		scriptsRuns++
		if !ok {
			return scriptsRuns, ok, nil
		}
	}
	if o2Scripted {
		ok, err := a.sc.callAccountScriptWithOrder(o2, blockInfo, checkerInfo.initialisation, acceptFailed)
		if err != nil {
			return 0, false, errors.Wrap(err, "failed to call script on second order")
		}
		scriptsRuns++
		if !ok {
			return scriptsRuns, ok, nil
		}
	}
	ride4DAppsActivated, err := a.stor.features.isActivated(int16(settings.Ride4DApps))
	if err != nil {
		return 0, false, err
	}
	if !ride4DAppsActivated {
		// Don't count before Ride4DApps activation.
		scriptsRuns = 0
	}
	return scriptsRuns, true, nil
}

func (a *txAppender) checkScriptsLimits(scriptsRuns uint64) error {
	smartAccountsActivated, err := a.stor.features.isActivated(int16(settings.SmartAccounts))
	if err != nil {
		return err
	}
	ride4DAppsActivated, err := a.stor.features.isActivated(int16(settings.Ride4DApps))
	if err != nil {
		return err
	}
	if ride4DAppsActivated {
		if a.sc.getTotalComplexity() > maxScriptsComplexityInBlock {
			zap.S().Warn("complexity limit per block is exceeded")
		}
		return nil
	} else if smartAccountsActivated {
		if scriptsRuns > maxScriptsRunsInBlock {
			return errors.Errorf("more scripts runs in block than allowed: %d > %d", scriptsRuns, maxScriptsRunsInBlock)
		}
	}
	return nil
}

func (a *txAppender) needToCheckOrdersSigs(transaction proto.Transaction, initialisation bool) (bool, bool, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return false, false, nil
	}
	o1Scripted, err := a.orderIsScripted(tx.GetOrder1(), initialisation)
	if err != nil {
		return false, false, err
	}
	o2Scripted, err := a.orderIsScripted(tx.GetOrder2(), initialisation)
	if err != nil {
		return false, false, err
	}
	return !o1Scripted, !o2Scripted, nil
}

func (a *txAppender) saveTransactionIdByAddresses(addrs []proto.Address, txID []byte, blockID proto.BlockID, filter bool) error {
	for _, addr := range addrs {
		if err := a.atx.saveTxIdByAddress(addr, txID, blockID, filter); err != nil {
			return err
		}
	}
	return nil
}

func (a *txAppender) appendBlock(params *appendBlockParams) error {
	blockID := params.block.BlockID()
	hasParent := params.parent != nil
	// Create miner balance diff.
	// This adds 60% of prev block fees as very first balance diff of the current block
	// in case NG is activated, or empty diff otherwise.
	minerDiff, err := a.blockDiffer.createMinerDiff(params.block, hasParent, params.height)
	if err != nil {
		return err
	}
	// Save miner diff first.
	if err := a.diffStor.saveTxDiff(minerDiff); err != nil {
		return err
	}
	curHeight := params.height + 1
	scriptsRuns := uint64(0)
	blockInfo, err := a.currentBlockInfo()
	if err != nil {
		return err
	}
	//TODO: join flags then after joining features
	blockV5Activated, err := a.stor.features.isActivated(int16(settings.BlockV5))
	if err != nil {
		return err
	}
	acceptFailedActivated, err := a.stor.features.isActivated(int16(settings.AcceptFailedScriptTransaction))
	if err != nil {
		return err
	}
	// Check transactions
	for _, tx := range params.transactions {
		// Check that Protobuf transaction could be accepted
		if err := a.checkProtobufVersion(tx, blockV5Activated); err != nil {
			return err
		}
		// Detect what signatures must be checked for this transaction.
		senderAddr, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, tx.GetSenderPK())
		if err != nil {
			return err
		}
		accountHasVerifierScript, err := a.stor.scriptsStorage.newestAccountHasVerifier(senderAddr, !params.initialisation)
		if err != nil {
			return err
		}
		checkTxSig := true
		if accountHasVerifierScript {
			// For transaction with SmartAccount we don't check signatures.
			checkTxSig = false
		}
		checkOrder1, checkOrder2, err := a.needToCheckOrdersSigs(tx, params.initialisation)
		if err != nil {
			return err
		}
		// Send transaction for validation of transaction's data correctness (using tx.Valid() method)
		// and simple cryptographic signature verification (using tx.Verify() and PK).
		task := &verifyTask{
			taskType:    verifyTx,
			tx:          tx,
			checkTxSig:  checkTxSig,
			checkOrder1: checkOrder1,
			checkOrder2: checkOrder2,
		}
		select {
		case verifyError := <-params.chans.errChan:
			return verifyError
		case params.chans.tasksChan <- task:
		}
		checkerInfo := &checkerInfo{
			initialisation:   params.initialisation,
			currentTimestamp: params.block.Timestamp,
			blockID:          blockID,
			blockVersion:     params.block.Version,
			height:           params.height,
		}
		if hasParent {
			checkerInfo.parentTimestamp = params.parent.Timestamp
		}
		// Check transaction for duplication of it's ID
		if err := a.checkDuplicateTxIds(tx, a.recentTxIds, params.block.Timestamp); err != nil {
			return err
		}
		// Add transaction ID to recent IDs.
		txID, err := tx.GetID(a.settings.AddressSchemeCharacter)
		if err != nil {
			return err
		}
		a.recentTxIds[string(txID)] = empty
		// After activation of AcceptFailedScriptTransactions feature we have to check availability of funds to
		// pay fees for all transaction types except LeaseCancel.
		if acceptFailedActivated {
			err := a.checkTxFees(tx, checkerInfo, blockInfo)
			if err != nil {
				return errors.Errorf("not enough balance to pay transaction's fees")
			}
		}
		// Status indicates that Invoke or Exchange transaction's scripts may have failed
		// but it have to be stored in state anyway. For other transactions it always true.
		status := true
		// The list of addresses that was used in transaction, to store the link to the transaction in extended API
		var addresses []proto.Address
		var scriptsRuns uint64
		// Some transaction types should be handled differently
		switch tx.GetTypeInfo().Type {
		case proto.InvokeScriptTransaction:
			// Invoke is handled in a special way.
			invokeTx, ok := tx.(*proto.InvokeScriptWithProofs)
			if !ok {
				return errors.New("failed to convert InvokeScriptTransaction to type InvokeScriptWithProofs")
			}
			txScriptsRuns, err := a.checkTransactionScripts(tx, accountHasVerifierScript, checkerInfo, blockInfo)
			if err != nil {
				return err
			}
			scriptsRuns += txScriptsRuns
			invokeInfo := &invokeAddlInfo{
				previousScriptRuns: txScriptsRuns,
				initialisation:     params.initialisation,
				block:              params.block,
				height:             curHeight,
				hitSource:          blockInfo.VRF,
				validatingUtx:      false,
			}
			addresses, status, err = a.ia.applyInvokeScriptWithProofs(invokeTx, invokeInfo, acceptFailedActivated)
			if err != nil {
				return errors.Errorf("failed to apply InvokeScript transaction %s to state: %v", invokeTx.ID.String(), err)
			}
		case proto.ExchangeTransaction:
			// Exchange is handled in a special way also
			var txScriptsRuns uint64
			txScriptsRuns, status, err = a.checkExchangeTransactionScripts(tx, accountHasVerifierScript, checkerInfo, blockInfo, acceptFailedActivated)
			if err != nil {
				return err
			}
			scriptsRuns += txScriptsRuns
			// Create balance diff of this tx.
			txChanges, err := a.blockDiffer.createTransactionDiff(tx, params.block, curHeight, blockInfo.VRF, params.initialisation, status)
			if err != nil {
				return err
			}
			addresses = txChanges.addresses()
			// Save balance diff of this tx.
			if err := a.diffStor.saveTxDiff(txChanges.diff); err != nil {
				return err
			}
		default:
			// Execute transaction's scripts.
			txScriptsRuns, err := a.checkTransactionScripts(tx, accountHasVerifierScript, checkerInfo, blockInfo)
			if err != nil {
				return err
			}
			scriptsRuns += txScriptsRuns
			// Create balance diff of this tx.
			txChanges, err := a.blockDiffer.createTransactionDiff(tx, params.block, curHeight, blockInfo.VRF, params.initialisation, true)
			if err != nil {
				return err
			}
			addresses = txChanges.addresses()
			// Save balance diff of this tx.
			if err := a.diffStor.saveTxDiff(txChanges.diff); err != nil {
				return err
			}
		}
		// Count current tx fee.
		if err := a.blockDiffer.countMinerFee(tx); err != nil {
			return err
		}
		// Perform state changes.
		performerInfo := &performerInfo{
			initialisation: params.initialisation,
			height:         params.height,
			blockID:        blockID,
		}
		if err := a.txHandler.performTx(tx, performerInfo); err != nil {
			return err
		}
		// Save transaction to storage.
		if err := a.rw.writeTransaction(tx, !status); err != nil {
			return err
		}
		// Store additional data for API: transaction by address.
		if a.buildApiData {
			if err := a.saveTransactionIdByAddresses(addresses, txID, blockID, !params.initialisation); err != nil {
				return err
			}
		}
	}
	if err := a.checkScriptsLimits(scriptsRuns); err != nil {
		return errors.Errorf("%s: %v", blockID.String(), err)
	}
	// Reset block complexity counter.
	a.sc.resetComplexity()
	// Save fee distribution of this block.
	// This will be needed for createMinerDiff() of next block due to NG.
	if err := a.blockDiffer.saveCurFeeDistr(params.block); err != nil {
		return err
	}
	return nil
}

func (a *txAppender) applyAllDiffs(initialisation bool) error {
	changes := a.diffStor.allChanges()
	a.recentTxIds = make(map[string]struct{})
	a.diffStor.reset()
	if err := a.diffApplier.applyBalancesChanges(changes, !initialisation); err != nil {
		return err
	}
	return nil
}

func (a *txAppender) checkUtxTxSig(tx proto.Transaction, scripted bool) error {
	// Check tx signature and data.
	checkOrder1, checkOrder2, err := a.needToCheckOrdersSigs(tx, false)
	if err != nil {
		return err
	}
	if err := checkTx(tx, !scripted, checkOrder1, checkOrder2, a.settings.AddressSchemeCharacter); err != nil {
		return err
	}
	return nil
}

func (a *txAppender) handleInvoke(tx proto.Transaction, height uint64, block *proto.BlockHeader, prevScriptRuns uint64, acceptFailed bool) error {
	invokeTx, ok := tx.(*proto.InvokeScriptWithProofs)
	if !ok {
		return errors.New("failed to convert transaction to type InvokeScriptWithProofs")
	}
	invokeInfo := &invokeAddlInfo{
		previousScriptRuns: prevScriptRuns,
		initialisation:     false,
		block:              block,
		height:             height,
		validatingUtx:      true,
	}
	_, _, err := a.ia.applyInvokeScriptWithProofs(invokeTx, invokeInfo, acceptFailed)
	if err != nil {
		return errors.Wrap(err, "InvokeScript validation failed")
	}
	return nil
}

func (a *txAppender) resetValidationList() {
	a.sc.resetComplexity()
	a.totalScriptsRuns = 0
	a.recentTxIds = make(map[string]struct{})
	a.diffStor.reset()
}

// For UTX validation.
func (a *txAppender) validateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, version proto.BlockVersion, vrf []byte, acceptFailed bool) error {
	if err := a.checkDuplicateTxIds(tx, a.recentTxIds, currentTimestamp); err != nil {
		return err
	}
	// Add transaction ID.
	txID, err := tx.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	a.recentTxIds[string(txID)] = empty
	scripted, err := a.hasAccountVerifyScript(tx, false)
	if err != nil {
		return err
	}
	// Check tx signature and data.
	if err := a.checkUtxTxSig(tx, scripted); err != nil {
		return err
	}
	// Check tx data against state.
	height, err := a.state.AddingBlockHeight()
	if err != nil {
		return err
	}
	checkerInfo := &checkerInfo{
		initialisation:   false,
		currentTimestamp: currentTimestamp,
		parentTimestamp:  parentTimestamp,
		blockVersion:     version,
		height:           height,
	}
	// TODO: Doesn't work correctly if miner doesn't work in NG mode.
	// In this case it returns the last block instead of what is being mined.
	block, err := a.currentBlock()
	if err != nil {
		return err
	}
	blockInfo, err := proto.BlockInfoFromHeader(a.settings.AddressSchemeCharacter, block, height, vrf)
	if err != nil {
		return err
	}

	txScriptsRuns := uint64(0)
	ok := true
	switch tx.GetTypeInfo().Type {
	case proto.InvokeScriptTransaction:
		// Invoke is handled in a special way.
		return a.handleInvoke(tx, height, block, txScriptsRuns, acceptFailed) //todo: check
	case proto.ExchangeTransaction:
		txScriptsRuns, ok, err = a.checkExchangeTransactionScripts(tx, scripted, checkerInfo, blockInfo, acceptFailed)
		if err != nil {
			return err
		}
	default:
		txScriptsRuns, err = a.checkTransactionScripts(tx, scripted, checkerInfo, blockInfo)
		if err != nil {
			return err
		}
	}
	if err := a.checkScriptsLimits(a.totalScriptsRuns + txScriptsRuns); err != nil {
		return err
	}
	a.totalScriptsRuns += txScriptsRuns
	// Create, validate and save balance diff
	var txDiff txBalanceChanges
	if ok {
		txDiff, err = a.txHandler.createDiffTx(tx, &differInfo{
			initialisation: false,
			blockInfo:      &proto.BlockInfo{Timestamp: currentTimestamp},
		})
		if err != nil {
			return err
		}
	} else {
		txDiff, err = a.txHandler.createFeeDiffTx(tx, &differInfo{
			initialisation: false,
			blockInfo:      &proto.BlockInfo{Timestamp: currentTimestamp},
		})
		if err != nil {
			return err
		}
	}
	changes, err := a.diffStor.changesByTxDiff(txDiff.diff)
	if err != nil {
		return err
	}
	if err := a.diffApplier.validateBalancesChanges(changes, true); err != nil {
		return err
	}
	if err := a.diffStor.saveBalanceChanges(changes); err != nil {
		return err
	}
	return nil
}

func (a *txAppender) reset() {
	a.sc.resetComplexity()
	a.totalScriptsRuns = 0
	a.recentTxIds = make(map[string]struct{})
	a.diffStor.reset()
	a.blockDiffer.reset()
}
