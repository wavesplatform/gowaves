package state

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type invokeApplierTestObjects struct {
	state *stateManager
}

func createInvokeApplierTestObjects(t *testing.T) (*invokeApplierTestObjects, string) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test state")
	state, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")
	return &invokeApplierTestObjects{state}, dataDir
}

func (to *invokeApplierTestObjects) setInitialWavesBalance(t *testing.T, addr proto.Address, balance uint64) {
	txDiff := newTxDiff()
	key := wavesBalanceKey{addr}
	diff := newBalanceDiff(int64(balance), 0, 0, false)
	diff.blockID = blockID0
	err := txDiff.appendBalanceDiff(key.bytes(), diff)
	assert.NoError(t, err, "appendBalanceDiff() failed")
	err = to.state.appender.diffStor.saveTxDiff(txDiff)
	assert.NoError(t, err, "saveTxDiff() failed")
}

func (to *invokeApplierTestObjects) setScript(t *testing.T, addr proto.Address, script proto.Script) {
	scriptAst, err := ast.BuildScript(reader.NewBytesReader(script))
	assert.NoError(t, err)
	estimator := estimatorByScript(scriptAst, 1)
	complexity, err := estimator.Estimate(scriptAst)
	assert.NoError(t, err)
	r := &accountScriptComplexityRecord{
		verifierComplexity: complexity.Verifier,
		byFuncs:            complexity.Functions,
		estimator:          byte(estimator.Version),
	}
	err = to.state.stor.scriptsComplexity.saveComplexityForAddr(addr, r, blockID0)
	assert.NoError(t, err, "failed to save complexity for address")
	err = to.state.stor.scriptsStorage.setAccountScript(addr, script, blockID0)
	assert.NoError(t, err, "failed to set account script")
}

/*
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func deposit() = {
   let pmt = extract(i.payment)
   if (isDefined(pmt.assetId)) then throw("can hold waves only at the moment")
   else {
        let currentKey = toBase58String(i.caller.bytes)
        let currentAmount = match getInteger(this, currentKey) {
            case a:Int => a
            case _ => 0
        }
        let newAmount = currentAmount + pmt.amount
        WriteSet([DataEntry(currentKey, newAmount)])
   }
}

@Callable(i)
func withdraw(amount: Int) = {
   let currentKey = toBase58String(i.caller.bytes)
    let currentAmount = match getInteger(this, currentKey) {
        case a:Int => a
        case _ => 0
    }
    let newAmount = currentAmount - amount
    if (amount < 0)
        then throw("Can't withdraw negative amount")
    else if (newAmount < 0)
            then throw("Not enough balance")
        else ScriptResult(
            WriteSet([DataEntry(currentKey, newAmount)]),
            TransferSet([ScriptTransfer(i.caller, amount, unit)])
        )
}

@Verifier(tx)
func verify() = {
    sigVerify(tx.bodyBytes, tx.proofs[0], tx.senderPublicKey)
}
*/

func TestApplyInvokeScriptWithProofsPaymentsAndData(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		assert.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		assert.NoError(t, err, "failed to remove test data dir")
	}()

	// Invoke applier object.
	ia := to.state.appender.ia
	info := &invokeAddlInfo{
		block:  &proto.BlockHeader{BlockSignature: blockID0.Signature(), Timestamp: to.state.settings.CheckTempNegativeAfterTime},
		height: 1,
	}
	err := to.state.stateDB.addBlock(info.block.BlockID())
	assert.NoError(t, err)
	dir, err := getLocalDir()
	assert.NoError(t, err, "getLocalDir() failed")
	dAppPath := filepath.Join(dir, "testdata", "scripts", "dapp.base64")
	scriptBase64, err := ioutil.ReadFile(dAppPath)
	assert.NoError(t, err, "ReadFile() failed")
	scriptBytes, err := reader.ScriptBytesFromBase64(scriptBase64)
	assert.NoError(t, err, "ScriptBytesFromBase64() failed")
	to.setScript(t, testGlobal.recipientInfo.addr, proto.Script(scriptBytes))

	amount := uint64(34)
	fee := FeeUnit * feeConstants[proto.InvokeScriptTransaction]
	startBalance := amount + fee + 1
	to.setInitialWavesBalance(t, testGlobal.senderInfo.addr, startBalance)
	senderBalance, err := to.state.NewestAccountBalance(proto.NewRecipientFromAddress(testGlobal.senderInfo.addr), nil)
	assert.NoError(t, err)
	assert.Equal(t, startBalance, senderBalance)

	pmts := []proto.ScriptPayment{
		{Amount: amount},
	}
	fc := proto.FunctionCall{Name: "deposit"}
	tx := createInvokeScriptWithProofs(t, pmts, fc, fee)
	tx.FeeAsset = proto.OptionalAsset{Present: false}
	ch, err := ia.applyInvokeScriptWithProofs(tx, info)
	assert.NoError(t, err, "failed to apply valid InvokeScriptWithProofs tx")
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr:    empty,
		testGlobal.recipientInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)

	// Check newest result state here.
	senderBalance, err = to.state.NewestAccountBalance(proto.NewRecipientFromAddress(testGlobal.senderInfo.addr), nil)
	assert.NoError(t, err)
	assert.Equal(t, startBalance-amount-fee, senderBalance)
	recipientBalance, err := to.state.NewestAccountBalance(tx.ScriptRecipient, nil)
	assert.NoError(t, err)
	assert.Equal(t, amount, recipientBalance)
	key := base58.Encode(testGlobal.senderInfo.addr[:])
	entry, err := to.state.RetrieveNewestIntegerEntry(tx.ScriptRecipient, key)
	assert.NoError(t, err)
	assert.Equal(t, &proto.IntegerDataEntry{Key: key, Value: int64(amount)}, entry)

	err = to.state.appender.applyAllDiffs(false)
	assert.NoError(t, err, "applyAllDiffs() failed")
	err = to.state.flush(false)
	assert.NoError(t, err, "state.flush() failed")
	err = to.state.reset(false)
	assert.NoError(t, err, "state.reset() failed")

	// Check after flushing.
	senderBalance, err = to.state.AccountBalance(proto.NewRecipientFromAddress(testGlobal.senderInfo.addr), nil)
	assert.NoError(t, err)
	assert.Equal(t, startBalance-amount-fee, senderBalance)
	recipientBalance, err = to.state.AccountBalance(tx.ScriptRecipient, nil)
	assert.NoError(t, err)
	assert.Equal(t, amount, recipientBalance)
	entry, err = to.state.RetrieveIntegerEntry(tx.ScriptRecipient, key)
	assert.NoError(t, err)
	assert.Equal(t, &proto.IntegerDataEntry{Key: key, Value: int64(amount)}, entry)
}

func TestApplyInvokeScriptWithProofsTransfers(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		assert.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		assert.NoError(t, err, "failed to remove test data dir")
	}()

	// Invoke applier object.
	ia := to.state.appender.ia
	info := &invokeAddlInfo{
		block:  &proto.BlockHeader{BlockSignature: blockID0.Signature(), Timestamp: to.state.settings.CheckTempNegativeAfterTime},
		height: 1,
	}
	err := to.state.stateDB.addBlock(info.block.BlockID())
	assert.NoError(t, err)
	dir, err := getLocalDir()
	assert.NoError(t, err, "getLocalDir() failed")
	dAppPath := filepath.Join(dir, "testdata", "scripts", "dapp.base64")
	scriptBase64, err := ioutil.ReadFile(dAppPath)
	assert.NoError(t, err, "ReadFile() failed")
	scriptBytes, err := reader.ScriptBytesFromBase64(scriptBase64)
	assert.NoError(t, err, "ScriptBytesFromBase64() failed")
	to.setScript(t, testGlobal.recipientInfo.addr, proto.Script(scriptBytes))

	amount := uint64(34)
	withdrawAmount := amount / 2
	fee := FeeUnit * feeConstants[proto.InvokeScriptTransaction]
	startBalance := amount + fee*2
	to.setInitialWavesBalance(t, testGlobal.senderInfo.addr, startBalance)
	senderBalance, err := to.state.NewestAccountBalance(proto.NewRecipientFromAddress(testGlobal.senderInfo.addr), nil)
	assert.NoError(t, err)
	assert.Equal(t, startBalance, senderBalance)

	pmts := []proto.ScriptPayment{
		{Amount: amount},
	}
	fc := proto.FunctionCall{Name: "deposit"}
	tx := createInvokeScriptWithProofs(t, pmts, fc, fee)
	tx.FeeAsset = proto.OptionalAsset{Present: false}
	ch, err := ia.applyInvokeScriptWithProofs(tx, info)
	assert.NoError(t, err, "failed to apply valid InvokeScriptWithProofs tx")
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr:    empty,
		testGlobal.recipientInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)

	fc = proto.FunctionCall{Name: "withdraw", Arguments: proto.Arguments{&proto.IntegerArgument{Value: int64(withdrawAmount)}}}
	tx = createInvokeScriptWithProofs(t, []proto.ScriptPayment{}, fc, fee)
	tx.FeeAsset = proto.OptionalAsset{Present: false}
	ch, err = ia.applyInvokeScriptWithProofs(tx, info)
	assert.NoError(t, err, "failed to apply valid InvokeScriptWithProofs tx")
	correctAddrs = map[proto.Address]struct{}{
		testGlobal.senderInfo.addr:    empty,
		testGlobal.recipientInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)

	// Check newest result state here.
	senderBalance, err = to.state.NewestAccountBalance(proto.NewRecipientFromAddress(testGlobal.senderInfo.addr), nil)
	assert.NoError(t, err)
	assert.Equal(t, withdrawAmount, senderBalance)
	recipientBalance, err := to.state.NewestAccountBalance(tx.ScriptRecipient, nil)
	assert.NoError(t, err)
	assert.Equal(t, amount-withdrawAmount, recipientBalance)

	err = to.state.appender.applyAllDiffs(false)
	assert.NoError(t, err, "applyAllDiffs() failed")
	err = to.state.flush(false)
	assert.NoError(t, err, "state.flush() failed")
	err = to.state.reset(false)
	assert.NoError(t, err, "state.reset() failed")

	// Check after flushing.
	senderBalance, err = to.state.AccountBalance(proto.NewRecipientFromAddress(testGlobal.senderInfo.addr), nil)
	assert.NoError(t, err)
	assert.Equal(t, withdrawAmount, senderBalance)
	recipientBalance, err = to.state.AccountBalance(tx.ScriptRecipient, nil)
	assert.NoError(t, err)
	assert.Equal(t, amount-withdrawAmount, recipientBalance)
}
