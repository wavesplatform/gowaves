package state

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type scriptCaller struct {
	state types.SmartState

	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings

	totalComplexity uint64
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

func (a *scriptCaller) callAccountScriptWithOrder(order proto.Order, lastBlockInfo *proto.BlockInfo, initialisation, acceptFailed bool) (bool, error) {
	sender, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, order.GetSenderPK())
	if err != nil {
		return false, err
	}
	id, err := order.GetID()
	if err != nil {
		return false, err
	}
	script, err := a.stor.scriptsStorage.newestScriptByAddr(sender, !initialisation)
	if err != nil {
		return false, errors.Wrap(err, "failed to retrieve account script")
	}
	obj, err := ast.NewVariablesFromOrder(a.settings.AddressSchemeCharacter, order)
	if err != nil {
		return false, errors.Wrap(err, "failed to convert order")
	}
	this := ast.NewAddressFromProtoAddress(sender)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	r, err := a.callVerifyScript(script, obj, this, lastBlock)
	if err != nil {
		return false, errors.Wrapf(err, "account script failed on order '%s'", base58.Encode(id))
	}
	if !r.OK && !acceptFailed {
		return false, errors.Errorf("account script failed on order '%s' with error: %s", base58.Encode(id), r.Message)
	}
	// Increase complexity.
	complexity, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(sender, !initialisation)
	if err != nil {
		return false, errors.Wrap(err, "newestScriptComplexityByAddr")
	}
	a.totalComplexity += complexity.verifierComplexity
	return r.OK, nil
}

func (a *scriptCaller) callAccountScriptWithTx(tx proto.Transaction, lastBlockInfo *proto.BlockInfo, initialisation, acceptFailed bool) (bool, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return false, err
	}
	script, err := a.stor.scriptsStorage.newestScriptByAddr(senderAddr, !initialisation)
	if err != nil {
		return false, err
	}
	obj, err := ast.NewVariablesFromTransaction(a.settings.AddressSchemeCharacter, tx)
	if err != nil {
		return false, err
	}
	this := ast.NewAddressFromProtoAddress(senderAddr)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	r, err := a.callVerifyScript(script, obj, this, lastBlock)
	if err != nil {
		id, _ := tx.GetID(a.settings.AddressSchemeCharacter)
		return false, errors.Wrapf(err, "account script failed on transaction '%s'", base58.Encode(id))
	}
	if !r.OK && !acceptFailed {
		id, _ := tx.GetID(a.settings.AddressSchemeCharacter)
		return false, errors.Errorf("account script failed on transaction '%s' with error: %s", base58.Encode(id), r.Message)
	}
	// Increase complexity.
	complexity, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(senderAddr, !initialisation)
	if err != nil {
		return false, err
	}
	a.totalComplexity += complexity.verifierComplexity
	return r.OK, nil
}

func (a *scriptCaller) callAssetScriptCommon(obj map[string]ast.Expr, assetID crypto.Digest, lastBlockInfo *proto.BlockInfo, initialisation, acceptFailed bool) (bool, error) {
	script, err := a.stor.scriptsStorage.newestScriptByAsset(assetID, !initialisation)
	if err != nil {
		return false, err
	}
	var this ast.Expr
	switch script.Version {
	case 4:
		assetInfo, err := a.state.NewestFullAssetInfo(assetID)
		if err != nil {
			return false, err
		}
		this = ast.NewObjectFromAssetInfoV4(*assetInfo)
	default:
		assetInfo, err := a.state.NewestAssetInfo(assetID)
		if err != nil {
			return false, err
		}
		this = ast.NewObjectFromAssetInfoV3(*assetInfo)
	}
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	r, err := a.callVerifyScript(script, obj, this, lastBlock)
	if err != nil {
		return false, errors.Wrapf(err, "script failure on asset '%s'", assetID.String())
	}
	if !r.OK && !acceptFailed {
		return false, errors.Errorf("script failure on asset '%s' with error: %s", assetID.String(), r.Message)
	}
	// Increase complexity.
	complexityRecord, err := a.stor.scriptsComplexity.newestScriptComplexityByAsset(assetID, !initialisation)
	if err != nil {
		return false, err
	}
	a.totalComplexity += complexityRecord.complexity
	return r.OK, nil
}

func (a *scriptCaller) callAssetScriptWithScriptTransfer(tr *proto.FullScriptTransfer, assetID crypto.Digest, lastBlockInfo *proto.BlockInfo, initialisation, acceptFailed bool) (bool, error) {
	obj, err := ast.NewVariablesFromScriptTransfer(tr)
	if err != nil {
		return false, errors.Wrap(err, "failed to convert transaction")
	}
	return a.callAssetScriptCommon(obj, assetID, lastBlockInfo, initialisation, acceptFailed)
}

func (a *scriptCaller) callAssetScript(tx proto.Transaction, assetID crypto.Digest, lastBlockInfo *proto.BlockInfo, initialisation, acceptFailed bool) (bool, error) {
	obj, err := ast.NewVariablesFromTransaction(a.settings.AddressSchemeCharacter, tx)
	obj["proofs"] = ast.NewUnit() // Proofs are not accessible from asset's script
	if err != nil {
		return false, errors.Wrap(err, "failed to convert transaction")
	}
	return a.callAssetScriptCommon(obj, assetID, lastBlockInfo, initialisation, acceptFailed)
}

func (a *scriptCaller) invokeFunction(script ast.Script, tx *proto.InvokeScriptWithProofs, lastBlockInfo *proto.BlockInfo, scriptAddress proto.Address, initialisation bool) ([]proto.ScriptAction, error) {
	this := ast.NewAddressFromProtoAddress(scriptAddress)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	actions, err := script.CallFunction(a.settings.AddressSchemeCharacter, a.state, tx, this, lastBlock)
	if err != nil {
		return nil, errors.Wrapf(err, "transaction ID %s", tx.ID.String())
	}
	// Increase complexity.
	complexityRecord, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(scriptAddress, !initialisation)
	if err != nil {
		return nil, errors.Wrap(err, "newestScriptComplexityByAsset()")
	}
	// TODO: check this!
	a.totalComplexity += complexityRecord.byFuncs[tx.FunctionCall.Name]
	return actions, nil
}

func (a *scriptCaller) getTotalComplexity() uint64 {
	return a.totalComplexity
}

func (a *scriptCaller) resetComplexity() {
	a.totalComplexity = 0
}
