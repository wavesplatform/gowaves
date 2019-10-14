package state

import (
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/evaluate"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type scriptCaller struct {
	state types.SmartState

	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
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
	ok, err := evaluate.Verify(a.settings.AddressSchemeCharacter, a.state, &script, obj, this, lastBlock)
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
	return nil
}

func (a *scriptCaller) callAssetScript(tx proto.Transaction, assetID crypto.Digest, lastBlockInfo *proto.BlockInfo, initialisation bool) error {
	script, err := a.stor.scriptsStorage.newestScriptByAsset(assetID, !initialisation)
	if err != nil {
		return errors.Errorf("failed to retrieve asset script: %v\n", err)
	}
	obj, err := ast.NewVariablesFromTransaction(a.settings.AddressSchemeCharacter, tx)
	if err != nil {
		return errors.Wrap(err, "failed to convert transaction")
	}
	assetInfo, err := a.state.NewestAssetInfo(assetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve asset info")
	}
	this := ast.NewObjectFromAssetInfo(*assetInfo)
	lastBlock := ast.NewObjectFromBlockInfo(*lastBlockInfo)
	if err := a.callVerifyScript(script, obj, this, lastBlock); err != nil {
		id, _ := tx.GetID()
		return errors.Errorf("asset script; transaction ID %s: %v\n", base58.Encode(id), err)
	}
	return nil
}
