package state

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

var (
	invokeFee = FeeUnit * feeConstants[proto.InvokeScriptTransaction]
	feeAsset  = proto.OptionalAsset{Present: false}
)

func invokeSenderRecipient() (proto.Recipient, proto.Recipient) {
	return testGlobal.senderInfo.rcp, testGlobal.recipientInfo.rcp
}

type invokeApplierTestObjects struct {
	state *stateManager
}

func createInvokeApplierTestObjects(t *testing.T) (*invokeApplierTestObjects, string) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test state")
	state, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")
	err = state.stateDB.addBlock(blockID0)
	assert.NoError(t, err)
	to := &invokeApplierTestObjects{state}
	to.activateFeature(t, int16(settings.Ride4DApps))
	return to, dataDir
}

func (to *invokeApplierTestObjects) fallibleValidationParams(t *testing.T) *fallibleValidationParams {
	info := defaultFallibleValidationParams(t)
	err := to.state.stateDB.addBlock(info.block.BlockID())
	assert.NoError(t, err)
	return info
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

func (to *invokeApplierTestObjects) setAndCheckInitialWavesBalance(t *testing.T, addr proto.Address, balance uint64) {
	to.setInitialWavesBalance(t, addr, balance)
	senderBalance, err := to.state.NewestAccountBalance(proto.NewRecipientFromAddress(addr), nil)
	assert.NoError(t, err)
	assert.Equal(t, balance, senderBalance)
}

func (to *invokeApplierTestObjects) setScript(t *testing.T, addr proto.Address, pk crypto.PublicKey, script proto.Script) {
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
	err = to.state.stor.scriptsStorage.setAccountScript(addr, script, pk, blockID0)
	assert.NoError(t, err, "failed to set account script")
}

func (to *invokeApplierTestObjects) setDApp(t *testing.T, dappFilename string, dappAddr *testAddrData) {
	dir, err := getLocalDir()
	assert.NoError(t, err, "getLocalDir() failed")
	dAppPath := filepath.Join(dir, "testdata", "scripts", dappFilename)
	scriptBase64, err := ioutil.ReadFile(dAppPath)
	assert.NoError(t, err, "ReadFile() failed")
	scriptBytes, err := reader.ScriptBytesFromBase64(scriptBase64)
	assert.NoError(t, err, "ScriptBytesFromBase64() failed")
	to.setScript(t, dappAddr.addr, dappAddr.pk, scriptBytes)
}

func (to *invokeApplierTestObjects) activateFeature(t *testing.T, feature int16) {
	req := &activatedFeaturesRecord{1}
	err := to.state.stor.features.activateFeature(feature, req, blockID0)
	assert.NoError(t, err)
	err = to.state.flush(true)
	assert.NoError(t, err)
	to.state.reset()
}

func (to *invokeApplierTestObjects) applyAndSaveInvoke(t *testing.T, tx *proto.InvokeScriptWithProofs, info *fallibleValidationParams) *applicationResult {
	// TODO: consider rewriting using txAppender.
	// This should simplify tests because we actually reimplement part of appendTx() here.
	defer to.state.stor.dropUncertain()

	res, err := to.state.appender.ia.applyInvokeScript(tx, info)
	assert.NoError(t, err)
	err = to.state.appender.diffStor.saveTxDiff(res.changes.diff)
	assert.NoError(t, err)
	if res.status {
		err = to.state.stor.commitUncertain(info.checkerInfo.blockID)
		assert.NoError(t, err)
	}
	return res
}

func createGeneratedAsset(t *testing.T) (crypto.Digest, string) {
	name := "Somerset"
	description := fmt.Sprintf("Asset '%s' was generated automatically", name)
	fc := proto.FunctionCall{Name: "issue", Arguments: []proto.Argument{&proto.StringArgument{Value: name}}}
	tx := createInvokeScriptWithProofs(t, []proto.ScriptPayment{}, fc, feeAsset, invokeFee)
	return proto.GenerateIssueScriptActionID(name, description, 2, 100000, true, 0, *tx.ID), name
}

type rcpAsset struct {
	rcp     proto.Recipient
	assetId *crypto.Digest
}

func (r *rcpAsset) asset() []byte {
	if r.assetId == nil {
		return nil
	}
	return r.assetId[:]
}

type rcpKey struct {
	rcp proto.Recipient
	key string
}

type invokeApplierTestData struct {
	// Indicates that invocation should happen multiple times.
	invokeMultipleTimes bool
	// How many times to run invoke.
	invokeTimes int

	// Invoke arguments.
	payments proto.ScriptPayments
	fc       proto.FunctionCall

	// Results.
	errorRes bool
	failRes  bool

	// Result state.
	correctBalances map[rcpAsset]uint64
	dataEntries     map[rcpKey]proto.DataEntry
	correctAddrs    []proto.Address
}

func (id *invokeApplierTestData) applyTest(t *testing.T, to *invokeApplierTestObjects, info *fallibleValidationParams) {
	tx := createInvokeScriptWithProofs(t, id.payments, id.fc, feeAsset, invokeFee)
	if id.errorRes {
		_, err := to.state.appender.ia.applyInvokeScript(tx, info)
		assert.Error(t, err)
		return
	}
	if !id.invokeMultipleTimes {
		id.invokeTimes = 1
	}
	for i := 0; i < id.invokeTimes; i++ {
		res := to.applyAndSaveInvoke(t, tx, info)
		assert.Equal(t, !id.failRes, res.status)
		assert.ElementsMatch(t, id.correctAddrs, res.changes.addresses())
	}

	// Check newest result state here.
	for aa, correct := range id.correctBalances {
		balance, err := to.state.NewestAccountBalance(aa.rcp, aa.asset())
		assert.NoError(t, err)
		assert.Equal(t, correct, balance)
	}
	for ak, correct := range id.dataEntries {
		entry, err := to.state.RetrieveNewestEntry(ak.rcp, ak.key)
		assert.NoError(t, err)
		assert.Equal(t, correct, entry)
	}

	// Flush.
	err := to.state.appender.applyAllDiffs(false)
	assert.NoError(t, err, "applyAllDiffs() failed")
	err = to.state.flush(false)
	assert.NoError(t, err, "state.flush() failed")
	to.state.reset()

	// Check state after flushing.
	for aa, correct := range id.correctBalances {
		balance, err := to.state.AccountBalance(aa.rcp, aa.asset())
		assert.NoError(t, err)
		assert.Equal(t, correct, balance)
	}
	for ak, correct := range id.dataEntries {
		entry, err := to.state.RetrieveEntry(ak.rcp, ak.key)
		assert.NoError(t, err)
		assert.Equal(t, correct, entry)
	}
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

func TestApplyInvokeScriptPaymentsAndData(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		assert.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		assert.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "dapp.base64", testGlobal.recipientInfo)

	amount := uint64(34)
	startBalance := amount + invokeFee + 1
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, startBalance)

	sender, dapp := invokeSenderRecipient()
	pmts := []proto.ScriptPayment{
		{Amount: amount},
	}
	fc0 := proto.FunctionCall{Name: "deposit"}
	key := base58.Encode(testGlobal.senderInfo.addr[:])
	tests := []invokeApplierTestData{
		{
			payments: pmts,
			fc:       fc0,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 1,
				{dapp, nil}:   amount,
			},
			dataEntries: map[rcpKey]proto.DataEntry{
				{dapp, key}: &proto.IntegerDataEntry{Key: key, Value: int64(amount)},
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

func TestApplyInvokeScriptTransfers(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		assert.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		assert.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "dapp.base64", testGlobal.recipientInfo)

	amount := uint64(34)
	startBalance := amount + invokeFee*2
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, startBalance)

	sender, dapp := invokeSenderRecipient()
	pmts := []proto.ScriptPayment{
		{Amount: amount},
	}
	fc0 := proto.FunctionCall{Name: "deposit"}
	withdrawAmount := amount / 2
	fc1 := proto.FunctionCall{Name: "withdraw", Arguments: proto.Arguments{&proto.IntegerArgument{Value: int64(withdrawAmount)}}}
	key := base58.Encode(testGlobal.senderInfo.addr[:])
	tests := []invokeApplierTestData{
		{
			payments: pmts,
			fc:       fc0,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: invokeFee,
				{dapp, nil}:   amount,
			},
			dataEntries: map[rcpKey]proto.DataEntry{
				{dapp, key}: &proto.IntegerDataEntry{Key: key, Value: int64(amount)},
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: withdrawAmount,
				{dapp, nil}:   amount - withdrawAmount,
			},
			dataEntries: map[rcpKey]proto.DataEntry{
				{dapp, key}: &proto.IntegerDataEntry{Key: key, Value: int64(amount - withdrawAmount)},
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

func TestApplyInvokeScriptWithIssues(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		require.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		require.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee)

	sender, dapp := invokeSenderRecipient()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.FunctionCall{Name: "issue", Arguments: []proto.Argument{&proto.StringArgument{Value: name}}}
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     0,
				{dapp, &newAsset}: 100000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

func TestApplyInvokeScriptWithIssuesThenReissue(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		require.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		require.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*2)

	sender, dapp := invokeSenderRecipient()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.FunctionCall{Name: "issue", Arguments: []proto.Argument{&proto.StringArgument{Value: name}}}
	fc1 := proto.FunctionCall{Name: "reissue", Arguments: []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}}}
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     invokeFee,
				{dapp, &newAsset}: 100000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     0,
				{dapp, &newAsset}: 110000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

func TestApplyInvokeScriptWithIssuesThenReissueThenBurn(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		require.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		require.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*3)

	sender, dapp := invokeSenderRecipient()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.FunctionCall{Name: "issue", Arguments: []proto.Argument{&proto.StringArgument{Value: name}}}
	fc1 := proto.FunctionCall{Name: "reissue", Arguments: []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}}}
	fc2 := proto.FunctionCall{Name: "burn", Arguments: []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}}}
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     invokeFee * 2,
				{dapp, &newAsset}: 100000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     invokeFee,
				{dapp, &newAsset}: 110000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc2,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     0,
				{dapp, &newAsset}: 105000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

func TestApplyInvokeScriptWithIssuesThenReissueThenFailOnReissue(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		require.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		require.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*3)

	sender, dapp := invokeSenderRecipient()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.FunctionCall{Name: "issue", Arguments: []proto.Argument{&proto.StringArgument{Value: name}}}
	fc1 := proto.FunctionCall{Name: "reissue", Arguments: []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}}}
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     invokeFee * 2,
				{dapp, &newAsset}: 100000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     invokeFee,
				{dapp, &newAsset}: 110000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: true, // Second reissue should fail as asset made non-reissuable with the first one.
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

func TestApplyInvokeScriptWithIssuesThenFailOnBurnTooMuch(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		require.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		require.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*100)

	sender, dapp := invokeSenderRecipient()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.FunctionCall{Name: "issue", Arguments: []proto.Argument{&proto.StringArgument{Value: name}}}
	fc1 := proto.FunctionCall{Name: "burn", Arguments: []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}}}
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     invokeFee * 99,
				{dapp, &newAsset}: 100000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			invokeMultipleTimes: true,
			invokeTimes:         20,
			payments:            []proto.ScriptPayment{},
			fc:                  fc1,
			errorRes:            false,
			failRes:             false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     invokeFee * 79,
				{dapp, &newAsset}: 0,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: true,
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

func TestFailedApplyInvokeScript(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)

	defer func() {
		err := to.state.Close()
		require.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		require.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	info.acceptFailed = true
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*3)

	sender, dapp := invokeSenderRecipient()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.FunctionCall{Name: "issue", Arguments: []proto.Argument{&proto.StringArgument{Value: name}}}
	fc1 := proto.FunctionCall{Name: "reissue", Arguments: []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}}}
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     invokeFee * 2,
				{dapp, &newAsset}: 100000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     invokeFee,
				{dapp, &newAsset}: 110000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: false,
			failRes:  true,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     0,
				{dapp, &newAsset}: 110000,
			},
			correctAddrs: []proto.Address{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr, // Script address should be although its balance does not change.
			},
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

// TODO: add test on sponsorship made by DApp: create new DApp that will issue and sponsor asset,
// test also the function call that issues and sets sponsorship in one turn.

// TODO: add test on impossibility of sponsorship of smart asset using DApp: issue smart asset with simple script using
// usual transaction and then try to set sponsorship using invoke.
