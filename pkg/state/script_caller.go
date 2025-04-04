package state

import (
	"fmt"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type scriptCaller struct {
	state types.EnrichedSmartState

	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings

	totalComplexity    uint64
	recentTxComplexity uint64
}

func newScriptCaller(
	state types.EnrichedSmartState,
	stor *blockchainEntitiesStorage,
	settings *settings.BlockchainSettings,
) (*scriptCaller, error) {
	return &scriptCaller{
		state:    state,
		stor:     stor,
		settings: settings,
	}, nil
}

// callAccountScriptWithOrder calls account script. This method must not be called for proto.EthereumAddress.
func (a *scriptCaller) callAccountScriptWithOrder(order proto.Order, lastBlockInfo *proto.BlockInfo, info *fallibleValidationParams) error {
	senderAddr, err := order.GetSender(a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	senderWavesAddr, ok := senderAddr.(proto.WavesAddress)
	if !ok {
		return errors.Errorf("address %q must be a waves address, not %T", senderAddr.String(), senderAddr)
	}
	id, err := order.GetID()
	if err != nil {
		return err
	}
	tree, err := a.stor.scriptsStorage.newestScriptByAddr(senderWavesAddr)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve account script")
	}
	env, err := ride.NewEnvironment(
		a.settings.AddressSchemeCharacter,
		a.state,
		a.settings.InternalInvokePaymentsValidationAfterHeight,
		a.settings.PaymentsFixAfterHeight,
		info.blockV5Activated,
		info.rideV6Activated,
		info.consensusImprovementsActivated,
		info.blockRewardDistributionActivated,
		info.lightNodeActivated,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create RIDE environment")
	}
	env.SetThisFromAddress(senderWavesAddr)
	env.ChooseSizeCheck(tree.LibVersion)
	if err = env.SetLastBlockFromBlockInfo(lastBlockInfo); err != nil {
		return errors.Wrap(err, "failed to convert order")
	}
	env.ChooseTakeString(info.rideV5Activated)
	env.ChooseMaxDataEntriesSize(info.rideV5Activated)
	env.SetLimit(ride.MaxVerifierComplexity(info.rideV5Activated))
	if err = env.SetTransactionFromOrder(order, tree.LibVersion); err != nil {
		return errors.Wrap(err, "failed to convert order")
	}
	r, err := ride.CallVerifier(env, tree)
	if err != nil {
		return errors.Wrapf(err, "account script on order '%s' thrown error with message", base58.Encode(id))
	}
	if !r.Result() {
		return errors.Errorf("account script on order '%s' returned false result", base58.Encode(id))
	}
	// Increase complexity.
	if info.rideV5Activated { // After activation of RideV5
		a.recentTxComplexity += uint64(r.Complexity())
	} else {
		// For account script we use original estimation
		est, scErr := a.stor.scriptsComplexity.newestScriptComplexityByAddr(senderWavesAddr)
		if scErr != nil {
			return errors.Wrapf(scErr, "failed to call account script on order '%s'", base58.Encode(id))
		}
		a.recentTxComplexity += uint64(est.Verifier)
	}
	return nil
}

// callAccountScriptWithTx calls account script. This method must not be called for proto.EthereumAddress.
func (a *scriptCaller) callAccountScriptWithTx(tx proto.Transaction, params *appendTxParams) error {
	senderAddr, err := tx.GetSender(a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	senderWavesAddr, ok := senderAddr.(proto.WavesAddress)
	if !ok {
		return errors.Errorf("address %q must be a waves address, not %T", senderAddr.String(), senderAddr)
	}
	tree, err := a.stor.scriptsStorage.newestScriptByAddr(senderWavesAddr)
	if err != nil {
		return err
	}
	id, err := tx.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	env, err := ride.NewEnvironment(
		a.settings.AddressSchemeCharacter,
		a.state,
		a.settings.InternalInvokePaymentsValidationAfterHeight,
		a.settings.PaymentsFixAfterHeight,
		params.blockV5Activated,
		params.rideV6Activated,
		params.consensusImprovementsActivated,
		params.blockRewardDistributionActivated,
		params.lightNodeActivated,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	env.ChooseSizeCheck(tree.LibVersion)
	env.ChooseTakeString(params.rideV5Activated)
	env.ChooseMaxDataEntriesSize(params.rideV5Activated)
	env.SetThisFromAddress(senderWavesAddr)
	if err := env.SetLastBlockFromBlockInfo(params.blockInfo); err != nil {
		return errors.Wrapf(err, "failed to call account scritp on transaction '%s'", base58.Encode(id))
	}
	env.SetLimit(ride.MaxVerifierComplexity(params.rideV5Activated))
	if err := env.SetTransaction(tx); err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	r, err := ride.CallVerifier(env, tree)
	if err != nil {
		return errors.Wrapf(err, "account script on transaction '%s' failed with error", base58.Encode(id))
	}
	if !r.Result() {
		return errs.NewTransactionNotAllowedByScript("script failed", id)
	}
	// Increase complexity.
	if params.rideV5Activated { // After activation of RideV5 add actual complexity
		a.recentTxComplexity += uint64(r.Complexity())
	} else {
		// For account script we use original estimation
		est, scErr := a.stor.scriptsComplexity.newestScriptComplexityByAddr(senderWavesAddr)
		if scErr != nil {
			return errors.Wrapf(scErr, "failed to call account script on transaction '%s'", base58.Encode(id))
		}
		a.recentTxComplexity += uint64(est.Verifier)
	}
	return nil
}

func (a *scriptCaller) callAssetScriptCommon(env *ride.EvaluationEnvironment, setTx func(*ride.EvaluationEnvironment) error, assetID crypto.Digest, params *appendTxParams) (ride.Result, error) {
	tree, err := a.stor.scriptsStorage.newestScriptByAsset(proto.AssetIDFromDigest(assetID))
	if err != nil {
		return nil, err
	}
	env.ChooseSizeCheck(tree.LibVersion)
	env.ChooseTakeString(params.rideV5Activated)
	env.ChooseMaxDataEntriesSize(params.rideV5Activated)
	env.SetLimit(ride.MaxAssetVerifierComplexity(tree.LibVersion))

	// Set transaction only after library version is set by `env.ChooseSizeCheck`
	if err = setTx(env); err != nil {
		return nil, err
	}

	switch tree.LibVersion {
	case ast.LibV1, ast.LibV2, ast.LibV3:
		assetInfo, err := a.state.NewestAssetInfo(assetID)
		if err != nil {
			return nil, err
		}
		env.SetThisFromAssetInfo(assetInfo)
	default:
		assetInfo, err := a.state.NewestFullAssetInfo(assetID)
		if err != nil {
			return nil, err
		}
		env.SetThisFromFullAssetInfo(assetInfo)
	}
	if err := env.SetLastBlockFromBlockInfo(params.blockInfo); err != nil {
		return nil, err
	}
	r, err := ride.CallVerifier(env, tree)
	if err != nil {
		return nil, errs.NewTransactionNotAllowedByScript(fmt.Sprintf("asset script: %v", err), assetID.Bytes())
	}
	if !r.Result() && !params.acceptFailed {
		return nil, errs.NewTransactionNotAllowedByScript("", assetID.Bytes())
	}
	// Increase complexity.
	if params.rideV5Activated { // After activation of RideV5 add actual execution complexity
		a.recentTxComplexity += uint64(r.Complexity())
	} else {
		// For asset script we use original estimation
		est, err := a.stor.scriptsComplexity.newestScriptComplexityByAsset(
			proto.AssetIDFromDigest(assetID),
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to call script on asset '%s'", assetID.String())
		}
		a.recentTxComplexity += uint64(est.Verifier)
	}
	return r, nil
}

func (a *scriptCaller) callAssetScriptWithScriptTransfer(tr *proto.FullScriptTransfer, assetID crypto.Digest, params *appendTxParams) (ride.Result, error) {
	env, err := ride.NewEnvironment(
		a.settings.AddressSchemeCharacter,
		a.state,
		a.settings.InternalInvokePaymentsValidationAfterHeight,
		a.settings.PaymentsFixAfterHeight,
		params.blockV5Activated,
		params.rideV6Activated,
		params.consensusImprovementsActivated,
		params.blockRewardDistributionActivated,
		params.lightNodeActivated,
	)
	if err != nil {
		return nil, err
	}
	setTx := func(env *ride.EvaluationEnvironment) error {
		env.SetTransactionFromScriptTransfer(tr)
		return nil
	}
	return a.callAssetScriptCommon(env, setTx, assetID, params)
}

func (a *scriptCaller) callAssetScript(tx proto.Transaction, assetID crypto.Digest, params *appendTxParams) (ride.Result, error) {
	env, err := ride.NewEnvironment(
		a.settings.AddressSchemeCharacter,
		a.state,
		a.settings.InternalInvokePaymentsValidationAfterHeight,
		a.settings.PaymentsFixAfterHeight,
		params.blockV5Activated,
		params.rideV6Activated,
		params.consensusImprovementsActivated,
		params.blockRewardDistributionActivated,
		params.lightNodeActivated,
	)
	if err != nil {
		return nil, err
	}

	setTx := func(env *ride.EvaluationEnvironment) error {
		return env.SetTransactionWithoutProofs(tx)
	}
	return a.callAssetScriptCommon(env, setTx, assetID, params)
}

func (a *scriptCaller) invokeFunction(
	tree *ast.Tree,
	scriptEstimationUpdate *scriptEstimation, // can be nil
	tx proto.Transaction,
	info *fallibleValidationParams,
	scriptAddress proto.WavesAddress,
) (ride.Result, error) {
	env, err := ride.NewEnvironment(
		a.settings.AddressSchemeCharacter,
		a.state,
		a.settings.InternalInvokePaymentsValidationAfterHeight,
		a.settings.PaymentsFixAfterHeight,
		info.blockV5Activated,
		info.rideV6Activated,
		info.consensusImprovementsActivated,
		info.blockRewardDistributionActivated,
		info.lightNodeActivated,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create RIDE environment")
	}
	env.SetThisFromAddress(scriptAddress)
	env.ChooseSizeCheck(tree.LibVersion)
	if err := env.SetLastBlockFromBlockInfo(info.blockInfo); err != nil {
		return nil, errors.Wrap(err, "failed to create RIDE environment")
	}
	env.SetTimestamp(tx.GetTimestamp())
	env.ChooseTakeString(info.rideV5Activated)
	env.ChooseMaxDataEntriesSize(info.rideV5Activated)
	limit, err := ride.MaxChainInvokeComplexityByVersion(tree.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set limit for invoke")
	}
	env.SetLimit(limit)

	err = env.SetTransaction(tx)
	if err != nil {
		return nil, err
	}

	senderAddr, err := tx.GetSender(a.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, err
	}
	sender, err := senderAddr.ToWavesAddress(a.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, err
	}
	r, functionCall, err := a.doTxInvoke(tx, sender, scriptAddress, env, tree, scriptEstimationUpdate, info)
	if err != nil {
		return nil, err
	}

	err = a.appendFunctionComplexity(r.Complexity(), scriptAddress, scriptEstimationUpdate, functionCall, info)
	return r, err
}

type workaroundResult struct {
	r       ride.Result
	fc      proto.FunctionCall
	rideErr error
}

func workaroundHandler(scheme proto.Scheme, tx proto.Transaction) (workaroundResult, bool, error) {
	// TODO: remove this workaround after the issue will be fixed
	switch scheme {
	case proto.MainNetScheme:
		txID, err := tx.GetID(scheme)
		if err != nil {
			return workaroundResult{}, false, errors.Wrap(err, "failed to calculate tx ID in workaround handler")
		}
		switch txIDStr := base58.Encode(txID); txIDStr {
		case "DGXQ69rv3PbwVa6TeT7AQy2pyhpeCawNwGdWcC3wdfVh",
			"GqKtPzT4judzqSPxtLpzeoZpZcdULeW2rGtGLYADoqmj",
			"DAcwgX2UkJ1zWYPE3ABrQQtRGamWDdJwCherhoVzkvJP":
			const txSpentComplexity = 16154
			rideErr := ride.EvaluationErrorSetComplexity( // set spent complexity
				ride.RuntimeError.Errorf("workaround for tx %q", txIDStr), // in scala - failed tx, go - ok
				txSpentComplexity,
			)
			res := workaroundResult{
				r:       nil,
				fc:      proto.FunctionCall{}, // default function call
				rideErr: rideErr,
			}
			return res, true, nil
		default:
			return workaroundResult{}, false, nil
		}
	case proto.TestNetScheme, proto.StageNetScheme:
		fallthrough
	default:
		return workaroundResult{}, false, nil
	}
}

func (a *scriptCaller) doTxInvoke(
	tx proto.Transaction,
	sender proto.WavesAddress,
	scriptAddress proto.WavesAddress,
	env *ride.EvaluationEnvironment,
	tree *ast.Tree,
	scriptEstimationUpdate *scriptEstimation,
	info *fallibleValidationParams,
) (ride.Result, proto.FunctionCall, error) {
	wr, ok, whErr := workaroundHandler(a.settings.AddressSchemeCharacter, tx)
	if whErr != nil {
		return nil, proto.FunctionCall{}, errors.Wrap(whErr, "failed to handle workaround invoke tx")
	}
	if ok {
		complexity := ride.EvaluationErrorSpentComplexity(wr.rideErr)
		appendErr := a.appendFunctionComplexity(complexity, scriptAddress, scriptEstimationUpdate, wr.fc, info)
		if appendErr != nil {
			return nil, proto.FunctionCall{}, errors.Wrap(appendErr,
				"failed to append function complexity for workaround invoke tx",
			)
		}
		return wr.r, wr.fc, wr.rideErr
	}
	switch transaction := tx.(type) {
	case *proto.InvokeScriptWithProofs:
		r, functionCall, err := a.invokeFunctionByInvokeWithProofsTx(transaction, sender, scriptAddress,
			env, tree, scriptEstimationUpdate, info,
		)
		if err != nil {
			return nil, functionCall, err
		}
		return r, functionCall, nil
	case *proto.InvokeExpressionTransactionWithProofs:
		// don't initialize function call because invoke expression tx can call only default function
		var functionCall proto.FunctionCall
		r, err := a.invokeFunctionByInvokeExpressionWithProofsTx(transaction, sender, scriptAddress,
			env, tree, scriptEstimationUpdate, info,
		)
		if err != nil {
			return nil, functionCall, err
		}
		return r, functionCall, nil
	case *proto.EthereumTransaction:
		r, functionCall, err := a.invokeFunctionByEthereumTx(transaction, sender, scriptAddress,
			env, tree, scriptEstimationUpdate, info,
		)
		if err != nil {
			return nil, functionCall, err
		}
		return r, functionCall, nil
	default:
		return nil, proto.FunctionCall{},
			errors.Errorf("failed to invoke function: unexpected type of transaction (%T)", transaction)
	}
}

func (a *scriptCaller) invokeFunctionByInvokeWithProofsTx(
	tx *proto.InvokeScriptWithProofs,
	sender proto.WavesAddress,
	scriptAddress proto.WavesAddress,
	env *ride.EvaluationEnvironment,
	tree *ast.Tree,
	scriptEstimationUpdate *scriptEstimation,
	info *fallibleValidationParams,
) (ride.Result, proto.FunctionCall, error) {
	err := env.SetInvoke(tx, tree.LibVersion)
	if err != nil {
		return nil, proto.FunctionCall{}, err
	}

	// Since V5 we have to create environment with wrapped state to which we put attached payments
	if tree.LibVersion >= ast.LibV5 {
		isPbTx := proto.IsProtobufTx(tx)
		env, err = ride.NewEnvironmentWithWrappedState(env, a.state, tx.Payments, sender, isPbTx, tree.LibVersion, true)
		if err != nil {
			return nil, proto.FunctionCall{}, errors.Wrapf(err, "failed to create RIDE environment with wrapped state")
		}
	}

	functionCall := tx.FunctionCall

	r, err := ride.CallFunction(env, tree, functionCall)
	if err != nil {
		complexity := ride.EvaluationErrorSpentComplexity(err)
		appendErr := a.appendFunctionComplexity(complexity, scriptAddress, scriptEstimationUpdate, functionCall, info)
		if appendErr != nil {
			return nil, proto.FunctionCall{}, appendErr
		}
		return nil, proto.FunctionCall{}, err
	}
	return r, functionCall, nil
}

func (a *scriptCaller) invokeFunctionByEthereumTx(
	tx *proto.EthereumTransaction,
	sender proto.WavesAddress,
	scriptAddress proto.WavesAddress,
	env *ride.EvaluationEnvironment,
	tree *ast.Tree,
	scriptEstimationUpdate *scriptEstimation,
	info *fallibleValidationParams,
) (ride.Result, proto.FunctionCall, error) {
	abiPayments := tx.TxKind.DecodedData().Payments
	scriptPayments := make([]proto.ScriptPayment, 0, len(abiPayments))
	for _, p := range abiPayments {
		if p.Amount <= 0 && info.checkerInfo.blockchainHeight > a.settings.InvokeNoZeroPaymentsAfterHeight {
			return nil, proto.FunctionCall{}, errors.Errorf("invalid payment amount '%d'", p.Amount)
		}
		optAsset := proto.NewOptionalAsset(p.PresentAssetID, p.AssetID)
		scriptPayment := proto.ScriptPayment{Amount: uint64(p.Amount), Asset: optAsset}
		scriptPayments = append(scriptPayments, scriptPayment)
	}

	err := env.SetEthereumInvoke(tx, tree.LibVersion, scriptPayments)
	if err != nil {
		return nil, proto.FunctionCall{}, err
	}
	// Since V5 we have to create environment with wrapped state to which we put attached payments
	if tree.LibVersion >= ast.LibV5 {
		const checkSenderBalance = false // skip initial payments validation for eth tx, see PR #965 for more info
		//TODO: Update last argument of the followinxg call with new feature activation flag or
		// something else depending on NODE-2531 issue resolution in scala implementation.
		isPbTx := proto.IsProtobufTx(tx)
		env, err = ride.NewEnvironmentWithWrappedState(env, a.state, scriptPayments, sender,
			isPbTx, tree.LibVersion, checkSenderBalance,
		)
		if err != nil {
			return nil, proto.FunctionCall{}, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
		}
	}

	decodedData := tx.TxKind.DecodedData()
	arguments, err := proto.ConvertDecodedEthereumArgumentsToProtoArguments(decodedData.Inputs)
	if err != nil {
		return nil, proto.FunctionCall{}, errors.Wrap(err, "failed to convert ethereum arguments")
	}
	functionCall := proto.NewFunctionCall(decodedData.Name, arguments)

	r, err := ride.CallFunction(env, tree, functionCall)
	if err != nil {
		complexity := ride.EvaluationErrorSpentComplexity(err)
		appendErr := a.appendFunctionComplexity(complexity, scriptAddress, scriptEstimationUpdate, functionCall, info)
		if appendErr != nil {
			return nil, proto.FunctionCall{}, appendErr
		}
		return nil, proto.FunctionCall{}, err
	}
	return r, functionCall, nil
}

func (a *scriptCaller) invokeFunctionByInvokeExpressionWithProofsTx(
	tx *proto.InvokeExpressionTransactionWithProofs,
	sender proto.WavesAddress,
	scriptAddress proto.WavesAddress,
	env *ride.EvaluationEnvironment,
	tree *ast.Tree,
	scriptEstimationUpdate *scriptEstimation,
	info *fallibleValidationParams,
) (ride.Result, error) {
	err := env.SetInvoke(tx, tree.LibVersion)
	if err != nil {
		return nil, err
	}
	var (
		functionCall proto.FunctionCall   // default function call
		payments     proto.ScriptPayments // payments aren't available for invoke expression transaction
	)
	// Since V5 we have to create environment with wrapped state to which we put attached payments
	if tree.LibVersion >= ast.LibV5 {
		isPbTx := proto.IsProtobufTx(tx)
		env, err = ride.NewEnvironmentWithWrappedState(env, a.state, payments, sender, isPbTx, tree.LibVersion, true)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create RIDE environment with wrapped state")
		}
	}

	r, err := ride.CallVerifier(env, tree)
	if err != nil {
		complexity := ride.EvaluationErrorSpentComplexity(err)
		appendErr := a.appendFunctionComplexity(complexity, scriptAddress, scriptEstimationUpdate, functionCall, info)
		if appendErr != nil {
			return nil, appendErr
		}
		return nil, err
	}
	return r, nil
}

func (a *scriptCaller) appendFunctionComplexity(
	evaluationComplexity int,
	scriptAddress proto.WavesAddress,
	scriptEstimationUpdate *scriptEstimation, // can be nil
	fc proto.FunctionCall,
	info *fallibleValidationParams,
) error {
	// Increase recent complexity
	if info.rideV5Activated {
		// After activation of RideV5 we have to add actual execution complexity
		a.recentTxComplexity += uint64(evaluationComplexity)
	} else {
		// Estimation based on estimated complexity
		// For callable (function) we have to use the latest possible estimation
		var est *ride.TreeEstimation
		if se := scriptEstimationUpdate; se.isPresent() { // newest estimation update made by last estimator
			est = &se.estimation
		} else { // the estimation
			r, err := a.stor.scriptsComplexity.newestScriptEstimationRecordByAddr(scriptAddress)
			if err != nil {
				return errors.Wrapf(err, "failed to get newest script complexity for script %q", scriptAddress)
			}
			est = &r.Estimation
		}
		functionName := fc.Name()
		c, ok := est.Functions[functionName]
		if !ok {
			return errors.Errorf("no estimation for function '%s'", functionName)
		}
		a.recentTxComplexity += uint64(c)
	}
	return nil
}

func (a *scriptCaller) getTotalComplexity() uint64 {
	return a.totalComplexity + a.recentTxComplexity
}

func (a *scriptCaller) resetRecentTxComplexity() {
	a.recentTxComplexity = 0
}

func (a *scriptCaller) addRecentTxComplexity() {
	a.totalComplexity += a.recentTxComplexity
	a.recentTxComplexity = 0
}

func (a *scriptCaller) resetComplexity() {
	a.totalComplexity = 0
	a.recentTxComplexity = 0
}
