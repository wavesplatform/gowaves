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

func (a *scriptCaller) callVerifyScript(script ast.Script, obj map[string]ast.Expr, this, lastBlock ast.Expr) error {
	ok, err := script.Verify(a.settings.AddressSchemeCharacter, a.state, obj, this, lastBlock)
	if err != nil {
		return errors.Wrap(err, "verifier script failed")
	}
	if !ok {
		return errors.New("verifier script does not allow to send transaction")
	}
	return nil
}

func (a *scriptCaller) callAccountScriptWithOrder(order proto.Order, lastBlockInfo *proto.BlockInfo, initialisation bool) error {
	sender, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, order.GetSenderPK())
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
	if err := a.callVerifyScript(script, obj, this, lastBlock); err != nil {
		id, _ := order.GetID()
		return errors.Errorf("account script; order ID %s: %v\n", base58.Encode(id), err)
	}
	// Increase complexity.
	complexity, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(sender, !initialisation)
	if err != nil {
		return errors.Wrap(err, "newestScriptComplexityByAddr")
	}
	a.totalComplexity += complexity.verifierComplexity
	return nil
}

func (a *scriptCaller) callAccountScriptWithTx(tx proto.Transaction, lastBlockInfo *proto.BlockInfo, initialisation bool) error {
	senderAddr, err := proto.NewAddressFromPublicKey(a.settings.AddressSchemeCharacter, tx.GetSenderPK())
	if err != nil {
		return err
	}
	script, err := a.stor.scriptsStorage.newestScriptByAddr(senderAddr, !initialisation)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve account script")
	}
	obj, err := ast.NewVariablesFromTransaction(a.settings.AddressSchemeCharacter, tx)
	if err != nil {
		return errors.Wrap(err, "failed to convert transaction")
	}
	this := ast.NewAddressFromProtoAddress(senderAddr)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	if err := a.callVerifyScript(script, obj, this, lastBlock); err != nil {
		id, _ := tx.GetID()
		return errors.Errorf("account script; transaction ID %s: %v\n", base58.Encode(id), err)
	}
	// Increase complexity.
	complexity, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(senderAddr, !initialisation)
	if err != nil {
		return errors.Wrap(err, "newestScriptComplexityByAddr")
	}
	a.totalComplexity += complexity.verifierComplexity
	return nil
}

func (a *scriptCaller) callAssetScriptCommon(obj map[string]ast.Expr, assetID crypto.Digest, lastBlockInfo *proto.BlockInfo, initialisation bool) error {
	script, err := a.stor.scriptsStorage.newestScriptByAsset(assetID, !initialisation)
	if err != nil {
		return errors.Errorf("failed to retrieve asset script: %v\n", err)
	}
	assetInfo, err := a.state.NewestAssetInfo(assetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve asset info")
	}
	this := ast.NewObjectFromAssetInfo(*assetInfo)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	if err := a.callVerifyScript(script, obj, this, lastBlock); err != nil {
		return errors.Wrap(err, "callVerifyScript failed")
	}
	// Increase complexity.
	complexityRecord, err := a.stor.scriptsComplexity.newestScriptComplexityByAsset(assetID, !initialisation)
	if err != nil {
		return errors.Wrap(err, "newestScriptComplexityByAsset()")
	}
	a.totalComplexity += complexityRecord.complexity
	return nil
}

func (a *scriptCaller) callAssetScriptWithScriptTransfer(tr *proto.FullScriptTransfer, assetID crypto.Digest, lastBlockInfo *proto.BlockInfo, initialisation bool) error {
	obj, err := ast.NewVariablesFromScriptTransfer(tr)
	if err != nil {
		return errors.Wrap(err, "failed to convert transaction")
	}
	if err := a.callAssetScriptCommon(obj, assetID, lastBlockInfo, initialisation); err != nil {
		return errors.Errorf("asset script; script transfer ID %s: %v\n", tr.ID.String(), err)
	}
	return nil
}

func (a *scriptCaller) callAssetScript(tx proto.Transaction, assetID crypto.Digest, lastBlockInfo *proto.BlockInfo, initialisation bool) error {
	obj, err := ast.NewVariablesFromTransaction(a.settings.AddressSchemeCharacter, tx)
	if err != nil {
		return errors.Wrap(err, "failed to convert transaction")
	}
	if err := a.callAssetScriptCommon(obj, assetID, lastBlockInfo, initialisation); err != nil {
		id, _ := tx.GetID()
		return errors.Errorf("asset script; transaction ID %s: %v\n", base58.Encode(id), err)
	}
	return nil
}

func (a *scriptCaller) invokeFunction(tx *proto.InvokeScriptV1, lastBlockInfo *proto.BlockInfo, initialisation bool) (*proto.ScriptResult, error) {
	scriptAddr, err := recipientToAddress(tx.ScriptRecipient, a.stor.aliases, !initialisation)
	if err != nil {
		return nil, err
	}
	script, err := a.stor.scriptsStorage.newestScriptByAddr(*scriptAddr, !initialisation)
	if err != nil {
		return nil, err
	}
	this := ast.NewAddressFromProtoAddress(*scriptAddr)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	sr, err := script.CallFunction(a.settings.AddressSchemeCharacter, a.state, tx, this, lastBlock)
	if err != nil {
		return nil, errors.Errorf("transaction ID %s: %v\n", tx.ID.String(), err)
	}
	// Increase complexity.
	complexityRecord, err := a.stor.scriptsComplexity.newestScriptComplexityByAddr(*scriptAddr, !initialisation)
	if err != nil {
		return nil, errors.Wrap(err, "newestScriptComplexityByAsset()")
	}
	a.totalComplexity += complexityRecord.verifierComplexity
	return sr, nil
}

func (a *scriptCaller) getTotalComplexity() uint64 {
	return a.totalComplexity
}

func (a *scriptCaller) resetComplexity() {
	a.totalComplexity = 0
}
