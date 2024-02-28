package state

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

var (
	invokeFee        = FeeUnit * feeConstants[proto.InvokeScriptTransaction]
	feeAsset         = proto.NewOptionalAssetWaves()
	cutCommentsRegex = regexp.MustCompile(`\s*#.*\n?`)
)

func invokeSenderRecipientAddresses() (proto.WavesAddress, proto.WavesAddress) {
	return *testGlobal.senderInfo.rcp.Address(), *testGlobal.recipientInfo.rcp.Address()
}

type invokeApplierTestObjects struct {
	state *stateManager
}

func createInvokeApplierTestObjects(t *testing.T) *invokeApplierTestObjects {
	state, err := newStateManager(t.TempDir(), true, DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")
	to := &invokeApplierTestObjects{state}
	randGenesisBlockID := genRandBlockId(t)
	to.addBlock(t, randGenesisBlockID)

	to.prepareBlock(t, blockID0) // height here is 2

	to.activateFeature(t, int16(settings.NG))
	to.activateFeature(t, int16(settings.SmartAccounts))
	to.activateFeature(t, int16(settings.Ride4DApps))
	t.Cleanup(func() {
		to.cleanup()
		assert.NoError(t, to.state.Close(), "state.Close() failed")
	})
	return to
}

// prepareBlock makes test block officially valid (but only after batch is flushed).
func (to *invokeApplierTestObjects) prepareBlock(t *testing.T, blockID proto.BlockID) {
	// Assign unique block number for this block ID, add this number to the list of valid blocks.
	err := to.state.stateDB.addBlock(blockID)
	assert.NoError(t, err, "stateDB.addBlock() failed")
}

// addBlock prepares, starts and finishes fake block.
func (to *invokeApplierTestObjects) addBlock(t *testing.T, blockID proto.BlockID) {
	to.prepareBlock(t, blockID)
	err := to.state.rw.startBlock(blockID)
	assert.NoError(t, err, "startBlock() failed")
	err = to.state.rw.finishBlock(blockID)
	assert.NoError(t, err, "finishBlock() failed")
}

func (to *invokeApplierTestObjects) cleanup() {
	to.state.stor.dropUncertain()
	to.state.appender.ia.sc.resetComplexity()
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
	const currentEstimatorVersion = 1
	tree, err := serialization.Parse(script)
	require.NoError(t, err)
	estimation, err := ride.EstimateTree(tree, currentEstimatorVersion)
	require.NoError(t, err)
	se := scriptEstimation{
		currentEstimatorVersion: currentEstimatorVersion,
		scriptIsEmpty:           script.IsEmpty(),
		estimation:              estimation,
	}
	err = to.state.stor.scriptsComplexity.saveComplexitiesForAddr(addr, se, blockID0)
	assert.NoError(t, err, "failed to save complexity for address")
	err = to.state.stor.scriptsStorage.setAccountScript(addr, script, pk, blockID0)
	assert.NoError(t, err, "failed to set account script")
}

func readTestScript(name string) ([]byte, error) {
	dir, err := getLocalDir()
	if err != nil {
		return nil, err
	}
	dAppPath := filepath.Join(dir, "testdata", "scripts", name)
	scriptFileContent, err := os.ReadFile(dAppPath)
	if err != nil {
		return nil, err
	}
	scriptBase64WithComments := string(scriptFileContent)
	scriptBase64WithoutComments := cutCommentsRegex.ReplaceAllString(scriptBase64WithComments, "")
	scriptBase64 := strings.TrimSpace(scriptBase64WithoutComments)

	return base64.StdEncoding.DecodeString(scriptBase64)
}

func (to *invokeApplierTestObjects) setDApp(t *testing.T, dappFilename string, dappAddr *testWavesAddrData) {
	scriptBytes, err := readTestScript(dappFilename)
	assert.NoError(t, err, "ScriptBytesFromBase64() failed")
	to.setScript(t, dappAddr.addr, dappAddr.pk, scriptBytes)
}

func (to *invokeApplierTestObjects) activateFeature(t *testing.T, feature int16) {
	req := &activatedFeaturesRecord{1}
	err := to.state.stor.features.activateFeature(feature, req, blockID0)
	assert.NoError(t, err)
	err = to.state.flush()
	assert.NoError(t, err)
	to.state.reset()
}

func (to *invokeApplierTestObjects) applyAndSaveInvoke(t *testing.T, tx *proto.InvokeScriptWithProofs,
	info *fallibleValidationParams,
	shouldCommitUncertain bool) (*invocationResult, *applicationResult) {
	// TODO: consider rewriting using txAppender.
	// This should simplify tests because we actually reimplement part of appendTx() here.
	if shouldCommitUncertain {
		defer func() {
			to.state.stor.dropUncertain()
			to.state.appender.ia.sc.resetComplexity()
		}()
	}

	invocationRes, applicationRes, err := to.state.appender.ia.applyInvokeScript(tx, info)
	require.NoError(t, err)
	err = to.state.appender.diffStor.saveTxDiff(applicationRes.changes.diff)
	assert.NoError(t, err)
	if applicationRes.status && shouldCommitUncertain {
		err = to.state.stor.commitUncertain(info.checkerInfo.blockID)
		assert.NoError(t, err)
	}
	return invocationRes, applicationRes
}

func createGeneratedAsset(t *testing.T) (crypto.Digest, string) {
	name := "Somerset"
	description := fmt.Sprintf("Asset '%s' was generated automatically", name)
	fc := proto.NewFunctionCall("issue", []proto.Argument{&proto.StringArgument{Value: name}})
	tx := createInvokeScriptWithProofs(t, []proto.ScriptPayment{}, fc, feeAsset, invokeFee)
	return proto.GenerateIssueScriptActionID(name, description, 2, 100000, true, 0, *tx.ID), name
}

type rcpAsset struct {
	rcp   proto.WavesAddress
	asset *crypto.Digest
}

type rcpKey struct {
	rcp proto.WavesAddress
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
	correctFullBalances map[proto.WavesAddress]fullBalance
	dataEntries         map[rcpKey]proto.DataEntry
	correctAddrs        []proto.WavesAddress
	info                *fallibleValidationParams
}

func (id *invokeApplierTestData) applyTestWithCleanup(t *testing.T, to *invokeApplierTestObjects) {
	defer to.cleanup()
	id.applyTest(t, to)
}

func (id *invokeApplierTestData) applyTest(t *testing.T, to *invokeApplierTestObjects) {
	tx := createInvokeScriptWithProofs(t, id.payments, id.fc, feeAsset, invokeFee)
	if id.errorRes {
		_, _, err := to.state.appender.ia.applyInvokeScript(tx, id.info)
		assert.Error(t, err)
		return
	}
	if !id.invokeMultipleTimes {
		id.invokeTimes = 1
	}
	for i := 0; i < id.invokeTimes; i++ {
		_, applicationRes := to.applyAndSaveInvoke(t, tx, id.info, true)
		assert.Equal(t, !id.failRes, applicationRes.status)
		assert.ElementsMatch(t, id.correctAddrs, applicationRes.changes.addresses())
	}

	// Check newest result state here.
	for aa, correct := range id.correctBalances {
		var (
			balance uint64
			err     error
			rcp     = proto.NewRecipientFromAddress(aa.rcp)
		)
		if aa.asset != nil {
			balance, err = to.state.NewestAssetBalance(rcp, *aa.asset)
		} else {
			balance, err = to.state.NewestWavesBalance(rcp)
		}
		assert.NoError(t, err)
		assert.Equal(t, int(correct), int(balance),
			"incorrect asset balance %q for addr %q", fmt.Stringer(aa.asset), rcp.String())
	}
	for aa, correct := range id.correctFullBalances {
		fb, err := to.state.NewestFullWavesBalance(proto.NewRecipientFromAddress(aa))
		assert.NoError(t, err)
		assert.Equal(t, int(correct.available), int(fb.Available))
		assert.Equal(t, int(correct.effective), int(fb.Effective))
		assert.Equal(t, int(correct.generating), int(fb.Generating))
		assert.Equal(t, int(correct.regular), int(fb.Regular))
	}
	for ak, correct := range id.dataEntries {
		entry, err := to.state.RetrieveNewestEntry(proto.NewRecipientFromAddress(ak.rcp), ak.key)
		assert.NoError(t, err)
		assert.Equal(t, correct, entry)
	}

	// Flush.
	err := to.state.appender.applyAllDiffs()
	assert.NoError(t, err, "applyAllDiffs() failed")
	err = to.state.flush()
	assert.NoError(t, err, "state.flush() failed")
	to.state.reset()

	// Check state after flushing.
	for aa, correct := range id.correctBalances {
		var (
			balance uint64
			err     error
			rcp     = proto.NewRecipientFromAddress(aa.rcp)
		)
		if aa.asset != nil {
			balance, err = to.state.AssetBalance(rcp, proto.AssetIDFromDigest(*aa.asset))
		} else {
			balance, err = to.state.WavesBalance(rcp)
		}
		assert.NoError(t, err)
		assert.Equal(t, int(correct), int(balance),
			"incorrect asset balance %q for addr %q", fmt.Stringer(aa.asset), rcp.String())
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
		entry, err := to.state.RetrieveEntry(proto.NewRecipientFromAddress(ak.rcp), ak.key)
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
	to := createInvokeApplierTestObjects(t)

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "dapp.base64", testGlobal.recipientInfo)

	amount := uint64(34)
	startBalance := amount + invokeFee + 1
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, startBalance)

	sender, dapp := invokeSenderRecipientAddresses()
	pmts := []proto.ScriptPayment{
		{Amount: amount},
	}
	fc0 := proto.NewFunctionCall("deposit", proto.Arguments{})
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
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

func TestApplyInvokeScriptTransfers(t *testing.T) {
	to := createInvokeApplierTestObjects(t)

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "dapp.base64", testGlobal.recipientInfo)

	amount := uint64(34)
	startBalance := amount + invokeFee*2
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, startBalance)

	sender, dapp := invokeSenderRecipientAddresses()
	pmts := []proto.ScriptPayment{
		{Amount: amount},
	}
	fc0 := proto.NewFunctionCall("deposit", proto.Arguments{})
	withdrawAmount := amount / 2
	fc1 := proto.NewFunctionCall("withdraw", proto.Arguments{&proto.IntegerArgument{Value: int64(withdrawAmount)}})
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
			info: info,
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
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

func TestApplyInvokeScriptWithIssues(t *testing.T) {
	to := createInvokeApplierTestObjects(t)

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee)

	sender, dapp := invokeSenderRecipientAddresses()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.NewFunctionCall("issue", []proto.Argument{&proto.StringArgument{Value: name}})
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
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

func TestApplyInvokeScriptWithIssuesThenReissue(t *testing.T) {
	to := createInvokeApplierTestObjects(t)

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*2)

	sender, dapp := invokeSenderRecipientAddresses()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.NewFunctionCall("issue", []proto.Argument{&proto.StringArgument{Value: name}})
	fc1 := proto.NewFunctionCall("reissue", []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}})
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
			info: info,
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
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

func TestApplyInvokeScriptWithIssuesThenReissueThenBurn(t *testing.T) {
	to := createInvokeApplierTestObjects(t)

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*3)

	sender, dapp := invokeSenderRecipientAddresses()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.NewFunctionCall("issue", []proto.Argument{&proto.StringArgument{Value: name}})
	fc1 := proto.NewFunctionCall("reissue", []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}})
	fc2 := proto.NewFunctionCall("burn", []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}})
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
			info: info,
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
			info: info,
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
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

func TestApplyInvokeScriptWithIssuesThenReissueThenFailOnReissue(t *testing.T) {
	to := createInvokeApplierTestObjects(t)

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*3)

	sender, dapp := invokeSenderRecipientAddresses()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.NewFunctionCall("issue", []proto.Argument{&proto.StringArgument{Value: name}})
	fc1 := proto.NewFunctionCall("reissue", []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}})
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
			info: info,
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
			info: info,
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: true, // Second reissue should fail as asset made non-reissuable with the first one.
			info:     info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

func TestApplyInvokeScriptWithIssuesThenFailOnBurnTooMuch(t *testing.T) {
	to := createInvokeApplierTestObjects(t)

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*100)

	sender, dapp := invokeSenderRecipientAddresses()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.NewFunctionCall("issue", []proto.Argument{&proto.StringArgument{Value: name}})
	fc1 := proto.NewFunctionCall("burn", []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}})
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
			info: info,
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
			info: info,
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: true,
			info:     info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

// TestFailedApplyInvokeScript in this test we
func TestFailedApplyInvokeScript(t *testing.T) {
	to := createInvokeApplierTestObjects(t)

	info := to.fallibleValidationParams(t)
	info.acceptFailed = true
	info.blockV5Activated = true
	// We have to move height forward here because MainNet settings are used and height must be more than 2792473
	info.checkerInfo.blockchainHeight = 3_000_000
	to.setDApp(t, "ride4_asset.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*3)

	sender, dapp := invokeSenderRecipientAddresses()
	newAsset, name := createGeneratedAsset(t)
	fc := proto.NewFunctionCall("issue", []proto.Argument{&proto.StringArgument{Value: name}})
	fc1 := proto.NewFunctionCall("reissue", []proto.Argument{&proto.BinaryArgument{Value: newAsset.Bytes()}})
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
			info: info,
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
			info: info,
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: true,
			failRes:  false, // Spent complexity is less than 1000, so this transaction will be rejected
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}:     0,
				{dapp, &newAsset}: 110000,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr, // Script address should be although its balance does not change.
			},
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

func TestFailedInvokeApplicationComplexity(t *testing.T) {
	to := createInvokeApplierTestObjects(t)

	infoBefore := to.fallibleValidationParams(t)
	infoBefore.acceptFailed = true
	infoBefore.blockV5Activated = true
	infoBefore.rideV5Activated = true

	infoAfter := to.fallibleValidationParams(t)
	infoAfter.acceptFailed = true
	infoAfter.blockV5Activated = true
	infoAfter.rideV5Activated = true
	infoAfter.checkerInfo.blockchainHeight = 2_800_000

	to.setDApp(t, "ride5_recursive_invoke.base64", testGlobal.recipientInfo)

	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*3)

	sender, dapp := invokeSenderRecipientAddresses()
	// This transaction produces 10889 bytes of data in 100 entries spending 11093 of complexity
	fcEverythingFine := proto.NewFunctionCall("keyvalue", []proto.Argument{&proto.IntegerArgument{Value: 99}, &proto.StringArgument{Value: strings.Repeat("0", 100)}})
	// This transaction reaches data entries size limit (16 KB) after reaching 1000 complexity limit
	fcSizeLimitAfterComplexityLimit := proto.NewFunctionCall("keyvalue", []proto.Argument{&proto.IntegerArgument{Value: 99}, &proto.StringArgument{Value: strings.Repeat("0", 150)}})
	// This transaction reaches data entries size limit (16 KB) before reaching 1000 complexity limit
	fcSizeLimitBeforeComplexityLimit := proto.NewFunctionCall("keyvalue", []proto.Argument{&proto.IntegerArgument{Value: 11}, &proto.StringArgument{Value: strings.Repeat("0", 2000)}})
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fcEverythingFine,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: invokeFee * 2,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: infoBefore,
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fcSizeLimitAfterComplexityLimit,
			errorRes: false,
			failRes:  true,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: invokeFee,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: infoBefore,
		},
		{ // Before activation of correct fail/reject behaviour
			payments: []proto.ScriptPayment{},
			fc:       fcSizeLimitBeforeComplexityLimit,
			errorRes: false,
			failRes:  true,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 0,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: infoBefore,
		},
		{ // After activation of correct fail/reject behaviour
			payments: []proto.ScriptPayment{},
			fc:       fcSizeLimitBeforeComplexityLimit,
			errorRes: true,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 0,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: infoAfter,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

func TestFailedInvokeApplicationComplexityAfterRideV6(t *testing.T) {
	to := createInvokeApplierTestObjects(t)
	to.activateFeature(t, int16(settings.RideV5))
	to.activateFeature(t, int16(settings.RideV6))

	info := to.fallibleValidationParams(t)
	info.acceptFailed = true
	info.blockV5Activated = true
	info.rideV5Activated = true
	info.checkerInfo.blockchainHeight = 2_800_000
	info.rideV6Activated = true

	to.setDApp(t, "ride5_recursive_invoke.base64", testGlobal.recipientInfo)
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*3)

	//Note that after activation of RideV6 only the size of payload is counted
	sender, dapp := invokeSenderRecipientAddresses()
	//This transaction produces 10889 bytes of data in 100 entries spending 11093 of complexity
	fcEverythingFine := proto.NewFunctionCall("keyvalue", []proto.Argument{&proto.IntegerArgument{Value: 99}, &proto.StringArgument{Value: strings.Repeat("0", 100)}})
	// This transaction reaches data entries size limit (16 KB) after reaching 1000 complexity limit
	fcSizeLimitAfterComplexityLimit := proto.NewFunctionCall("keyvalue", []proto.Argument{&proto.IntegerArgument{Value: 99}, &proto.StringArgument{Value: strings.Repeat("0", 200)}})
	// This transaction reaches data entries size limit (16 KB) before reaching 1000 complexity limit
	fcSizeLimitBeforeComplexityLimit := proto.NewFunctionCall("keyvalue", []proto.Argument{&proto.IntegerArgument{Value: 10}, &proto.StringArgument{Value: strings.Repeat("0", 2000)}})
	tests := []invokeApplierTestData{
		{ // No error, no failure - transaction applied
			payments: []proto.ScriptPayment{},
			fc:       fcEverythingFine,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: invokeFee * 2,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
		{ // Failed transaction because of too much spent complexity
			payments: []proto.ScriptPayment{},
			fc:       fcSizeLimitAfterComplexityLimit,
			errorRes: false,
			failRes:  true,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: invokeFee,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
		{ // Rejected transaction because of low spent complexity
			payments: []proto.ScriptPayment{},
			fc:       fcSizeLimitBeforeComplexityLimit,
			errorRes: true,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 0,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
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
	to := createInvokeApplierTestObjects(t)
	to.activateFeature(t, int16(settings.RideV5))

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride5_leasing.base64", testGlobal.recipientInfo)

	var thousandWaves int64 = 1_000 * 100_000_000
	// Invoker pays only fee, but receives a leasing of 1000 waves
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee)
	to.setAndCheckInitialWavesBalance(t, testGlobal.recipientInfo.addr, uint64(2*thousandWaves))

	sender, dapp := invokeSenderRecipientAddresses()
	fc := proto.NewFunctionCall("simpleLeaseToSender", []proto.Argument{&proto.IntegerArgument{Value: thousandWaves}})
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 0,
			},
			correctFullBalances: map[proto.WavesAddress]fullBalance{
				sender: {regular: 0, generating: 0, available: 0, effective: uint64(thousandWaves)},
				dapp:   {regular: uint64(2 * thousandWaves), generating: 0, available: uint64(thousandWaves), effective: uint64(thousandWaves)},
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

func TestApplyInvokeScriptWithLeaseAndLeaseCancel(t *testing.T) {
	to := createInvokeApplierTestObjects(t)
	to.activateFeature(t, int16(settings.RideV5))

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride5_leasing.base64", testGlobal.recipientInfo)

	var thousandWaves int64 = 1_000 * 100_000_000
	// Invoker pays only fee, but receives a leasing of 1000 waves
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, 2*invokeFee)
	to.setAndCheckInitialWavesBalance(t, testGlobal.recipientInfo.addr, uint64(2*thousandWaves))

	sender, dapp := invokeSenderRecipientAddresses()
	fc1 := proto.NewFunctionCall("simpleLeaseToSender", []proto.Argument{&proto.IntegerArgument{Value: thousandWaves}})
	tx := createInvokeScriptWithProofs(t, []proto.ScriptPayment{}, fc1, feeAsset, invokeFee)
	id := proto.GenerateLeaseScriptActionID(proto.NewRecipientFromAddress(sender), thousandWaves, 0, *tx.ID)
	fc2 := proto.NewFunctionCall("cancel", []proto.Argument{&proto.BinaryArgument{Value: id.Bytes()}})
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       fc1,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: invokeFee,
			},
			correctFullBalances: map[proto.WavesAddress]fullBalance{
				sender: {regular: invokeFee, generating: 0, available: invokeFee, effective: uint64(thousandWaves) + invokeFee},
				dapp:   {regular: uint64(2 * thousandWaves), generating: 0, available: uint64(thousandWaves), effective: uint64(thousandWaves)},
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       fc2,
			errorRes: false,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 0,
			},
			correctFullBalances: map[proto.WavesAddress]fullBalance{
				sender: {regular: 0, generating: 0, available: 0, effective: 0},
				dapp:   {regular: uint64(2 * thousandWaves), generating: 0, available: uint64(2 * thousandWaves), effective: uint64(2 * thousandWaves)},
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

// TODO: add test on sponsorship made by DApp: create new DApp that will issue and sponsor asset,
// test also the function call that issues and sets sponsorship in one turn.

// TODO: add test on impossibility of sponsorship of smart asset using DApp: issue smart asset with simple script using
// usual transaction and then try to set sponsorship using invoke.

func TestFailRejectOnThrow(t *testing.T) {
	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		let m = base64'REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU='
		let s = base64'cSsxjrYkwfagdcwmA+5emRGspA6132BE/zU/QiG0pXOcaJCFE/DQaz0zPFUv/+D4BBdTx/7T/fUKFA4b3oU9KQ3RvUWaUGruwURsQ10rbmVleQdh8eODSuW38r9Vf2n/qq6VvE/2LBTM8Kamd3/czE/5RAJyCcywFmOKMKkkV96asZlb/bBeBtRSz8ZDpbyGbjm2k/cC5sxuEYgR6X1veH0wmANIsrM04+Dj6AZ4LtpUfG7hNCDUpiONmeO5KpBGvN+3bHwxuNXz311CtpJZcsr5ONvtD4l7vPv7ggQB+C1x9VvZXuJaieyk8Gm5F4oGXXfgmKsve6vAlfonpl4pmg=='
		let pk = base64'MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAkDg8m0bCDX7fTbBlHZm+BZIHVOfC2I4klRbjSqwFi/eCdfhGjYRYvu/frpSO0LIm0beKOUvwat6DY4dEhNt2PW3UeQvT2udRQ9VBcpwaJlLreCr837sn4fa9UG9FQFaGofSww1O9eBBjwMXeZr1jOzR9RBIwoL1TQkIkZGaDXRltEaMxtNnzotPfF3vGIZZuZX4CjiitHaSC0zlmQrEL3BDqqoLwo3jq8U3Zz8XUMyQElwufGRbZqdFCeiIs/EoHiJm8q8CVExRoxB0H/vE2uDFK/OXLGTgfwnDlrCa/qGt9Zsb8raUSz9IIHx72XB+kOXTt/GOuW7x2dJvTJIqKTwIDAQAB'

		func produceThrow(msg: String) = throw(msg)

		@Callable(i)
		func heavyDirectThrow() = {
		  strict r1 = rsaVerify(SHA3512, m , s, pk)
		  strict r2 = rsaVerify(SHA3512, m , s, pk)
		  if r1 || r2 then throw("from heavyDirectThrow") else []
		}

		@Callable(i)
		func heavyIndirectThrow() = {
		  strict r1 = rsaVerify(SHA3512, m , s, pk)
		  strict r2 = rsaVerify(SHA3512, m , s, pk)
		  if r1 || r2 then produceThrow("from heavyIndirectThrow") else []
		}

		@Callable(i)
		func lightDirectThrow() = {
		  strict r = rsaVerify_16Kb(SHA3512, m , s, pk)
		  if r then throw("from lightDirectThrow") else []
		}

		@Callable(i)
		func lightIndirectThrow() = {
		  strict r = rsaVerify_16Kb(SHA3512, m , s, pk)
		  if r then produceThrow("from lightIndirectThrow") else []
		}
	*/

	to := createInvokeApplierTestObjects(t)

	info := to.fallibleValidationParams(t)
	info.acceptFailed = true
	info.blockV5Activated = true
	info.rideV5Activated = true
	info.checkerInfo.blockchainHeight = 2_800_000

	to.setDApp(t, "ride5_fail_on_throw.base64", testGlobal.recipientInfo)
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, invokeFee*3)

	sender, dapp := invokeSenderRecipientAddresses()
	heavyDirectThrow := proto.NewFunctionCall("heavyDirectThrow", []proto.Argument{})
	heavyIndirectThrow := proto.NewFunctionCall("heavyIndirectThrow", []proto.Argument{})
	lightDirectThrow := proto.NewFunctionCall("lightDirectThrow", []proto.Argument{})
	lightIndirectThrow := proto.NewFunctionCall("lightIndirectThrow", []proto.Argument{})
	tests := []invokeApplierTestData{
		{
			payments: []proto.ScriptPayment{},
			fc:       heavyDirectThrow,
			errorRes: false,
			failRes:  true,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: invokeFee * 2,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       heavyIndirectThrow,
			errorRes: false,
			failRes:  true,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: invokeFee,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       lightDirectThrow,
			errorRes: true,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 0,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
		{
			payments: []proto.ScriptPayment{},
			fc:       lightIndirectThrow,
			errorRes: true,
			failRes:  false,
			correctBalances: map[rcpAsset]uint64{
				{sender, nil}: 0,
				{dapp, nil}:   0,
			},
			correctAddrs: []proto.WavesAddress{
				testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr,
			},
			info: info,
		},
	}
	for _, tc := range tests {
		tc.applyTestWithCleanup(t, to)
	}
}

// TODO: check correctness of the test, check TODOs
func TestIssuesInInvokes(t *testing.T) {
	to := createInvokeApplierTestObjects(t)
	to.activateFeature(t, int16(settings.RideV5))

	sender := testGlobal.senderInfo  // 3P3p1SmQq78f1wf8mzUBr5BYWfxcwQJ4Fcz
	dApp := testGlobal.recipientInfo // 3P2S4mBei2JfbpjiAnC5ssdjnVC8hG21yti

	info := to.fallibleValidationParams(t)
	to.setDApp(t, "ride5_issues_in_invokes.base64", dApp)

	var payments []proto.ScriptPayment // empty payments
	functionCall := proto.NewFunctionCall("call", []proto.Argument{})
	tx := createInvokeScriptWithProofs(t, payments, functionCall, feeAsset, invokeFee)

	expectedIssueActionsOrder := []*proto.IssueScriptAction{
		{Sender: &dApp.pk, Name: "FirstIssue", Description: "first issue", Quantity: 42, Decimals: 5, Reissuable: true, Script: nil, Nonce: 0},
		{Sender: &dApp.pk, Name: "CatCoin", Description: "kitty", Quantity: 1, Decimals: 0, Reissuable: false, Script: nil, Nonce: 0}, // nft
		{Sender: &dApp.pk, Name: "PugCoin", Description: "pug", Quantity: 1, Decimals: 0, Reissuable: false, Script: nil, Nonce: 0},   // nft
		{Sender: &dApp.pk, Name: "ParrotCoin", Description: "parrots", Quantity: 10000, Decimals: 1, Reissuable: true, Script: nil, Nonce: 0},
		{Sender: &dApp.pk, Name: "OneMoreAsset", Description: "one more asset", Quantity: 42, Decimals: 5, Reissuable: true, Script: nil, Nonce: 0},
		{Sender: &dApp.pk, Name: "Asset1", Description: "just an asset", Quantity: 100500, Decimals: 2, Reissuable: true, Script: nil, Nonce: 0},
		{Sender: &dApp.pk, Name: "Asset2", Description: "just an asset", Quantity: 100100, Decimals: 3, Reissuable: true, Script: nil, Nonce: 0},
	}
	assetIDToSequenceInBlock := make(map[crypto.Digest]uint32, len(expectedIssueActionsOrder))
	correctBalances := map[rcpAsset]uint64{
		{sender.addr, nil}: 0,
		{dApp.addr, nil}:   0,
	}
	for i := range expectedIssueActionsOrder {
		a := expectedIssueActionsOrder[i]
		a.ID = proto.GenerateIssueScriptActionID(a.Name, a.Description, int64(a.Decimals), a.Quantity, a.Reissuable, a.Nonce, *tx.ID)
		assetIDToSequenceInBlock[a.ID] = uint32(i + 1)
		correctBalances[rcpAsset{dApp.addr, &a.ID}] = uint64(a.Quantity)
	}

	const waves = 0                                                               // TODO: should be == 100_000_000
	senderBalance := invokeFee + uint64((len(expectedIssueActionsOrder)-2)*waves) // TODO: why is it enough to perform invoke with issues?
	to.setAndCheckInitialWavesBalance(t, sender.addr, senderBalance)
	to.setAndCheckInitialWavesBalance(t, dApp.addr, 0)

	testData := invokeApplierTestData{
		payments:        payments,
		fc:              functionCall,
		errorRes:        false,
		failRes:         false,
		correctBalances: correctBalances,
		correctAddrs:    []proto.WavesAddress{sender.addr, dApp.addr},
		info:            info,
	}
	testData.applyTest(t, to)

	for i, action := range expectedIssueActionsOrder {
		ai, err := to.state.EnrichedFullAssetInfo(proto.AssetIDFromDigest(action.ID))
		require.NoError(t, err)
		sequenceInBlock := uint32(i + 1)
		ai.SequenceInBlock = sequenceInBlock // sequence in block is not set in invoke applier anymore, it's set in
		// snapshot applier
		assert.Equal(t, sequenceInBlock, ai.SequenceInBlock, "invalid SequenceInBlock for asset %q", ai.Name)
		assert.Equal(t, info.blockInfo.Height, ai.IssueHeight, "invalid IssueHeight for asset %q", ai.Name)
	}
}
