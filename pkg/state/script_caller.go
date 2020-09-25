package state

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
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

func (a *scriptCaller) callVerifyScript(script ast.Script, obj map[string]ast.Expr, this, lastBlock ast.Expr) (ast.Result, error) {
	return script.Verify(a.settings.AddressSchemeCharacter, a.state, obj, this, lastBlock)
}

func (a *scriptCaller) callAccountScriptWithOrder(order proto.Order, lastBlockInfo *proto.BlockInfo, initialisation bool) error {
	sender, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, order.GetSenderPK())
	if err != nil {
		return err
	}
	id, err := order.GetID()
	if err != nil {
		return err
	}
	script, err := a.stor.scriptsStorage.newestScriptByAddr(sender, !initialisation)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve account script")
	}
	obj, err := ast.NewVariablesFromOrder(a.settings.AddressSchemeCharacter, order)
	if err != nil {
		return errors.Wrap(err, "failed to convert order")
	}
	this := ast.NewAddressFromProtoAddress(sender)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	r, err := a.callVerifyScript(script, obj, this, lastBlock)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on order '%s'", base58.Encode(id))
	}
	if !r.Value {
		return errors.Errorf("account script on order '%s' returned false result", base58.Encode(id))
	}
	if r.Throw {
		return errors.Errorf("account script on order '%s' thrown error with message: %s", base58.Encode(id), r.Message)
	}
	// Increase complexity.
	ev, err := a.state.EstimatorVersion()
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on order '%s'", base58.Encode(id))
	}
	est, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(sender, ev, !initialisation)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on order '%s'", base58.Encode(id))
	}
	a.recentTxComplexity += uint64(est.Verifier)
	return nil
}

func (a *scriptCaller) callAccountScriptWithTx(tx proto.Transaction, lastBlockInfo *proto.BlockInfo, initialisation bool) error {
	senderAddr, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return err
	}
	script, err := a.stor.scriptsStorage.newestScriptByAddr(senderAddr, !initialisation)
	if err != nil {
		return err
	}
	id, err := tx.GetID(a.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}
	obj, err := ast.NewVariablesFromTransaction(a.settings.AddressSchemeCharacter, tx)
	if err != nil {
		return err
	}
	this := ast.NewAddressFromProtoAddress(senderAddr)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	r, err := a.callVerifyScript(script, obj, this, lastBlock)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	if r.Failed() {
		if !r.Value {
			return errs.NewTransactionNotAllowedByScript(r.Error().Error(), nil)
		}
		return errors.Errorf("account script on transaction '%s' failed with error: %v", base58.Encode(id), r.Error())
	}
	// Increase complexity.
	ev, err := a.state.EstimatorVersion()
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	est, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(senderAddr, ev, !initialisation)
	if err != nil {
		return errors.Wrapf(err, "failed to call account script on transaction '%s'", base58.Encode(id))
	}
	a.recentTxComplexity += uint64(est.Verifier)
	return nil
}

func (a *scriptCaller) callAssetScriptCommon(
	obj map[string]ast.Expr,
	assetID crypto.Digest,
	lastBlockInfo *proto.BlockInfo,
	initialisation bool,
	acceptFailed bool,
) (ast.Result, error) {
	script, err := a.stor.scriptsStorage.newestScriptByAsset(assetID, !initialisation)
	if err != nil {
		return ast.Result{}, err
	}
	var this ast.Expr
	switch script.Version {
	case 4:
		assetInfo, err := a.state.NewestFullAssetInfo(assetID)
		if err != nil {
			return ast.Result{}, err
		}
		this = ast.NewObjectFromAssetInfoV4(*assetInfo)
	default:
		assetInfo, err := a.state.NewestAssetInfo(assetID)
		if err != nil {
			return ast.Result{}, err
		}
		this = ast.NewObjectFromAssetInfoV3(*assetInfo)
	}
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	r, err := a.callVerifyScript(script, obj, this, lastBlock)
	if err != nil {
		return ast.Result{}, errors.Wrapf(err, "failed to call script on asset '%s'", assetID.String())
	}
	if r.Failed() && !acceptFailed {
		if r.Throw {
			return ast.Result{}, errors.Errorf("script failure on asset '%s' with error: %s", assetID.String(), r.Error())
		}
		if !r.Value {
			return ast.Result{}, errs.NewTransactionNotAllowedByScript(r.Error().Error(), assetID.Bytes())
		}
		return ast.Result{}, errors.Errorf("script failure on asset '%s' with error: %s", assetID.String(), r.Error())
	}
	// Increase complexity.
	ev, err := a.state.EstimatorVersion()
	if err != nil {
		return ast.Result{}, errors.Wrapf(err, "failed to call script on asset '%s'", assetID.String())
	}
	est, err := a.stor.scriptsComplexity.newestScriptComplexityByAsset(assetID, ev, !initialisation)
	if err != nil {
		return ast.Result{}, errors.Wrapf(err, "failed to call script on asset '%s'", assetID.String())
	}
	a.recentTxComplexity += uint64(est.Verifier)
	return r, nil
}

func (a *scriptCaller) callAssetScriptWithScriptTransfer(
	tr *proto.FullScriptTransfer,
	assetID crypto.Digest,
	lastBlockInfo *proto.BlockInfo,
	initialisation bool,
	acceptFailed bool,
) (ast.Result, error) {
	obj, err := ast.NewVariablesFromScriptTransfer(tr)
	if err != nil {
		return ast.Result{}, errors.Wrap(err, "failed to convert transaction")
	}
	return a.callAssetScriptCommon(obj, assetID, lastBlockInfo, initialisation, acceptFailed)
}

func (a *scriptCaller) callAssetScript(
	tx proto.Transaction,
	assetID crypto.Digest,
	lastBlockInfo *proto.BlockInfo,
	initialisation bool,
	acceptFailed bool,
) (ast.Result, error) {
	obj, err := ast.NewVariablesFromTransaction(a.settings.AddressSchemeCharacter, tx)
	obj["proofs"] = ast.NewUnit() // Proofs are not accessible from asset's script
	if err != nil {
		return ast.Result{}, errors.Wrap(err, "failed to convert transaction")
	}
	return a.callAssetScriptCommon(obj, assetID, lastBlockInfo, initialisation, acceptFailed)
}

func (a *scriptCaller) invokeFunction(script ast.Script, tx *proto.InvokeScriptWithProofs, lastBlockInfo *proto.BlockInfo, scriptAddress proto.Address, initialisation bool) (bool, []proto.ScriptAction, error) {
	this := ast.NewAddressFromProtoAddress(scriptAddress)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	_, actions, err := script.CallFunction(a.settings.AddressSchemeCharacter, a.state, tx, this, lastBlock)
	if err != nil {
		return false, nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
	}
	// Increase complexity.
	ev, err := a.state.EstimatorVersion()
	if err != nil {
		return false, nil, errors.Wrapf(err, "invocation of transaction '%s' failed", tx.ID.String())
	}
	est, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(scriptAddress, ev, !initialisation)
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
	return ok, actions, nil
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
