package state

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/compiler"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	invokeExecutionTestWorkers  = 4
	invokeExecutionTestAccounts = 6
	// invokeExecutionTestScriptThis is replaced by the address of the dApp in the scripts of the tests.
	invokeExecutionTestScriptThis = "THIS"
	// invokeExecutionTestScript is the script of every dApp of the tests.
	//   - put and inc write the data of the dApp, inc also reads it;
	//   - copy reads the data of another account, it fails cheaply if there is no entry, so it is rejected;
	//   - reject always fails cheaply;
	//   - heavy fails after spending more than FailFreeInvokeComplexity, so it is saved as failed;
	//   - deposit stores the balance of the dApp, which includes the attached payment.
	invokeExecutionTestScript = `
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func put(key: String, value: Int) = [IntegerEntry(key, value)]

		@Callable(i)
		func inc(key: String) = [IntegerEntry(key, valueOrElse(getInteger(this, key), 0) + 1)]

		@Callable(i)
		func copy(from: String, key: String) =
			[IntegerEntry(key, getIntegerValue(addressFromStringValue(from), key))]

		@Callable(i)
		func reject() = throw("cheap failure")

		@Callable(i)
		func heavy(key: String) = {
			let b = toBytes(key)
			if (sigVerify(b, b, b) || sigVerify(b, b, b) || sigVerify(b, b, b) ||
				sigVerify(b, b, b) || sigVerify(b, b, b) || sigVerify(b, b, b))
			then []
			else throw("heavy failure")
		}

		@Callable(i)
		func deposit(key: String) = [IntegerEntry(key, wavesBalance(this).available)]`

	// invokeExecutionTestScriptV3 is the script with the same functions for the time before BlockV5.
	invokeExecutionTestScriptV3 = `
		{-# STDLIB_VERSION 3 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let self = Address(base58'THIS')

		@Callable(i)
		func put(key: String, value: Int) = WriteSet([DataEntry(key, value)])

		@Callable(i)
		func inc(key: String) = {
			let v = match getInteger(self, key) {
				case x: Int => x
				case _ => 0
			}
			WriteSet([DataEntry(key, v + 1)])
		}

		@Callable(i)
		func copy(from: String, key: String) =
			WriteSet([DataEntry(key, getIntegerValue(addressFromStringValue(from), key))])

		@Callable(i)
		func reject() = throw("cheap failure")

		@Callable(i)
		func heavy(key: String) = {
			let b = toBytes(key)
			if (sigVerify(b, b, b) || sigVerify(b, b, b) || sigVerify(b, b, b) ||
				sigVerify(b, b, b) || sigVerify(b, b, b) || sigVerify(b, b, b))
			then WriteSet([])
			else throw("heavy failure")
		}

		@Callable(i)
		func deposit(key: String) = WriteSet([DataEntry(key, wavesBalance(self))])`
)

// invokeExecutionRegime is the set of rules the blocks of the tests are applied by.
type invokeExecutionRegime struct {
	name string
	// features are activated in addition to NG, SmartAccounts and Ride4DApps.
	features []settings.Feature
	script   string
}

func (r invokeExecutionRegime) activated(feature settings.Feature) bool {
	return slices.Contains(r.features, feature)
}

// invokeExecutionDefaultRegime is the regime of the most of the tests.
func invokeExecutionDefaultRegime() invokeExecutionRegime {
	return invokeExecutionRegime{
		name:     "RideV5",
		features: []settings.Feature{settings.BlockV5, settings.RideV5, settings.SmartAccountTrading},
		script:   invokeExecutionTestScript,
	}
}

// invokeExecutionTestObjects is the state with several funded senders and several dApps with the same script.
// Transactions are applied to it by the transactions loop of the block appender.
type invokeExecutionTestObjects struct {
	*invokeApplierTestObjects
	regime  invokeExecutionRegime
	senders []*testWavesAddrData
	dApps   []*testWavesAddrData
	// snapshots are the snapshots of the blocks applied by applyToBothStates.
	snapshots []proto.BlockSnapshot
	// challenged makes the applied blocks the blocks with a challenged header, transactions that can not be
	// applied are elided from such blocks.
	challenged bool
	// leftResults is the number of speculative results left by the last applied block.
	leftResults int
}

func invokeExecutionTestAccount(t *testing.T, role string, i int) *testWavesAddrData {
	acc, err := newTestWavesAddrData(base58.Encode(fmt.Appendf(nil, "invoke execution test %s %d", role, i)), nil)
	require.NoError(t, err)
	return acc
}

// createInvokeExecutionTestObjects creates the state that executes invokes in parallel
// with invokeExecutionTestWorkers goroutines.
func createInvokeExecutionTestObjects(t *testing.T) *invokeExecutionTestObjects {
	return createInvokeExecutionTestObjectsOfRegime(t, invokeExecutionTestWorkers, invokeExecutionDefaultRegime())
}

func createInvokeExecutionTestObjectsOfRegime(
	t *testing.T, goroutinesNum int, regime invokeExecutionRegime,
) *invokeExecutionTestObjects {
	params := DefaultTestingStateParams()
	params.InvokeExecutionGoroutinesNum = goroutinesNum
	to := &invokeExecutionTestObjects{
		invokeApplierTestObjects: createInvokeApplierTestObjectsWithParams(t, params),
		regime:                   regime,
	}
	require.Equal(t, goroutinesNum > 0, to.state.appender.invokeExec != nil)
	require.Equal(t, to.state.appender.invokeExec, to.state.appender.sc.invokeExec)
	for _, f := range regime.features {
		to.activateFeature(t, int16(f))
	}
	for i := range invokeExecutionTestAccounts {
		sender := invokeExecutionTestAccount(t, "sender", i)
		dApp := invokeExecutionTestAccount(t, "dApp", i)
		to.senders = append(to.senders, sender)
		to.dApps = append(to.dApps, dApp)
		src := strings.ReplaceAll(regime.script, invokeExecutionTestScriptThis, dApp.addr.String())
		scriptBytes, errs := compiler.Compile(src, false, true)
		require.NoError(t, errors.Join(errs...))
		to.setScript(t, dApp.addr, dApp.pk, scriptBytes)
		for _, acc := range []*testWavesAddrData{sender, dApp} {
			require.NoError(t, to.state.stor.balances.setWavesBalance(acc.addr.ID(),
				wavesValue{profile: balanceProfile{Balance: 1000 * invokeFee}}, blockID0))
		}
	}
	require.NoError(t, to.state.flush())
	to.state.reset()
	return to
}

// invokeExecutionTxBuilder builds the invokes, which are the same for all the states they are applied to.
type invokeExecutionTxBuilder struct {
	t         *testing.T
	timestamp uint64
	// version is the version of the built transactions, the second one is not allowed before BlockV5.
	version byte
}

func newInvokeExecutionTxBuilder(t *testing.T) *invokeExecutionTxBuilder {
	return &invokeExecutionTxBuilder{t: t, timestamp: defaultTimestamp, version: 2}
}

func (b *invokeExecutionTxBuilder) invoke(
	sender, dApp *testWavesAddrData, function string, args ...proto.Argument,
) proto.Transaction {
	b.timestamp++ // Makes transactions unique.
	return b.invokeAt(sender, dApp, b.timestamp, function, args...)
}

// invokeWithPayment builds the invoke with the payment of the given amount of Waves attached.
func (b *invokeExecutionTxBuilder) invokeWithPayment(
	sender, dApp *testWavesAddrData, amount uint64, function string, args ...proto.Argument,
) proto.Transaction {
	b.timestamp++
	payments := proto.ScriptPayments{{Amount: amount}}
	tx := proto.NewUnsignedInvokeScriptWithProofs(b.version, sender.pk, proto.NewRecipientFromAddress(dApp.addr),
		proto.NewFunctionCall(function, args), payments, feeAsset, invokeFee, b.timestamp)
	require.NoError(b.t, tx.Sign(proto.MainNetScheme, sender.sk))
	return tx
}

// invokeAt builds the invoke with the given timestamp.
func (b *invokeExecutionTxBuilder) invokeAt(
	sender, dApp *testWavesAddrData, timestamp uint64, function string, args ...proto.Argument,
) *proto.InvokeScriptWithProofs {
	tx := proto.NewUnsignedInvokeScriptWithProofs(b.version, sender.pk, proto.NewRecipientFromAddress(dApp.addr),
		proto.NewFunctionCall(function, args), nil, feeAsset, invokeFee, timestamp)
	require.NoError(b.t, tx.Sign(proto.MainNetScheme, sender.sk))
	return tx
}

// appendTxs applies the transactions as the transactions of a block and finishes the block for the executor
// the way appendBlock does.
func (to *invokeExecutionTestObjects) appendTxs(
	t *testing.T, txs []proto.Transaction,
) (proto.BlockSnapshot, crypto.Digest, error) {
	info := defaultCheckerInfo()
	info.rideV5Activated = to.regime.activated(settings.RideV5)
	info.rideV6Activated = to.regime.activated(settings.RideV6)
	cleanup := to.state.appender.txHandler.sa.SetApplierInfo(newBlockSnapshotsApplierInfo(info, proto.MainNetScheme))
	defer cleanup()
	hasher, err := newTxSnapshotHasherDefault()
	require.NoError(t, err)
	defer hasher.Release()
	block := defaultBlock()
	block.GeneratorPublicKey = testGlobal.minerInfo.pk
	if to.challenged {
		block.ChallengedHeader = &proto.ChallengedHeader{GeneratorPublicKey: testGlobal.senderInfo.pk}
	}
	params := &appendBlockParams{
		transactions:     txs,
		chans:            launchVerifier(t.Context(), 1, proto.MainNetScheme),
		block:            block,
		blockchainHeight: info.blockchainHeight,
	}
	bs, stateHash, err := to.state.appender.appendTxs(params, info, defaultBlockInfo(), crypto.Digest{}, hasher)
	require.NoError(t, params.chans.closeAndWait())
	if exec := to.state.appender.invokeExec; exec != nil && err == nil {
		to.leftResults = len(exec.speculator.results)
		exec.finishBlock(info.blockchainHeight + 1)
		exec.resetBlock()
	}
	return bs, stateHash, err
}

func (to *invokeExecutionTestObjects) counters() invokeExecutionCounters {
	return to.state.appender.invokeExec.total
}

func (to *invokeExecutionTestObjects) integerEntry(t *testing.T, acc *testWavesAddrData, key string) int64 {
	entry, err := to.state.RetrieveNewestIntegerEntry(acc.rcp, key)
	require.NoError(t, err)
	return entry.Value
}

func (to *invokeExecutionTestObjects) stats(t *testing.T, dApp *testWavesAddrData, function string) *invokeStats {
	stats, ok, err := to.state.stor.invokeStats.stats(invokeStatsKey{dApp: dApp.addr.ID(), function: function})
	require.NoError(t, err)
	require.True(t, ok, "no statistics for function %q", function)
	return stats
}

func stringArg(v string) proto.Argument { return &proto.StringArgument{Value: v} }

// invokeExecutionBlocks returns the blocks of transactions applied by the tests to the sequential and to
// the parallel states.
type invokeExecutionBlocks func(b *invokeExecutionTxBuilder, to *invokeExecutionTestObjects) [][]proto.Transaction

// applyToBothStates applies the blocks to the sequential and to the parallel states and checks that the results
// are the same. It returns the parallel state and the error of the last block. The snapshots of the applied
// blocks are put to the snapshots field of the returned state.
func applyToBothStates(
	t *testing.T, blocks invokeExecutionBlocks, prepare ...func(parallel *invokeExecutionTestObjects),
) (*invokeExecutionTestObjects, error) {
	return applyBlocksToBothStates(t, invokeExecutionDefaultRegime(), false, blocks, prepare...)
}

// applyBlocksToBothStates does the job of applyToBothStates, the blocks are the blocks with a challenged header
// if challenged is set.
func applyBlocksToBothStates(
	t *testing.T, regime invokeExecutionRegime, challenged bool, blocks invokeExecutionBlocks,
	prepare ...func(parallel *invokeExecutionTestObjects),
) (*invokeExecutionTestObjects, error) {
	sequential := createInvokeExecutionTestObjectsOfRegime(t, 0, regime)
	parallel := createInvokeExecutionTestObjectsOfRegime(t, invokeExecutionTestWorkers, regime)
	for _, p := range prepare {
		p(parallel)
	}
	sequential.challenged, parallel.challenged = challenged, challenged
	var lastErr error
	builder := newInvokeExecutionTxBuilder(t)
	if !regime.activated(settings.BlockV5) {
		builder.version = 1
	}
	for i, txs := range blocks(builder, sequential) {
		seqSnapshot, seqHash, seqErr := sequential.appendTxs(t, txs)
		parSnapshot, parHash, parErr := parallel.appendTxs(t, txs)
		if seqErr != nil {
			require.Error(t, parErr, "block %d", i)
			assert.Equal(t, seqErr.Error(), parErr.Error(), "block %d", i)
		} else {
			require.NoError(t, parErr, "block %d", i)
		}
		// The order of balance snapshots of a transaction is not defined, the state hash does not depend on it.
		require.Len(t, parSnapshot.TxSnapshots, len(seqSnapshot.TxSnapshots), "block %d", i)
		for j := range seqSnapshot.TxSnapshots {
			assert.ElementsMatch(t, seqSnapshot.TxSnapshots[j], parSnapshot.TxSnapshots[j], "block %d, tx %d", i, j)
		}
		assert.Equal(t, seqHash, parHash, "block %d", i)
		// The complexity of scripts is accumulated over the blocks, it is what the limit of a block is
		// checked against.
		assert.Equal(t, sequential.state.appender.sc.getTotalComplexity(),
			parallel.state.appender.sc.getTotalComplexity(), "complexity after block %d", i)
		parallel.snapshots = append(parallel.snapshots, parSnapshot)
		lastErr = parErr
	}
	return parallel, lastErr
}

func TestParallelInvokeExecutionIndependentInvokes(t *testing.T) {
	const blocksNum = 3
	to, err := applyToBothStates(t, func(
		b *invokeExecutionTxBuilder, to *invokeExecutionTestObjects,
	) [][]proto.Transaction {
		blocks := make([][]proto.Transaction, blocksNum)
		for i := range blocks {
			for j := range to.senders { // Every sender calls its own dApp.
				blocks[i] = append(blocks[i], b.invoke(to.senders[j], to.dApps[j], "inc", stringArg("counter")))
			}
		}
		return blocks
	})
	require.NoError(t, err)
	for _, dApp := range to.dApps {
		assert.Equal(t, int64(blocksNum), to.integerEntry(t, dApp, "counter"))
	}
	// All invokes of a block are evaluated by the batch launched before the first of them, with and without
	// statistics.
	total := blocksNum * invokeExecutionTestAccounts
	assert.Equal(t, invokeExecutionCounters{speculated: total, reused: total}, to.counters())
	for i, dApp := range to.dApps {
		stats := to.stats(t, dApp, "inc")
		assert.Equal(t, uint64(blocksNum), stats.executions)
		assert.Zero(t, stats.mispredictions)
		reads, writes := instantiate(stats, to.senders[i].addr.ID(), dApp.addr.ID())
		assert.True(t, reads.has(accountDataAccessKey(dApp.addr.ID())))
		assert.True(t, writes.has(accountDataAccessKey(dApp.addr.ID())))
		assert.True(t, writes.has(accountBalanceAccessKey(to.senders[i].addr.ID())))
	}
}

func TestParallelInvokeExecutionMisprediction(t *testing.T) {
	// In every block the first invoke writes the entry read by the second one from the data of another dApp.
	// The evaluation of the second invoke over the state before the first one fails or gives the stale value.
	to, err := applyToBothStates(t, func(
		b *invokeExecutionTxBuilder, to *invokeExecutionTestObjects,
	) [][]proto.Transaction {
		source := stringArg(to.dApps[0].addr.String())
		blocks := make([][]proto.Transaction, 2)
		for i := range blocks {
			blocks[i] = []proto.Transaction{
				b.invoke(to.senders[0], to.dApps[0], "put", stringArg("k"), proto.NewIntegerArgument(int64(i+5))),
				b.invoke(to.senders[1], to.dApps[1], "copy", source, stringArg("k")),
				b.invoke(to.senders[2], to.dApps[2], "inc", stringArg("counter")),
			}
		}
		return blocks
	})
	require.NoError(t, err)
	assert.Equal(t, int64(6), to.integerEntry(t, to.dApps[0], "k"))
	assert.Equal(t, int64(6), to.integerEntry(t, to.dApps[1], "k"))
	assert.Equal(t, int64(2), to.integerEntry(t, to.dApps[2], "counter"))

	// Block 0, no statistics: all three invokes are speculated together, the result of copy is invalidated by
	// put, so copy is evaluated again in order.
	// Block 1: statistics tell that copy depends on put, so only put and inc are speculated together and copy
	// is evaluated in order without a misprediction.
	assert.Equal(t, invokeExecutionCounters{speculated: 5, reused: 4, mispredicted: 1, inline: 2}, to.counters())

	stats := to.stats(t, to.dApps[1], "copy")
	assert.Equal(t, uint64(2), stats.executions)
	assert.Equal(t, uint64(1), stats.mispredictions)
	reads, _ := instantiate(stats, to.senders[1].addr.ID(), to.dApps[1].addr.ID())
	assert.True(t, reads.has(accountDataAccessKey(to.dApps[0].addr.ID())),
		"statistics must have the data of the source dApp among the reads of copy")
	assert.Zero(t, to.stats(t, to.dApps[0], "put").mispredictions)
}

func TestParallelInvokeExecutionSameDApp(t *testing.T) {
	// Different senders call the same function of the same dApp: every invoke depends on the previous one.
	const blocksNum = 2
	to, err := applyToBothStates(t, func(
		b *invokeExecutionTxBuilder, to *invokeExecutionTestObjects,
	) [][]proto.Transaction {
		blocks := make([][]proto.Transaction, blocksNum)
		for i := range blocks {
			for _, sender := range to.senders {
				blocks[i] = append(blocks[i], b.invoke(sender, to.dApps[0], "inc", stringArg("counter")))
			}
		}
		return blocks
	})
	require.NoError(t, err)
	assert.Equal(t, int64(blocksNum*invokeExecutionTestAccounts), to.integerEntry(t, to.dApps[0], "counter"))
	c := to.counters()
	assert.Equal(t, blocksNum*invokeExecutionTestAccounts, c.reused+c.inline, "every invoke is evaluated once")
	assert.Equal(t, c.speculated, c.reused+c.mispredicted, "every speculative result is either used or dropped")
	assert.Positive(t, c.mispredicted, "without statistics the invokes are speculated together")
}

func TestParallelInvokeExecutionFailedInvoke(t *testing.T) {
	// Failed invokes are saved to the block, the invokes after them are not affected.
	const failedIndex = 1
	to, err := applyToBothStates(t, func(
		b *invokeExecutionTxBuilder, to *invokeExecutionTestObjects,
	) [][]proto.Transaction {
		blocks := make([][]proto.Transaction, 2)
		for i := range blocks {
			blocks[i] = []proto.Transaction{
				b.invoke(to.senders[0], to.dApps[0], "inc", stringArg("counter")),
				b.invoke(to.senders[1], to.dApps[1], "heavy", stringArg("k")),
				b.invoke(to.senders[2], to.dApps[1], "inc", stringArg("counter")),
				b.invoke(to.senders[3], to.dApps[3], "inc", stringArg("counter")),
			}
		}
		return blocks
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), to.integerEntry(t, to.dApps[1], "counter"))
	assert.Equal(t, uint64(2), to.stats(t, to.dApps[1], "heavy").executions)
	c := to.counters()
	assert.Equal(t, 8, c.reused+c.inline)
	assert.Positive(t, c.reused)
	require.Len(t, to.snapshots, 2)
	for _, bs := range to.snapshots {
		for i, txSnapshot := range bs.TxSnapshots {
			expected := proto.TransactionSucceeded
			if i == failedIndex {
				expected = proto.TransactionFailed
			}
			assert.Contains(t, txSnapshot, &proto.TransactionStatusSnapshot{Status: expected}, "transaction %d", i)
		}
	}
}

func TestParallelInvokeExecutionRejectedInvoke(t *testing.T) {
	for _, test := range []struct {
		name     string
		function string
		args     []proto.Argument
	}{
		{name: "script error", function: "reject"},
		{name: "unknown function", function: "unknown"},
		{name: "missing entry", function: "copy", args: []proto.Argument{
			stringArg(testGlobal.minerInfo.addr.String()), stringArg("k"),
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			to, err := applyToBothStates(t, func(
				b *invokeExecutionTxBuilder, to *invokeExecutionTestObjects,
			) [][]proto.Transaction {
				return [][]proto.Transaction{{
					b.invoke(to.senders[0], to.dApps[0], "inc", stringArg("counter")),
					b.invoke(to.senders[1], to.dApps[1], test.function, test.args...),
					b.invoke(to.senders[2], to.dApps[2], "inc", stringArg("counter")),
				}}
			})
			require.Error(t, err)
			// The block is invalid. The state drops everything left of it the way it does after a failure.
			exec := to.state.appender.invokeExec
			assert.NotEmpty(t, exec.speculator.results, "the result of the last invoke was not used")
			to.state.reset()
			assert.Empty(t, exec.speculator.results)
			assert.Empty(t, exec.speculator.lastWriter)
			assert.Nil(t, exec.current)
		})
	}
}

// TestParallelInvokeExecutionElidedInvokes checks the block with a challenged header: transactions that can not
// be applied are elided from it and the application of the block goes on.
func TestParallelInvokeExecutionElidedInvokes(t *testing.T) {
	const (
		blocksNum    = 2
		rejectedIdx  = 1
		earlyIdx     = 3
		duplicateIdx = 5
	)
	elided := map[int]bool{rejectedIdx: true, earlyIdx: true, duplicateIdx: true}
	to, err := applyBlocksToBothStates(t, invokeExecutionDefaultRegime(), true, func(
		b *invokeExecutionTxBuilder, to *invokeExecutionTestObjects,
	) [][]proto.Transaction {
		blocks := make([][]proto.Transaction, blocksNum)
		for i := range blocks {
			first := b.invoke(to.senders[0], to.dApps[0], "inc", stringArg("counter"))
			blocks[i] = []proto.Transaction{
				first,
				// The script fails cheaply, so the invoke is rejected after the evaluation of the script.
				b.invoke(to.senders[1], to.dApps[1], "reject"),
				b.invoke(to.senders[2], to.dApps[2], "inc", stringArg("counter")),
				// The invoke is too old, so it is rejected before the script is called. Its script is evaluated
				// speculatively anyway, the result has to be dropped.
				b.invokeAt(to.senders[3], to.dApps[3], uint64(i+1), "inc", stringArg("counter")),
				b.invoke(to.senders[4], to.dApps[4], "inc", stringArg("counter")),
				first, // The duplicate is rejected before the script is called as well.
				// The invoke after elided ones reads what was written before them.
				b.invoke(to.senders[5], to.dApps[0], "inc", stringArg("counter")),
			}
		}
		return blocks
	})
	require.NoError(t, err)
	require.Len(t, to.snapshots, blocksNum)
	for _, bs := range to.snapshots {
		for i, txSnapshot := range bs.TxSnapshots {
			if elided[i] {
				assert.Equal(t, []proto.AtomicSnapshot{&proto.TransactionStatusSnapshot{Status: proto.TransactionElided}},
					txSnapshot, "transaction %d", i)
			} else {
				assert.Contains(t, txSnapshot, &proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
					"transaction %d", i)
			}
		}
	}
	assert.Equal(t, int64(2*blocksNum), to.integerEntry(t, to.dApps[0], "counter"))
	assert.Equal(t, int64(blocksNum), to.integerEntry(t, to.dApps[2], "counter"))
	assert.Equal(t, int64(blocksNum), to.integerEntry(t, to.dApps[4], "counter"))
	_, err = to.state.RetrieveNewestIntegerEntry(to.dApps[3].rcp, "counter")
	require.Error(t, err, "nothing of an elided transaction is applied")

	// An elided transaction leaves the appender between transactions, so the invokes after it are speculated.
	assert.True(t, to.state.appender.betweenTransactions())
	assert.Zero(t, to.leftResults, "the result of an invoke rejected before the script is called must be dropped")
	c := to.counters()
	// The scripts of five invokes of a block are called: four applied ones and the one rejected by its script.
	assert.Equal(t, 5*blocksNum, c.reused+c.inline)
	assert.Positive(t, c.reused)
	// The results of the invokes rejected before the script is called are neither used nor counted as mispredicted.
	assert.GreaterOrEqual(t, c.speculated, c.reused+c.mispredicted+blocksNum)

	// Statistics are collected only for the applied invokes.
	assert.Equal(t, uint64(2*blocksNum), to.stats(t, to.dApps[0], "inc").executions)
	for _, key := range []invokeStatsKey{
		{dApp: to.dApps[1].addr.ID(), function: "reject"},
		{dApp: to.dApps[3].addr.ID(), function: "inc"},
	} {
		_, ok, sErr := to.state.stor.invokeStats.stats(key)
		require.NoError(t, sErr)
		assert.False(t, ok, "statistics of elided invoke of %q", key.function)
	}
}

// invokeExecutionTestBatch launches the batch of independent invokes of different dApps the way the block appender
// does before the first of them, and returns the invokes with the parameters they are applied with.
func invokeExecutionTestBatch(
	t *testing.T, to *invokeExecutionTestObjects,
) ([]proto.Transaction, []*proto.InvokeScriptWithProofs, *appendTxParams) {
	const invokesNum = 3
	b := newInvokeExecutionTxBuilder(t)
	txs := make([]proto.Transaction, invokesNum)
	invokes := make([]*proto.InvokeScriptWithProofs, invokesNum)
	for i := range invokes {
		invokes[i] = b.invokeAt(to.senders[i], to.dApps[i], defaultTimestamp+uint64(i), "inc", stringArg("counter"))
		txs[i] = invokes[i]
	}
	params := defaultAppendTxParams()
	params.rideV5Activated, params.blockV5Activated = true, true
	exec := to.state.appender.invokeExec
	exec.beforeTx(txs, 0, params)
	require.Len(t, exec.speculator.results, len(txs))
	require.Equal(t, len(txs), exec.counters.speculated)
	t.Cleanup(exec.resetBlock)
	return txs, invokes, params
}

// TestParallelInvokeExecutionUnknownWriteSet checks that nothing evaluated speculatively is used after
// a transaction which write set can not be obtained.
func TestParallelInvokeExecutionUnknownWriteSet(t *testing.T) {
	to := createInvokeExecutionTestObjects(t)
	exec := to.state.appender.invokeExec
	_, invokes, _ := invokeExecutionTestBatch(t, to)

	// The snapshot of the status belongs to a transaction, without the ID its key is unknown.
	snapshot := txSnapshot{regular: []proto.AtomicSnapshot{
		&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
	}}
	_, err := writeSetFromSnapshot(nil, snapshot, proto.MainNetScheme)
	require.Error(t, err)
	exec.afterTx(0, nil, snapshot, true)

	assert.Empty(t, exec.speculator.results)
	assert.Nil(t, exec.current)
	for _, tx := range invokes {
		assert.False(t, exec.speculator.hasValid(*tx.ID))
	}
	_, ok, err := to.state.stor.invokeStats.stats(invokeStatsKey{dApp: to.dApps[0].addr.ID(), function: "inc"})
	require.NoError(t, err)
	assert.False(t, ok, "statistics must not be updated with an unknown write set")
}

// TestParallelInvokeExecutionScriptAddressMismatch checks that a speculative result is not used if it was
// evaluated for another dApp, the script is evaluated in order then.
func TestParallelInvokeExecutionScriptAddressMismatch(t *testing.T) {
	to := createInvokeExecutionTestObjects(t)
	exec := to.state.appender.invokeExec
	txs, invokes, params := invokeExecutionTestBatch(t, to)
	tx := invokes[0]
	dApp := to.dApps[0].addr
	tree, err := to.state.stor.scriptsStorage.newestScriptByAddr(dApp)
	require.NoError(t, err)
	info := &fallibleValidationParams{appendTxParams: params}

	speculative := exec.speculator.results[*tx.ID]
	require.Equal(t, dApp, speculative.scriptAddress)
	expected := speculative.evaluation
	speculative.scriptAddress = to.dApps[1].addr
	speculative.evaluation = invokeEvaluation{} // The result that must not be used.

	ev, ok, err := exec.evaluation(tree, tx, info, dApp)
	require.NoError(t, err)
	require.True(t, ok)
	assertEqualInvokeEvaluations(t, expected, ev)
	assert.Equal(t, invokeExecutionCounters{speculated: len(txs), mispredicted: 1, inline: 1}, exec.counters)
	require.NotNil(t, exec.current)
	assert.True(t, exec.current.mispredicted)
	assert.True(t, exec.current.reads.has(accountDataAccessKey(dApp.ID())))
	assert.NotContains(t, exec.speculator.results, *tx.ID)

	// The evaluation is given only once, and only for the invoke being applied.
	_, ok, err = exec.evaluation(tree, tx, info, dApp)
	require.NoError(t, err)
	assert.False(t, ok)
	_, ok, err = exec.evaluation(tree, invokes[1], info, to.dApps[1].addr)
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestParallelInvokeExecutionPredictionsCache checks that the accesses of the transactions of a block are
// predicted once and are predicted again only when the predictions may change.
func TestParallelInvokeExecutionPredictionsCache(t *testing.T) {
	to := createInvokeExecutionTestObjects(t)
	exec := to.state.appender.invokeExec
	txs, invokes, _ := invokeExecutionTestBatch(t, to)
	samePrediction := func(a, b predictedAccess) bool {
		// Access sets are maps, the sets of the same prediction are the same maps.
		a.reads.add(accessKey{})
		defer delete(a.reads, accessKey{})
		return b.reads.has(accessKey{})
	}
	require.Len(t, exec.predictions, len(txs))
	cached := make([]predictedAccess, len(txs))
	for i, tx := range txs {
		require.Equal(t, tx, exec.predictions[i].tx)
		cached[i] = exec.predictions[i].access
		assert.True(t, cached[i].speculatable)
		assert.True(t, samePrediction(cached[i], exec.predict(txs, i)), "transaction %d", i)
	}

	t.Run("another transaction at the same index", func(t *testing.T) {
		other := []proto.Transaction{txs[1], txs[0], txs[2]}
		assert.False(t, samePrediction(cached[0], exec.predict(other, 0)))
		assert.Equal(t, other[0], exec.predictions[0].tx)
		assert.True(t, samePrediction(cached[2], exec.predict(other, 2)))
	})
	t.Run("another block", func(t *testing.T) {
		assert.False(t, samePrediction(cached[0], exec.predict(txs[:2], 0)))
		assert.Len(t, exec.predictions, 2)
	})
	t.Run("changed statistics", func(t *testing.T) {
		first := exec.predict(txs, 0)
		sender, dApp := to.senders[0].addr.ID(), to.dApps[0].addr.ID()
		key := invokeStatsKey{dApp: dApp, function: "inc"}
		other := accountDataAccessKey(to.dApps[5].addr.ID())
		reads := newAccessSet(other)
		require.NoError(t, exec.stats.update(key, sender, reads, newAccessSet(), false))
		second := exec.predict(txs, 0)
		assert.False(t, samePrediction(first, second))
		assert.True(t, second.reads.has(other), "the prediction must be made from the new statistics")
		// The same execution once again changes nothing.
		require.NoError(t, exec.stats.update(key, sender, reads, newAccessSet(), false))
		assert.True(t, samePrediction(second, exec.predict(txs, 0)))
		exec.stats.reset()
		assert.False(t, samePrediction(second, exec.predict(txs, 0)))
	})
	t.Run("created alias", func(t *testing.T) {
		first := exec.predict(txs, 0)
		id := *invokes[0].ID
		alias := *proto.NewAlias(proto.MainNetScheme, "predictions")
		exec.afterTx(0, id.Bytes(), txSnapshot{regular: []proto.AtomicSnapshot{
			&proto.AliasSnapshot{Address: to.senders[0].addr, Alias: alias.Alias},
		}}, true)
		assert.Nil(t, exec.predictions)
		assert.False(t, samePrediction(first, exec.predict(txs, 0)))
	})
	t.Run("end of the block", func(t *testing.T) {
		exec.predict(txs, 0)
		exec.resetBlock()
		assert.Nil(t, exec.predictions)
	})
}

// TestParallelInvokeExecutionBetweenTransactions checks that a batch is launched only when the appender holds
// no changes of a partially applied transaction.
func TestParallelInvokeExecutionBetweenTransactions(t *testing.T) {
	to := createInvokeExecutionTestObjects(t)
	a := to.state.appender
	exec := a.invokeExec
	b := newInvokeExecutionTxBuilder(t)
	txs := []proto.Transaction{
		b.invoke(to.senders[0], to.dApps[0], "inc", stringArg("counter")),
		b.invoke(to.senders[1], to.dApps[1], "inc", stringArg("counter")),
	}
	params := defaultAppendTxParams()
	params.rideV5Activated, params.blockV5Activated = true, true

	require.True(t, a.betweenTransactions())
	speculated := 0
	for name, dirty := range map[string]func(){
		"balance diff": func() { to.setInitialWavesBalance(t, to.senders[0].addr, 1) },
		"invoke balance diff": func() {
			diff := newTxDiff()
			key := wavesBalanceKey{to.senders[0].addr.ID()}
			require.NoError(t, diff.appendBalanceDiff(key.bytes(), newBalanceDiff(1, 0, 0, 0, false)))
			require.NoError(t, a.diffStorInvoke.saveTxDiff(diff))
		},
		"uncertain data entry": func() {
			entry := &proto.IntegerDataEntry{Key: "k", Value: 1}
			a.stor.accountsDataStor.appendEntryUncertain(to.dApps[0].addr, entry)
		},
		"uncertain asset": func() {
			a.stor.assets.issueAssetUncertain(proto.AssetIDFromDigest(testGlobal.asset0.assetID), &assetInfo{})
		},
		"uncertain asset script": func() {
			require.NoError(t, a.stor.scriptsStorage.setAssetScriptUncertain(
				testGlobal.asset0.assetID, testGlobal.scriptBytes, testGlobal.senderInfo.pk))
		},
		"uncertain lease": func() {
			a.stor.leases.addLeasingUncertain(testGlobal.asset0.assetID, &leasing{})
		},
		"uncertain sponsorship": func() {
			a.stor.sponsoredAssets.sponsorAssetUncertain(testGlobal.asset0.assetID, 1)
		},
	} {
		// The results evaluated while the state was between transactions.
		exec.beforeTx(txs, 0, params)
		require.Len(t, exec.speculator.results, len(txs), name)
		speculated += len(txs)
		dirty()
		assert.False(t, a.betweenTransactions(), name)
		// Nothing is evaluated over the state with partial changes, and the results evaluated before are not
		// used: the evaluation in order sees these changes.
		exec.beforeTx(txs, 0, params)
		assert.Empty(t, exec.speculator.results, name)
		assert.Equal(t, speculated, exec.counters.speculated, name)
		require.NotNil(t, exec.current, name)
		// Back to the state between transactions.
		a.diffStor.reset()
		a.diffStorInvoke.invokeDiffsStor.reset()
		a.stor.dropUncertain()
		require.True(t, a.betweenTransactions(), name)
	}

	exec.beforeTx(txs, 0, params)
	assert.Len(t, exec.speculator.results, len(txs))
	assert.Equal(t, speculated+len(txs), exec.counters.speculated)
	exec.resetBlock()
}

// TestParallelInvokeExecutionOutsideOfBlock checks that an invoke applied not by the transactions loop of a block,
// as it is on validation of UTX, is evaluated the usual way.
func TestParallelInvokeExecutionOutsideOfBlock(t *testing.T) {
	to := createInvokeExecutionTestObjects(t)
	params := defaultAppendTxParams()
	params.checkerInfo.rideV5Activated = true
	params.rideV5Activated, params.blockV5Activated, params.acceptFailed = true, true, true
	params.currentMinerPK = testGlobal.minerInfo.pk
	params.chans = launchVerifier(t.Context(), 1, proto.MainNetScheme)
	defer func() { require.NoError(t, params.chans.closeAndWait()) }()
	cleanup := to.state.appender.txHandler.sa.SetApplierInfo(
		newBlockSnapshotsApplierInfo(params.checkerInfo, proto.MainNetScheme))
	defer cleanup()

	tx := newInvokeExecutionTxBuilder(t).invoke(to.senders[0], to.dApps[0], "inc", stringArg("counter"))
	_, err := to.state.appender.appendTx(tx, params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), to.integerEntry(t, to.dApps[0], "counter"))
	exec := to.state.appender.invokeExec
	assert.True(t, exec.counters.isZero())
	_, ok, err := to.state.stor.invokeStats.stats(invokeStatsKey{dApp: to.dApps[0].addr.ID(), function: "inc"})
	require.NoError(t, err)
	assert.False(t, ok, "statistics are collected only on application of blocks")
}

// TestParallelInvokeExecutionRegimes checks the equivalence of the sequential and the parallel execution under
// the rules of different times: before BlockV5 a failure of a script makes the block invalid, before RideV5
// the complexity of a script is taken from its estimation, after RideV6 it is limited for a block.
func TestParallelInvokeExecutionRegimes(t *testing.T) {
	regimes := []invokeExecutionRegime{
		{name: "Ride4DApps", features: nil, script: invokeExecutionTestScriptV3},
		{
			name:     "BlockV5",
			features: []settings.Feature{settings.BlockV5, settings.SmartAccountTrading},
			script:   strings.Replace(invokeExecutionTestScript, "STDLIB_VERSION 5", "STDLIB_VERSION 4", 1),
		},
		invokeExecutionDefaultRegime(),
		{
			name: "RideV6",
			features: []settings.Feature{
				settings.BlockV5, settings.RideV5, settings.RideV6, settings.SmartAccountTrading,
			},
			script: strings.Replace(invokeExecutionTestScript, "STDLIB_VERSION 5", "STDLIB_VERSION 6", 1),
		},
	}
	const payment = 1000
	failingArgs := map[string][]proto.Argument{"reject": nil, "heavy": {stringArg("k")}}
	for _, regime := range regimes {
		for _, failing := range []string{"reject", "heavy"} {
			t.Run(regime.name+"/"+failing, func(t *testing.T) {
				to, err := applyBlocksToBothStates(t, regime, false, func(
					b *invokeExecutionTxBuilder, to *invokeExecutionTestObjects,
				) [][]proto.Transaction {
					source := stringArg(to.dApps[0].addr.String())
					var blocks [][]proto.Transaction
					for i := range 2 { // Without statistics and with them.
						blocks = append(blocks, []proto.Transaction{
							b.invoke(to.senders[0], to.dApps[0], "put", stringArg("k"), proto.NewIntegerArgument(int64(i))),
							b.invoke(to.senders[1], to.dApps[1], "copy", source, stringArg("k")),
							b.invoke(to.senders[2], to.dApps[2], "inc", stringArg("counter")),
							b.invokeWithPayment(to.senders[3], to.dApps[3], payment, "deposit", stringArg("balance")),
							b.invokeWithPayment(to.senders[4], to.dApps[3], payment, "deposit", stringArg("balance")),
							b.invoke(to.senders[5], to.dApps[5], "inc", stringArg("counter")),
						})
					}
					// The failure comes last, because before BlockV5 it fails the block.
					return append(blocks, []proto.Transaction{
						b.invoke(to.senders[0], to.dApps[0], "inc", stringArg("counter")),
						b.invoke(to.senders[1], to.dApps[1], failing, failingArgs[failing]...),
						b.invoke(to.senders[2], to.dApps[2], "inc", stringArg("counter")),
					})
				})
				c := to.counters().add(to.state.appender.invokeExec.counters) // The block that failed is not finished.
				assert.Positive(t, c.reused, "no speculative result was used")
				assert.Equal(t, int64(1), to.integerEntry(t, to.dApps[1], "k"))
				// Whether the script sees the attached payment in the balance of the dApp depends on the regime.
				assert.Positive(t, to.integerEntry(t, to.dApps[3], "balance"))
				saved := failing == "heavy" && regime.activated(settings.BlockV5)
				if !saved {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Contains(t, to.snapshots[2].TxSnapshots[1],
					&proto.TransactionStatusSnapshot{Status: proto.TransactionFailed})
			})
		}
	}
}

// failingInvokeStatsDB is the database that fails to read.
type failingInvokeStatsDB struct {
	keyvalue.KeyValue
}

func (failingInvokeStatsDB) Get([]byte) ([]byte, error) { return nil, errors.New("database is broken") }

// TestParallelInvokeExecutionStatisticsFailure checks that statistics are advisory: the application of blocks
// does not depend on the failures to read them.
func TestParallelInvokeExecutionStatisticsFailure(t *testing.T) {
	const blocksNum = 2
	var broken *invokeExecutionTestObjects
	to, err := applyToBothStates(t, func(
		b *invokeExecutionTxBuilder, to *invokeExecutionTestObjects,
	) [][]proto.Transaction {
		blocks := make([][]proto.Transaction, blocksNum)
		for i := range blocks {
			for j := range to.senders {
				blocks[i] = append(blocks[i], b.invoke(to.senders[j], to.dApps[0], "inc", stringArg("counter")),
					b.invoke(to.senders[j], to.dApps[j], "inc", stringArg("own")))
			}
		}
		return blocks
	}, func(parallel *invokeExecutionTestObjects) {
		stats := parallel.state.stor.invokeStats
		stats.db = failingInvokeStatsDB{KeyValue: stats.db}
		broken = parallel
	})
	require.NoError(t, err)
	require.Same(t, broken, to)
	assert.Equal(t, int64(blocksNum*invokeExecutionTestAccounts), to.integerEntry(t, to.dApps[0], "counter"))
	stats := to.state.stor.invokeStats
	assert.Empty(t, stats.entries, "nothing is cached for the statistics that failed to load")
	assert.Zero(t, stats.revision)
	c := to.counters()
	assert.Positive(t, c.reused)
	assert.Positive(t, c.mispredicted, "without statistics the invokes of the same dApp are speculated together")
}
