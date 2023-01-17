package state

import (
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
	state types.SmartState

	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings

	totalComplexity    uint64
	recentTxComplexity uint64
}

func newScriptCaller(
	state types.SmartState,
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
		info.blockV5Activated,
		info.rideV6Activated,
		info.consensusImprovementsActivated,
		info.invokeExpressionActivated,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create RIDE environment")
	}
	env.SetThisFromAddress(senderWavesAddr)
	env.SetLastBlock(lastBlockInfo)
	env.ChooseSizeCheck(tree.LibVersion)
	env.ChooseTakeString(info.rideV5Activated)
	env.ChooseMaxDataEntriesSize(info.rideV5Activated)
	env.SetLimit(ride.MaxVerifierComplexity(info.rideV5Activated))
	err = env.SetTransactionFromOrder(order)
	if err != nil {
		return errors.Wrap(err, "failed to convert order")
	}
	r, err := ride.CallVerifier(env, tree)
	if err != nil {
		return errors.Errorf("account script on order '%s' thrown error with message: %s", base58.Encode(id), err.Error())
	}
	if !r.Result() {
		return errors.Errorf("account script on order '%s' returned false result", base58.Encode(id))
	}
	// Increase complexity.
	if info.rideV5Activated { // After activation of RideV5
		a.recentTxComplexity += uint64(r.Complexity())
	} else {
		// For account script we use original estimation
		est, err := a.stor.scriptsComplexity.newestOriginalScriptComplexityByAddr(senderWavesAddr)
		if err != nil {
			return errors.Wrapf(err, "failed to call account script on order '%s'", base58.Encode(id))
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
		params.blockV5Activated,
		params.rideV6Activated,
		params.consensusImprovementsActivated,
		params.invokeExpressionActivated,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	env.ChooseSizeCheck(tree.LibVersion)
	env.ChooseTakeString(params.rideV5Activated)
	env.ChooseMaxDataEntriesSize(params.rideV5Activated)
	env.SetThisFromAddress(senderWavesAddr)
	env.SetLastBlock(params.blockInfo)
	env.SetLimit(ride.MaxVerifierComplexity(params.rideV5Activated))
	err = env.SetTransaction(tx)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	r, err := ride.CallVerifier(env, tree)
	if err != nil {
		return errors.Errorf("account script on transaction '%s' failed with error: %v", base58.Encode(id), err.Error())
	}
	if !r.Result() {
		return errs.NewTransactionNotAllowedByScript("script failed", id)
	}
	// Increase complexity.
	if params.rideV5Activated { // After activation of RideV5 add actual complexity
		a.recentTxComplexity += uint64(r.Complexity())
	} else {
		// For account script we use original estimation
		est, err := a.stor.scriptsComplexity.newestOriginalScriptComplexityByAddr(senderWavesAddr)
		if err != nil {
			return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
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
	env.SetLastBlock(params.blockInfo)
	r, err := ride.CallVerifier(env, tree)
	if err != nil {
		return nil, errs.NewTransactionNotAllowedByScript(err.Error(), assetID.Bytes())
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
		params.blockV5Activated,
		params.rideV6Activated,
		params.consensusImprovementsActivated,
		params.invokeExpressionActivated,
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
		params.blockV5Activated,
		params.rideV6Activated,
		params.consensusImprovementsActivated,
		params.invokeExpressionActivated,
	)
	if err != nil {
		return nil, err
	}

	setTx := func(env *ride.EvaluationEnvironment) error {
		return env.SetTransactionWithoutProofs(tx)
	}
	return a.callAssetScriptCommon(env, setTx, assetID, params)
}

func (a *scriptCaller) invokeFunction(tree *ast.Tree, tx proto.Transaction, info *fallibleValidationParams, scriptAddress proto.WavesAddress) (ride.Result, error) {
	env, err := ride.NewEnvironment(
		a.settings.AddressSchemeCharacter,
		a.state,
		a.settings.InternalInvokePaymentsValidationAfterHeight,
		info.blockV5Activated,
		info.rideV6Activated,
		info.consensusImprovementsActivated,
		info.invokeExpressionActivated,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create RIDE environment")
	}
	env.SetThisFromAddress(scriptAddress)
	env.SetLastBlock(info.blockInfo)
	env.SetTimestamp(tx.GetTimestamp())
	env.ChooseSizeCheck(tree.LibVersion)
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

	var (
		functionName      string
		functionArguments proto.Arguments
		defaultFunction   bool
		payments          proto.ScriptPayments
		sender            proto.WavesAddress
		r                 ride.Result
	)
	switch transaction := tx.(type) {
	case *proto.InvokeScriptWithProofs:
		err = env.SetInvoke(tx, tree.LibVersion)
		if err != nil {
			return nil, err
		}
		payments = transaction.Payments
		sender, err = proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, transaction.SenderPK)
		if err != nil {
			return nil, err
		}
		functionName = transaction.FunctionCall.Name
		functionArguments = transaction.FunctionCall.Arguments
		defaultFunction = transaction.FunctionCall.Default

		// Since V5 we have to create environment with wrapped state to which we put attached payments
		if tree.LibVersion >= ast.LibV5 {
			env, err = ride.NewEnvironmentWithWrappedState(env, payments, sender, proto.IsProtobufTx(tx), tree.LibVersion, true)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create RIDE environment with wrapped state")
			}
		}

		r, err = ride.CallFunction(env, tree, functionName, functionArguments)
		if err != nil {
			if appendErr := a.appendFunctionComplexity(ride.EvaluationErrorSpentComplexity(err), scriptAddress, functionName, defaultFunction, info); appendErr != nil {
				return nil, appendErr
			}
			return nil, err
		}
	case *proto.InvokeExpressionTransactionWithProofs:
		err = env.SetInvoke(tx, tree.LibVersion)
		if err != nil {
			return nil, err
		}
		sender, err = proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, transaction.SenderPK)
		if err != nil {
			return nil, err
		}
		functionName = ""

		// Since V5 we have to create environment with wrapped state to which we put attached payments
		if tree.LibVersion >= ast.LibV5 {
			env, err = ride.NewEnvironmentWithWrappedState(env, payments, sender, proto.IsProtobufTx(tx), tree.LibVersion, true)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create RIDE environment with wrapped state")
			}
		}

		r, err = ride.CallVerifier(env, tree)
		if err != nil {
			if appendErr := a.appendFunctionComplexity(ride.EvaluationErrorSpentComplexity(err), scriptAddress, functionName, defaultFunction, info); appendErr != nil {
				return nil, appendErr
			}
			return nil, err
		}

	case *proto.EthereumTransaction:
		abiPayments := transaction.TxKind.DecodedData().Payments
		scriptPayments := make([]proto.ScriptPayment, 0, len(abiPayments))
		for _, p := range abiPayments {
			if p.Amount <= 0 && info.checkerInfo.height > a.settings.InvokeNoZeroPaymentsAfterHeight {
				return nil, errors.Errorf("invalid payment amount '%d'", p.Amount)
			}
			optAsset := proto.NewOptionalAsset(p.PresentAssetID, p.AssetID)
			scriptPayment := proto.ScriptPayment{Amount: uint64(p.Amount), Asset: optAsset}
			scriptPayments = append(scriptPayments, scriptPayment)
		}
		payments = scriptPayments

		err = env.SetEthereumInvoke(transaction, tree.LibVersion, scriptPayments)
		if err != nil {
			return nil, err
		}
		sender, err = transaction.WavesAddressFrom(a.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, errors.Errorf("failed to get waves address from ethereum transaction %v", err)
		}
		decodedData := transaction.TxKind.DecodedData()
		functionName = decodedData.Name
		arguments, err := ride.ConvertDecodedEthereumArgumentsToProtoArguments(decodedData.Inputs)
		if err != nil {
			return nil, errors.Errorf("failed to convert ethereum arguments, %v", err)
		}
		functionArguments = arguments
		defaultFunction = true
		// Since V5 we have to create environment with wrapped state to which we put attached payments
		if tree.LibVersion >= ast.LibV5 {
			//TODO: Update last argument of the followinxg call with new feature activation flag or
			// something else depending on NODE-2531 issue resolution in scala implementation.
			env, err = ride.NewEnvironmentWithWrappedState(env, payments, sender, proto.IsProtobufTx(tx), tree.LibVersion, false)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create RIDE environment with wrapped state")
			}
		}

		r, err = ride.CallFunction(env, tree, functionName, functionArguments)
		if err != nil {
			if appendErr := a.appendFunctionComplexity(ride.EvaluationErrorSpentComplexity(err), scriptAddress, functionName, defaultFunction, info); appendErr != nil {
				return nil, appendErr
			}
			return nil, err
		}

	default:
		return nil, errors.Errorf("failed to invoke function: unexpected type of transaction (%T)", transaction)
	}

	if err := a.appendFunctionComplexity(r.Complexity(), scriptAddress, functionName, defaultFunction, info); err != nil {
		return nil, err
	}
	return r, nil
}

func (a *scriptCaller) appendFunctionComplexity(evaluationComplexity int, scriptAddress proto.Address, functionName string, functionDefault bool, info *fallibleValidationParams) error {
	// Increase recent complexity
	if info.rideV5Activated {
		// After activation of RideV5 we have to add actual execution complexity
		a.recentTxComplexity += uint64(evaluationComplexity)
	} else {
		// Estimation based on estimated complexity
		// For callable (function) we have to use the latest possible estimation
		ev, err := a.state.EstimatorVersion()
		if err != nil {
			return err
		}
		est, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(scriptAddress, ev)
		if err != nil {
			return err
		}
		if functionName == "" && functionDefault {
			functionName = "default"
		}
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
