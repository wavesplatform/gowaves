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

// For UTX validation, this returns the last stable block, which is in fact current block.
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

func (a *txAppender) checkTxFees(tx proto.Transaction, info *fallibleValidationParams) error {
	differInfo := &differInfo{info.initialisation, info.blockInfo}
	var feeChanges txBalanceChanges
	var err error
	switch tx.GetTypeInfo().Type {
	case proto.ExchangeTransaction:
		feeChanges, err = a.txHandler.td.createDiffForExchangeFeeValidation(tx, differInfo)
		if err != nil {
			return err
		}
	case proto.InvokeScriptTransaction:
		feeChanges, err = a.txHandler.td.createFeeDiffInvokeScriptWithProofs(tx, differInfo)
		if err != nil {
			return err
		}
	}
	return a.diffApplier.validateTxDiff(feeChanges.diff, a.diffStor, !info.initialisation)
}

// This function is used for script validation of transaction that can't fail.
func (a *txAppender) checkTransactionScripts(tx proto.Transaction, accountScripted bool, checkerInfo *checkerInfo, blockInfo *proto.BlockInfo) (uint64, error) {
	scriptsRuns := uint64(0)
	if accountScripted {
		// Check script.
		if err := a.sc.callAccountScriptWithTx(tx, blockInfo, checkerInfo.initialisation); err != nil {
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
	// Reset block complexity counter.
	defer a.sc.resetComplexity()

	blockID := params.block.BlockID()
	hasParent := params.parent != nil
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
	scriptsRuns := uint64(0)
	blockInfo, err := a.currentBlockInfo()
	if err != nil {
		return err
	}
	blockV5Activated, err := a.stor.features.isActivated(int16(settings.BlockV5))
	if err != nil {
		return err
	}
	// Check transactions.
	for _, tx := range params.transactions {
		// Check that Protobuf transactions are accepted.
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
		// Check transaction for duplication of it's ID.
		if err := a.checkDuplicateTxIds(tx, a.recentTxIds, params.block.Timestamp); err != nil {
			return err
		}
		// Add transaction ID to recent IDs.
		txID, err := tx.GetID(a.settings.AddressSchemeCharacter)
		if err != nil {
			return err
		}
		a.recentTxIds[string(txID)] = empty

		// Status indicates that Invoke or Exchange transaction's scripts could failed
		// but they have to be stored in state anyway.
		// For other transactions it is always true.
		status := true
		var txChanges txBalanceChanges
		switch tx.GetTypeInfo().Type {
		case proto.InvokeScriptTransaction, proto.ExchangeTransaction:
			// Invoke and Exchange transactions should be handled differently.
			// They may fail, and will be saved to blockchain anyway.
			fallibleInfo := &fallibleValidationParams{
				checkerInfo:    checkerInfo,
				blockInfo:      blockInfo,
				block:          params.block,
				senderScripted: accountHasVerifierScript,
				checkScripts:   true,
				acceptFailed:   blockV5Activated,
				validatingUtx:  false,
				initialisation: params.initialisation,
			}
			applicationInfo, err := a.handleFallible(tx, fallibleInfo)
			if err != nil {
				return err
			}
			scriptsRuns += applicationInfo.totalScriptsRuns
			txChanges = applicationInfo.changes
			status = applicationInfo.status
		default:
			// Execute transaction's scripts.
			txScriptsRuns, err := a.checkTransactionScripts(tx, accountHasVerifierScript, checkerInfo, blockInfo)
			if err != nil {
				return err
			}
			scriptsRuns += txScriptsRuns
			// Create balance diff of this tx.
			differInfo := &differInfo{params.initialisation, blockInfo}
			txChanges, err = a.blockDiffer.createTransactionDiff(tx, params.block, differInfo)
			if err != nil {
				return err
			}
		}
		// Save balance diff of this tx.
		if err := a.diffStor.saveTxDiff(txChanges.diff); err != nil {
			return err
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
			if err := a.saveTransactionIdByAddresses(txChanges.addresses(), txID, blockID, !params.initialisation); err != nil {
				return err
			}
		}
	}
	if err := a.checkScriptsLimits(scriptsRuns); err != nil {
		return errors.Errorf("%s: %v", blockID.String(), err)
	}
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

type fallibleValidationParams struct {
	checkerInfo    *checkerInfo
	blockInfo      *proto.BlockInfo
	block          *proto.BlockHeader
	senderScripted bool
	checkScripts   bool
	acceptFailed   bool
	validatingUtx  bool
	initialisation bool
}

type applicationResult struct {
	status           bool
	totalScriptsRuns uint64
	changes          txBalanceChanges
}

func (a *txAppender) handleInvoke(tx proto.Transaction, info *fallibleValidationParams) (*applicationResult, error) {
	invokeTx, ok := tx.(*proto.InvokeScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert transaction to type InvokeScriptWithProofs")
	}
	res, err := a.ia.applyInvokeScriptWithProofs(invokeTx, info)
	if err != nil {
		return nil, errors.Errorf("failed to apply InvokeScript transaction %s to state: %v", invokeTx.ID.String(), err)
	}
	return res, nil
}

func (a *txAppender) countExchangeScriptsRuns(scriptsRuns uint64) (uint64, error) {
	// Some bug in historical blockchain, no logic here.
	ride4DAppsActivated, err := a.stor.features.isActivated(int16(settings.Ride4DApps))
	if err != nil {
		return 0, err
	}
	if !ride4DAppsActivated {
		// Don't count before Ride4DApps activation.
		return 0, nil
	}
	return scriptsRuns, nil
}

func (a *txAppender) handleExchange(tx proto.Transaction, info *fallibleValidationParams) (*applicationResult, error) {
	exchange, ok := tx.(proto.Exchange)
	if !ok {
		return nil, errors.New("failed to convert transaction to Exchange")
	}
	scriptsRuns := uint64(0)
	// At first, we call accounts and orders scripts which must not fail.
	if info.senderScripted {
		// Check script on account.
		err := a.sc.callAccountScriptWithTx(tx, info.blockInfo, info.initialisation)
		if err != nil {
			return nil, err
		}
		scriptsRuns++
	}
	// Smart account trading.
	smartAccountTradingActivated, err := a.stor.features.isActivated(int16(settings.SmartAccountTrading))
	if err != nil {
		return nil, err
	}
	if smartAccountTradingActivated {
		// Check orders scripts.
		o1 := exchange.GetOrder1()
		o2 := exchange.GetOrder2()
		o1Scripted, err := a.orderIsScripted(o1, info.initialisation)
		if err != nil {
			return nil, err
		}
		o2Scripted, err := a.orderIsScripted(o2, info.initialisation)
		if err != nil {
			return nil, err
		}
		if o1Scripted {
			if err := a.sc.callAccountScriptWithOrder(o1, info.blockInfo, info.initialisation); err != nil {
				return nil, errors.Wrap(err, "script failure on first order")
			}
			scriptsRuns++
		}
		if o2Scripted {
			if err := a.sc.callAccountScriptWithOrder(o2, info.blockInfo, info.initialisation); err != nil {
				return nil, errors.Wrap(err, "script failure on second order")
			}
			scriptsRuns++
		}
	}
	// Validate transaction, orders and extract smart assets.
	txSmartAssets, err := a.txHandler.checkTx(tx, info.checkerInfo)
	if err != nil {
		return nil, err
	}
	// Count total scripts runs.
	scriptsRuns += uint64(len(txSmartAssets))
	scriptsRuns, err = a.countExchangeScriptsRuns(scriptsRuns)
	if err != nil {
		return nil, err
	}
	// Create balance changes for both failure and success.
	differInfo := &differInfo{info.initialisation, info.blockInfo}
	failedChanges, err := a.blockDiffer.createFailedTransactionDiff(tx, info.block, differInfo)
	if err != nil {
		return nil, err
	}
	successfulChanges, err := a.blockDiffer.createTransactionDiff(tx, info.block, differInfo)
	if err != nil {
		return nil, err
	}
	if !info.checkScripts {
		// There is special mode for UTX validation when we don't check any scripts which might fail.
		// Instead, we just return failed balance diff here.
		return &applicationResult{false, scriptsRuns, failedChanges}, nil
	}
	// Check smart assets' scripts.
	for _, smartAsset := range txSmartAssets {
		res, err := a.sc.callAssetScript(tx, smartAsset, info.blockInfo, info.initialisation, info.acceptFailed)
		if err != nil && !info.acceptFailed {
			return nil, err
		}
		if err != nil || res.Failed() {
			// Smart asset script failed, return failed diff.
			return &applicationResult{false, scriptsRuns, failedChanges}, nil
		}
	}
	if info.acceptFailed {
		// If accepting failed, we must also check resulting balances.
		filter := !info.initialisation
		if err := a.diffApplier.validateTxDiff(successfulChanges.diff, a.diffStor, filter); err != nil {
			// Not enough balance for successful diff = fail, return failed diff.
			// We only check successful diff for negative balances, because failed diff is already checked in checkTxFees().
			return &applicationResult{false, scriptsRuns, failedChanges}, nil
		}
	}
	// Return successful diff.
	return &applicationResult{true, scriptsRuns, successfulChanges}, nil
}

func (a *txAppender) handleFallible(tx proto.Transaction, info *fallibleValidationParams) (*applicationResult, error) {
	if info.acceptFailed {
		if err := a.checkTxFees(tx, info); err != nil {
			return nil, err
		}
	}
	switch tx.GetTypeInfo().Type {
	case proto.InvokeScriptTransaction:
		return a.handleInvoke(tx, info)
	case proto.ExchangeTransaction:
		return a.handleExchange(tx, info)
	}
	return nil, errors.New("transaction is not fallible")
}

func (a *txAppender) resetValidationList() {
	a.sc.resetComplexity()
	a.totalScriptsRuns = 0
	a.recentTxIds = make(map[string]struct{})
	a.diffStor.reset()
	a.stor.reset()
}

// For UTX validation.
func (a *txAppender) validateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, version proto.BlockVersion, checkScripts bool) error {
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
	// TODO: Doesn't work correctly if miner doesn't work in NG mode.
	// In this case it returns the last block instead of what is being mined.
	block, err := a.currentBlock()
	if err != nil {
		return err
	}
	blockInfo, err := a.currentBlockInfo()
	if err != nil {
		return err
	}
	blockInfo.Timestamp = currentTimestamp
	checkerInfo := &checkerInfo{
		initialisation:   false,
		currentTimestamp: currentTimestamp,
		parentTimestamp:  parentTimestamp,
		blockVersion:     version,
		height:           blockInfo.Height,
	}
	blockV5Activated, err := a.stor.features.isActivated(int16(settings.BlockV5))
	if err != nil {
		return err
	}
	// Check tx data against state.
	var txScriptsRuns uint64
	var changes txBalanceChanges
	switch tx.GetTypeInfo().Type {
	case proto.InvokeScriptTransaction, proto.ExchangeTransaction:
		fallibleInfo := &fallibleValidationParams{
			checkerInfo:    checkerInfo,
			blockInfo:      blockInfo,
			block:          block,
			senderScripted: scripted,
			checkScripts:   checkScripts,
			acceptFailed:   blockV5Activated,
			validatingUtx:  true,
			initialisation: false,
		}
		applicationInfo, err := a.handleFallible(tx, fallibleInfo)
		if err != nil {
			return err
		}
		txScriptsRuns = applicationInfo.totalScriptsRuns
		changes = applicationInfo.changes
	default:
		txScriptsRuns, err = a.checkTransactionScripts(tx, scripted, checkerInfo, blockInfo)
		if err != nil {
			return err
		}
		// Create balance diff.
		differInfo := &differInfo{false, blockInfo}
		changes, err = a.blockDiffer.createTransactionDiff(tx, block, differInfo)
		if err != nil {
			return err
		}
		// Validate tx diff.
		if err := a.diffApplier.validateTxDiff(changes.diff, a.diffStor, true); err != nil {
			return err
		}
	}
	// Validate script runs.
	if err := a.checkScriptsLimits(a.totalScriptsRuns + txScriptsRuns); err != nil {
		return err
	}
	a.totalScriptsRuns += txScriptsRuns
	// Save balance diff.
	if err := a.diffStor.saveTxDiff(changes.diff); err != nil {
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
