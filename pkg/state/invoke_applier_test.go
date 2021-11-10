package state

import (
	"encoding/base64"
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
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

var (
	invokeFee = FeeUnit * feeConstants[proto.InvokeScriptTransaction]
	feeAsset  = proto.NewOptionalAssetWaves()
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
	to.activateFeature(t, int16(settings.SmartAccounts))
	to.activateFeature(t, int16(settings.Ride4DApps))
	return to, dataDir
}

func (to *invokeApplierTestObjects) fallibleValidationParams(t *testing.T) *fallibleValidationParams {
	info := defaultFallibleValidationParams()
	err := to.state.stateDB.addBlock(info.block.BlockID())
	assert.NoError(t, err)
	return info
}

func (to *invokeApplierTestObjects) setInitialWavesBalance(t *testing.T, addr proto.WavesAddress, balance uint64) {
	txDiff := newTxDiff()
	key := wavesBalanceKey{addr.ID()}
	diff := newBalanceDiff(int64(balance), 0, 0, false)
	diff.blockID = blockID0
	err := txDiff.appendBalanceDiff(key.bytes(), diff)
	assert.NoError(t, err, "appendBalanceDiff() failed")
	err = to.state.appender.diffStor.saveTxDiff(txDiff)
	assert.NoError(t, err, "saveTxDiff() failed")
}

func (to *invokeApplierTestObjects) setAndCheckInitialWavesBalance(t *testing.T, addr proto.WavesAddress, balance uint64) {
	to.setInitialWavesBalance(t, addr, balance)
	senderBalance, err := to.state.NewestWavesBalance(proto.NewRecipientFromAddress(addr))
	assert.NoError(t, err)
	assert.Equal(t, balance, senderBalance)
}

func (to *invokeApplierTestObjects) setScript(t *testing.T, addr proto.WavesAddress, pk crypto.PublicKey, script proto.Script) {
	tree, err := ride.Parse(script)
	require.NoError(t, err)
	estimation, err := ride.EstimateTree(tree, 1)
	require.NoError(t, err)
	err = to.state.stor.scriptsComplexity.saveComplexitiesForAddr(addr, map[int]ride.TreeEstimation{1: estimation}, blockID0)
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
	scriptBytes := make([]byte, base64.StdEncoding.DecodedLen(len(scriptBase64)))
	l, err := base64.StdEncoding.Decode(scriptBytes, scriptBase64)
	assert.NoError(t, err, "ScriptBytesFromBase64() failed")
	to.setScript(t, dappAddr.addr, dappAddr.pk, scriptBytes[:l])
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
	require.NoError(t, err)
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
	rcp   proto.Recipient
	asset *crypto.Digest
}

type rcpKey struct {
	rcp proto.Recipient
	key string
}

type fullBalance struct {
	regular    uint64
	generating uint64
	available  uint64
	effective  uint64
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
	correctBalances     map[rcpAsset]uint64
	correctFullBalances map[proto.Recipient]fullBalance
	dataEntries         map[rcpKey]proto.DataEntry
	correctAddrs        []proto.WavesAddress
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
		var (
			balance uint64
			err     error
		)
		if aa.asset != nil {
			balance, err = to.state.NewestAssetBalance(aa.rcp, *aa.asset)
		} else {
			balance, err = to.state.NewestWavesBalance(aa.rcp)
		}
		assert.NoError(t, err)
		assert.Equal(t, int(correct), int(balance))
	}
	for aa, correct := range id.correctFullBalances {
		fb, err := to.state.NewestFullWavesBalance(aa)
		assert.NoError(t, err)
		assert.Equal(t, int(correct.available), int(fb.Available))
		assert.Equal(t, int(correct.effective), int(fb.Effective))
		assert.Equal(t, int(correct.generating), int(fb.Generating))
		assert.Equal(t, int(correct.regular), int(fb.Regular))
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
		var (
			balance uint64
			err     error
		)
		if aa.asset != nil {
			balance, err = to.state.AssetBalance(aa.rcp, proto.AssetIDFromDigest(*aa.asset))
		} else {
			balance, err = to.state.WavesBalance(aa.rcp)
		}
		assert.NoError(t, err)
		assert.Equal(t, int(correct), int(balance))
	}
	//for aa, correct := range id.correctFullBalances {
	//	fb, err := to.state.FullWavesBalance(aa)
	//	assert.NoError(t, err)
	//	assert.Equal(t, correct.available, fb.Available)
	//	assert.Equal(t, correct.effective, fb.Effective)
	//	assert.Equal(t, correct.generating, fb.Generating)
	//	assert.Equal(t, correct.regular, fb.Regular)
	//}
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
	info.blockV5Activated = true
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
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
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr, // Script address should be although its balance does not change.
			},
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

// Tests on leasing actions use the following script
/*
{-# STDLIB_VERSION 5 #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# CONTENT_TYPE DAPP #-}

@Callable(i)
func simpleLeaseToAddress(rcp: String, amount: Int) = {
    let addr = addressFromStringValue(rcp)
    ([Lease(addr, amount)], unit)
}

@Callable(i)
func detailedLeaseToAddress(rcp: String, amount: Int) = {
    let addr = addressFromStringValue(rcp)
    let lease = Lease(addr, amount, 0)
    let id = calculateLeaseId(lease)
    ([lease], id)
}

@Callable(i)
func simpleLeaseToAlias(rcp: String, amount: Int) = {
    let alias = Alias(rcp)
    ([Lease(alias, amount)], unit)
}

@Callable(i)
func detailedLeaseToAlias(rcp: String, amount: Int) = {
    let alias = Alias(rcp)
    let lease = Lease(alias, amount, 0)
    let id = calculateLeaseId(lease)
    ([lease], id)
}

@Callable(i)
func simpleLeaseToSender(amount: Int) = {
    ([Lease(i.caller, amount)], unit)
}

@Callable(i)
func detailedLeaseToSender(amount: Int) = {
    let lease = Lease(i.caller, amount, 0)
    let id = calculateLeaseId(lease)
    ([lease], id)
}

@Callable(i)
func cancel(id: ByteVector) = ([LeaseCancel(id)], unit)

*/

func TestApplyInvokeScriptWithLease(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)
	to.activateFeature(t, int16(settings.RideV5))

	defer func() {
		err := to.state.Close()
		require.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		require.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride5_leasing.base64", testGlobal.recipientInfo)

	var thousandWaves int64 = 1_000 * 100_000_000
	// Invoker pays only fee, but receives a leasing of 1000 waves
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee)
	to.setAndCheckInitialWavesBalance(t, testGlobal.recipientInfo.addr, uint64(2*thousandWaves))

	sender, dapp := invokeSenderRecipient()
	fc := proto.FunctionCall{
		Name:      "simpleLeaseToSender",
		Arguments: []proto.Argument{&proto.IntegerArgument{Value: thousandWaves}},
	}
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 0,
			},
			correctFullBalances: map[proto.Recipient]fullBalance{
				sender: {regular: 0, generating: 0, available: 0, effective: uint64(thousandWaves)},
				dapp:   {regular: uint64(2 * thousandWaves), generating: 0, available: uint64(thousandWaves), effective: uint64(thousandWaves)},
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
	}
	for _, tc := range tests {
		tc.applyTest(t, to, info)
	}
}

func TestApplyInvokeScriptWithLeaseAndLeaseCancel(t *testing.T) {
	to, path := createInvokeApplierTestObjects(t)
	to.activateFeature(t, int16(settings.RideV5))

	defer func() {
		err := to.state.Close()
		require.NoError(t, err, "state.Close() failed")
		err = os.RemoveAll(path)
		require.NoError(t, err, "failed to remove test data dir")
	}()

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride5_leasing.base64", testGlobal.recipientInfo)

	var thousandWaves int64 = 1_000 * 100_000_000
	// Invoker pays only fee, but receives a leasing of 1000 waves
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, 2*invokeFee)
	to.setAndCheckInitialWavesBalance(t, testGlobal.recipientInfo.addr, uint64(2*thousandWaves))

	sender, dapp := invokeSenderRecipient()
	fc1 := proto.FunctionCall{
		Name:      "simpleLeaseToSender",
		Arguments: []proto.Argument{&proto.IntegerArgument{Value: thousandWaves}},
	}
	tx := createInvokeScriptWithProofs(t, []proto.ScriptPayment{}, fc1, feeAsset, invokeFee)
	id := proto.GenerateLeaseScriptActionID(sender, thousandWaves, 0, *tx.ID)
	fc2 := proto.FunctionCall{
		Name:      "cancel",
		Arguments: []proto.Argument{&proto.BinaryArgument{Value: id.Bytes()}},
	}
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: invokeFee,
			},
			correctFullBalances: map[proto.Recipient]fullBalance{
				sender: {regular: invokeFee, generating: 0, available: invokeFee, effective: uint64(thousandWaves) + invokeFee},
				dapp:   {regular: uint64(2 * thousandWaves), generating: 0, available: uint64(thousandWaves), effective: uint64(thousandWaves)},
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc2,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 0,
			},
			correctFullBalances: map[proto.Recipient]fullBalance{
				sender: {regular: 0, generating: 0, available: 0, effective: 0},
				dapp:   {regular: uint64(2 * thousandWaves), generating: 0, available: uint64(2 * thousandWaves), effective: uint64(2 * thousandWaves)},
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
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
