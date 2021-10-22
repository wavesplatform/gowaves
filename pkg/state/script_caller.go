package state

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
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

func (a *scriptCaller) callAccountScriptWithOrder(order proto.Order, lastBlockInfo *proto.BlockInfo, isRideV5 bool, initialisation bool) error {
	sender, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, order.GetSenderPK())
	if err != nil {
		return err
	}
	id, err := order.GetID()
	if err != nil {
		return err
	}
	tree, err := a.stor.scriptsStorage.newestScriptByAddr(sender, !initialisation)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve account script")
	}
	env, err := ride.NewEnvironment(a.settings.AddressSchemeCharacter, a.state, a.settings.InternalInvokePaymentsValidationAfterHeight)
	if err != nil {
		return errors.Wrap(err, "failed to create RIDE environment")
	}
	env.SetThisFromAddress(sender)
	env.SetLastBlock(lastBlockInfo)
	env.ChooseSizeCheck(tree.LibVersion)
	env.ChooseTakeString(isRideV5)
	env.ChooseMaxDataEntriesSize(isRideV5)
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
	if isRideV5 { // After activation of RideV5
		a.recentTxComplexity += uint64(r.Complexity())
	} else {
		// For account script we use original estimation
		est, err := a.stor.scriptsComplexity.newestOriginalScriptComplexityByAddr(sender, !initialisation)
		if err != nil {
			return errors.Wrapf(err, "failed to call account script on order '%s'", base58.Encode(id))
		}
		a.recentTxComplexity += uint64(est.Verifier)
	}
	return nil
}

func (a *scriptCaller) callAccountScriptWithTx(tx proto.Transaction, params *appendTxParams) error {
	senderAddr, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return err
	}
	tree, err := a.stor.scriptsStorage.newestScriptByAddr(senderAddr, !params.initialisation)
	if err != nil {
		return err
	}
	id, err := tx.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	env, err := ride.NewEnvironment(a.settings.AddressSchemeCharacter, a.state, a.settings.InternalInvokePaymentsValidationAfterHeight)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	env.ChooseSizeCheck(tree.LibVersion)
	env.ChooseTakeString(params.rideV5Activated)
	env.ChooseMaxDataEntriesSize(params.rideV5Activated)
	env.SetThisFromAddress(senderAddr)
	env.SetLastBlock(params.blockInfo)
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
		est, err := a.stor.scriptsComplexity.newestOriginalScriptComplexityByAddr(senderAddr, !params.initialisation)
		if err != nil {
			return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
		}
		a.recentTxComplexity += uint64(est.Verifier)
	}
	return nil
}

func (a *scriptCaller) callAssetScriptCommon(env *ride.EvaluationEnvironment, assetID crypto.Digest, params *appendTxParams) (ride.Result, error) {
	tree, err := a.stor.scriptsStorage.newestScriptByAsset(assetID, !params.initialisation)
	if err != nil {
		return nil, err
	}
	env.ChooseSizeCheck(tree.LibVersion)
	env.ChooseTakeString(params.rideV5Activated)
	env.ChooseMaxDataEntriesSize(params.rideV5Activated)
	switch tree.LibVersion {
	case 4, 5:
		assetInfo, err := a.state.NewestFullAssetInfo(assetID)
		if err != nil {
			return nil, err
		}
		env.SetThisFromFullAssetInfo(assetInfo)
	default:
		assetInfo, err := a.state.NewestAssetInfo(assetID)
		if err != nil {
			return nil, err
		}
		env.SetThisFromAssetInfo(assetInfo)
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
		est, err := a.stor.scriptsComplexity.newestScriptComplexityByAsset(assetID, !params.initialisation)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to call script on asset '%s'", assetID.String())
		}
		a.recentTxComplexity += uint64(est.Verifier)
	}
	return r, nil
}

func (a *scriptCaller) callAssetScriptWithScriptTransfer(tr *proto.FullScriptTransfer, assetID crypto.Digest, params *appendTxParams) (ride.Result, error) {
	env, err := ride.NewEnvironment(a.settings.AddressSchemeCharacter, a.state, a.settings.InternalInvokePaymentsValidationAfterHeight)
	if err != nil {
		return nil, err
	}
	env.SetTransactionFromScriptTransfer(tr)
	return a.callAssetScriptCommon(env, assetID, params)
}

func (a *scriptCaller) callAssetScript(tx proto.Transaction, assetID crypto.Digest, params *appendTxParams) (ride.Result, error) {
	env, err := ride.NewEnvironment(a.settings.AddressSchemeCharacter, a.state, a.settings.InternalInvokePaymentsValidationAfterHeight)
	if err != nil {
		return nil, err
	}
	err = env.SetTransactionWithoutProofs(tx)
	if err != nil {
		return nil, err
	}
	return a.callAssetScriptCommon(env, assetID, params)
}

func (a *scriptCaller) invokeFunction(tree *ride.Tree, tx *proto.InvokeScriptWithProofs, info *fallibleValidationParams, scriptAddress proto.Address) (ride.Result, error) {
	env, err := ride.NewEnvironment(a.settings.AddressSchemeCharacter, a.state, a.settings.InternalInvokePaymentsValidationAfterHeight)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create RIDE environment")
	}
	env.SetThisFromAddress(scriptAddress)
	env.SetLastBlock(info.blockInfo)
	env.SetTimestamp(tx.Timestamp)
	err = env.SetTransaction(tx)
	if err != nil {
		return nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
	}
	err = env.SetInvoke(tx, tree.LibVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
	}
	env.ChooseSizeCheck(tree.LibVersion)

	env.ChooseTakeString(info.rideV5Activated)
	env.ChooseMaxDataEntriesSize(info.rideV5Activated)

	// Since V5 we have to create environment with wrapped state to which we put attached payments
	if tree.LibVersion >= 5 {
		env, err = ride.NewEnvironmentWithWrappedState(env, tx.Payments, tx.SenderPK)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create RIDE environment with wrapped state")
		}
	}

	r, err := ride.CallFunction(env, tree, tx.FunctionCall.Name, tx.FunctionCall.Arguments)
	if err != nil {
		return nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
	}
	if err := a.appendFunctionComplexity(r, scriptAddress, tx.FunctionCall, info); err != nil {
		return nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
	}
	return r, nil
}

func (a *scriptCaller) appendFunctionComplexity(result ride.Result, scriptAddress proto.Address, function proto.FunctionCall, info *fallibleValidationParams) error {
	if result == nil {
		return nil
	}
	// Increase recent complexity
	if info.rideV5Activated {
		// After activation of RideV5 we have to add actual execution complexity
		a.recentTxComplexity += uint64(result.Complexity())
	} else {
		// Estimation based on evaluated complexity
		// For callable (function) we have to use the latest possible estimation
		ev, err := a.state.EstimatorVersion()
		if err != nil {
			return err
		}
		est, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(scriptAddress, ev, !info.initialisation)
		if err != nil {
			return err
		}
		fn := function.Name
		if fn == "" && function.Default {
			fn = "default"
		}
		c, ok := est.Functions[fn]
		if !ok {
			return errors.Errorf("no estimation for function '%s'", function.Name)
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
