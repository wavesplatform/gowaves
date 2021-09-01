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
	if err := env.ChooseTakeString(isRideV5); err != nil {
		return errors.Wrap(err, "failed to initialize environment")
	}
	err = env.SetTransactionFromOrder(order)
	if err != nil {
		return errors.Wrap(err, "failed to convert order")
	}
	r, err := ride.CallVerifier(env, tree)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on order '%s'", base58.Encode(id))
	}
	if !r.Result() {
		if r.UserError() != "" {
			return errors.Errorf("account script on order '%s' thrown error with message: %s", base58.Encode(id), r.UserError())
		}
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
	if err := env.ChooseTakeString(params.rideV5Activated); err != nil {
		return errors.Wrap(err, "failed to initialize environment")
	}
	env.SetThisFromAddress(senderAddr)
	env.SetLastBlock(params.blockInfo)
	err = env.SetTransaction(tx)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	r, err := ride.CallVerifier(env, tree)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	if !r.Result() {
		if r.UserError() != "" {
			return errors.Errorf("account script on transaction '%s' failed with error: %v", base58.Encode(id), r.UserError())
		}
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
	if err := env.ChooseTakeString(params.rideV5Activated); err != nil {
		return nil, errors.Wrap(err, "failed to initialize environment")
	}
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
		return nil, errors.Wrapf(err, "failed to call script on asset '%s'", assetID.String())
	}
	if !r.Result() && !params.acceptFailed {
		return nil, errs.NewTransactionNotAllowedByScript(r.UserError(), assetID.Bytes())
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

func (a *scriptCaller) invokeFunction(tree *ride.Tree, tx *proto.InvokeScriptWithProofs, info *fallibleValidationParams, scriptAddress proto.Address) (bool, []proto.ScriptAction, error) {
	env, err := ride.NewEnvironment(a.settings.AddressSchemeCharacter, a.state, a.settings.InternalInvokePaymentsValidationAfterHeight)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to create RIDE environment")
	}
	env.SetThisFromAddress(scriptAddress)
	env.SetLastBlock(info.blockInfo)
	env.SetTimestamp(tx.Timestamp)
	err = env.SetTransaction(tx)
	if err != nil {
		return false, nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
	}
	err = env.SetInvoke(tx, tree.LibVersion)
	if err != nil {
		return false, nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
	}
	env.ChooseSizeCheck(tree.LibVersion)

	err = env.ChooseTakeString(info.rideV5Activated)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to choose takeString")
	}
	// Since V5 we have to create environment with wrapped state to which we put attached payments
	if tree.LibVersion >= 5 {
		env, err = ride.NewEnvironmentWithWrappedState(env, tx.Payments, tx.SenderPK)
		if err != nil {
			return false, nil, errors.Wrapf(err, "failed to create RIDE environment with wrapped state")
		}
	}

	r, err := ride.CallFunction(env, tree, tx.FunctionCall.Name, tx.FunctionCall.Arguments)
	if err != nil {
		return false, nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
	}
	if sr, ok := r.(ride.ScriptResult); ok {
		return false, nil, errors.Errorf("unexpected ScriptResult: %v", sr)
	}
	// Increase complexity.
	if info.rideV5Activated { // After activation of RideV5 add actual execution complexity
		a.recentTxComplexity += uint64(r.Complexity())
	} else {
		// For callable (function) we have to use latest possible estimation
		ev, err := a.state.EstimatorVersion()
		if err != nil {
			return false, nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
		}
		est, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(scriptAddress, ev, !info.initialisation)
		if err != nil {
			return false, nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
		}
		fn := tx.FunctionCall.Name
		if fn == "" && tx.FunctionCall.Default {
			fn = "default"
		}
		c, ok := est.Functions[fn]
		if !ok {
			return false, nil, errors.Errorf("no estimation for function '%s' on invocation of transaction '%s'", fn, tx.ID.String())
		}
		a.recentTxComplexity += uint64(c)
	}
	err = nil
	if !r.Result() { // Replace failure status with an error
		err = errors.Errorf("call failed: %s", r.UserError())
	}
	return true, r.ScriptActions(), err
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
