package state

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// speculatorStub describes how the stubbed evaluation behaves for a transaction.
type speculatorStub struct {
	reads  []accessKey
	err    error
	panics bool
}

// speculatorTestObjects holds a speculator with stubbed evaluation. The spent complexity of a stubbed
// evaluation is the fee of the transaction, which is also what makes transactions different.
type speculatorTestObjects struct {
	sp    *invokeSpeculator
	stubs map[*proto.InvokeScriptWithProofs]speculatorStub
	calls atomic.Int64
}

func newSpeculatorTestObjects(t *testing.T, workersNum int) *speculatorTestObjects {
	to := &speculatorTestObjects{stubs: make(map[*proto.InvokeScriptWithProofs]speculatorStub)}
	evaluate := func(state types.EnrichedSmartState, task invokeSpeculationTask) (
		invokeEvaluation, proto.WavesAddress, error,
	) {
		to.calls.Add(1)
		view, ok := state.(*recordingSmartState)
		if !ok {
			return invokeEvaluation{}, proto.WavesAddress{}, errors.Errorf("unexpected state view %T", state)
		}
		stub := to.stubs[task.tx] // The map is only read while a batch runs.
		for _, k := range stub.reads {
			view.reads.add(k)
		}
		if stub.panics {
			panic("stubbed evaluation panic")
		}
		if stub.err != nil {
			return invokeEvaluation{}, proto.WavesAddress{}, stub.err
		}
		ev := invokeEvaluation{functionCall: task.tx.FunctionCall, spentComplexity: int(task.tx.Fee)}
		return ev, testGlobal.recipientInfo.addr, nil
	}
	to.sp = newInvokeSpeculatorWithEvaluator(workersNum, nil, evaluate)
	t.Cleanup(to.sp.close)
	return to
}

// addTx creates a unique transaction with the given behaviour of its evaluation.
func (to *speculatorTestObjects) addTx(t *testing.T, stub speculatorStub) invokeSpeculationTask {
	fee := invokeFee + uint64(len(to.stubs))
	tx := createInvokeScriptWithProofs(t, nil, proto.NewFunctionCall("call", proto.Arguments{}), feeAsset, fee)
	require.NotNil(t, tx.ID)
	to.stubs[tx] = stub
	return speculationTestTask(tx, nil)
}

// speculationTestTask creates the task of the transaction, which ID must be generated already.
func speculationTestTask(tx *proto.InvokeScriptWithProofs, info *fallibleValidationParams) invokeSpeculationTask {
	return invokeSpeculationTask{tx: tx, txID: *tx.ID, info: info}
}

func speculatorTestKeys(t *testing.T) (accessKey, accessKey, accessKey) {
	_, a := accessTestAccount(t, "speculator-account-a")
	_, b := accessTestAccount(t, "speculator-account-b")
	return accountDataAccessKey(a.ID()), accountBalanceAccessKey(a.ID()), accountDataAccessKey(b.ID())
}

func TestInvokeSpeculatorRunBatch(t *testing.T) {
	keyA, keyB, keyC := speculatorTestKeys(t)
	for _, test := range []struct {
		name       string
		workersNum int
		tasksNum   int
	}{
		{name: "empty batch", workersNum: 2, tasksNum: 0},
		{name: "single task", workersNum: 2, tasksNum: 1},
		{name: "fewer tasks than workers", workersNum: 8, tasksNum: 3},
		{name: "more tasks than workers", workersNum: 3, tasksNum: 50},
		{name: "single worker", workersNum: 1, tasksNum: 10},
	} {
		t.Run(test.name, func(t *testing.T) {
			const baseIndex = 7
			to := newSpeculatorTestObjects(t, test.workersNum)
			tasks := make([]invokeSpeculationTask, test.tasksNum)
			for i := range tasks {
				tasks[i] = to.addTx(t, speculatorStub{reads: []accessKey{keyA, keyB, keyC}[:i%3+1]})
			}
			assert.Equal(t, test.tasksNum, to.sp.runBatch(tasks, baseIndex))
			assert.Equal(t, int64(test.tasksNum), to.calls.Load())
			assert.Len(t, to.sp.results, test.tasksNum)
			for i, task := range tasks {
				txID := *task.tx.ID
				assert.True(t, to.sp.hasValid(txID))
				res, outcome := to.sp.take(txID)
				require.Equal(t, speculationReused, outcome)
				require.NotNil(t, res)
				assert.Equal(t, txID, res.txID)
				assert.Equal(t, baseIndex, res.baseIndex)
				assert.Equal(t, testGlobal.recipientInfo.addr, res.scriptAddress)
				assert.Equal(t, int(invokeFee)+i, res.evaluation.spentComplexity)
				assert.Equal(t, newAccessSet([]accessKey{keyA, keyB, keyC}[:i%3+1]...), res.reads)
				// The result is handed over only once.
				assert.False(t, to.sp.hasValid(txID))
				res, outcome = to.sp.take(txID)
				assert.Nil(t, res)
				assert.Equal(t, speculationMissing, outcome)
			}
			assert.Empty(t, to.sp.results)
		})
	}
}

func TestInvokeSpeculatorValidation(t *testing.T) {
	keyA, keyB, keyC := speculatorTestKeys(t)
	const baseIndex = 5
	for _, test := range []struct {
		name    string
		reads   []accessKey
		observe func(sp *invokeSpeculator)
		valid   bool
	}{
		{name: "no writes", reads: []accessKey{keyA, keyB}, observe: func(*invokeSpeculator) {}, valid: true},
		{
			name: "no reads", reads: nil, valid: true,
			observe: func(sp *invokeSpeculator) { sp.observe(baseIndex, newAccessSet(keyA, keyB, keyC)) },
		},
		{
			name: "write of other key", reads: []accessKey{keyA, keyB}, valid: true,
			observe: func(sp *invokeSpeculator) { sp.observe(baseIndex, newAccessSet(keyC)) },
		},
		{
			name: "write of the same key before the batch", reads: []accessKey{keyA}, valid: true,
			observe: func(sp *invokeSpeculator) { sp.observe(baseIndex-1, newAccessSet(keyA)) },
		},
		{
			name: "write of the same key by the first transaction after the batch", reads: []accessKey{keyA, keyB},
			observe: func(sp *invokeSpeculator) { sp.observe(baseIndex, newAccessSet(keyB)) },
		},
		{
			name: "write of the same key by a later transaction", reads: []accessKey{keyA},
			observe: func(sp *invokeSpeculator) {
				sp.observe(baseIndex, newAccessSet(keyC))
				sp.observe(baseIndex+1, newAccessSet(keyB))
				sp.observe(baseIndex+2, newAccessSet(keyA, keyC))
			},
		},
		{
			name: "same account, other kind of access", reads: []accessKey{keyA}, valid: true,
			observe: func(sp *invokeSpeculator) { sp.observe(baseIndex, newAccessSet(keyB)) },
		},
		{
			name: "last writer moves past the base", reads: []accessKey{keyA},
			observe: func(sp *invokeSpeculator) {
				sp.observe(baseIndex-2, newAccessSet(keyA))
				sp.observe(baseIndex+3, newAccessSet(keyA))
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			to := newSpeculatorTestObjects(t, 2)
			task := to.addTx(t, speculatorStub{reads: test.reads})
			other := to.addTx(t, speculatorStub{})
			require.Equal(t, 2, to.sp.runBatch([]invokeSpeculationTask{task, other}, baseIndex))
			test.observe(to.sp)

			txID := *task.tx.ID
			assert.Equal(t, test.valid, to.sp.valid(to.sp.results[txID]))
			assert.Equal(t, test.valid, to.sp.hasValid(txID))
			res, outcome := to.sp.take(txID)
			if test.valid {
				require.Equal(t, speculationReused, outcome)
				require.NotNil(t, res)
				assert.Equal(t, txID, res.txID)
			} else {
				assert.Equal(t, speculationMispredicted, outcome)
				assert.Nil(t, res)
			}
			// The result is removed in both cases, a misprediction is reported once.
			_, outcome = to.sp.take(txID)
			assert.Equal(t, speculationMissing, outcome)
		})
	}
}

func TestInvokeSpeculatorRespeculation(t *testing.T) {
	keyA, _, _ := speculatorTestKeys(t)
	to := newSpeculatorTestObjects(t, 2)
	task := to.addTx(t, speculatorStub{reads: []accessKey{keyA}})
	other := to.addTx(t, speculatorStub{})
	batch := []invokeSpeculationTask{task, other}
	txID := *task.tx.ID

	require.Equal(t, 2, to.sp.runBatch(batch, 0))
	to.sp.observe(0, newAccessSet(keyA))
	require.False(t, to.sp.hasValid(txID))
	require.True(t, to.sp.hasValid(*other.tx.ID))

	// The new batch replaces the invalid result with the one based on the current state.
	require.Equal(t, 1, to.sp.runBatch(batch[:1], 1))
	require.True(t, to.sp.hasValid(txID))
	res, outcome := to.sp.take(txID)
	require.Equal(t, speculationReused, outcome)
	assert.Equal(t, 1, res.baseIndex)
	// The untouched result of the first batch is still there.
	res, outcome = to.sp.take(*other.tx.ID)
	require.Equal(t, speculationReused, outcome)
	assert.Equal(t, 0, res.baseIndex)
}

func TestInvokeSpeculatorContainsWorkerFailures(t *testing.T) {
	keyA, _, _ := speculatorTestKeys(t)
	for _, workersNum := range []int{1, 4} {
		to := newSpeculatorTestObjects(t, workersNum)
		var tasks []invokeSpeculationTask
		for i := range 12 {
			var stub speculatorStub
			switch i % 3 {
			case 1:
				stub = speculatorStub{reads: []accessKey{keyA}, panics: true}
			case 2:
				stub = speculatorStub{err: errors.New("failed to set up evaluation")}
			}
			tasks = append(tasks, to.addTx(t, stub))
		}
		assert.Equal(t, 4, to.sp.runBatch(tasks, 0))
		for i, task := range tasks {
			res, outcome := to.sp.take(*task.tx.ID)
			if i%3 == 0 {
				assert.Equal(t, speculationReused, outcome)
				assert.NotNil(t, res)
			} else {
				// A failed speculation is not a misprediction, the transaction is just evaluated inline.
				assert.Equal(t, speculationMissing, outcome)
				assert.Nil(t, res)
			}
		}
		// Workers survive panics.
		assert.Equal(t, 4, to.sp.runBatch(tasks, 0))

		// A panic of a single task evaluated on the caller's goroutine is contained as well.
		assert.Equal(t, 0, to.sp.runBatch(tasks[1:2], 0))
	}
}

func TestInvokeSpeculatorResetBlock(t *testing.T) {
	keyA, keyB, _ := speculatorTestKeys(t)
	to := newSpeculatorTestObjects(t, 2)
	first := to.addTx(t, speculatorStub{reads: []accessKey{keyA}})
	second := to.addTx(t, speculatorStub{reads: []accessKey{keyB}})
	batch := []invokeSpeculationTask{first, second}
	require.Equal(t, 2, to.sp.runBatch(batch, 0))
	to.sp.observe(0, newAccessSet(keyA))

	to.sp.resetBlock()
	assert.Empty(t, to.sp.results)
	assert.Empty(t, to.sp.lastWriter)
	for _, task := range batch {
		assert.False(t, to.sp.hasValid(*task.tx.ID))
		res, outcome := to.sp.take(*task.tx.ID)
		assert.Nil(t, res)
		assert.Equal(t, speculationMissing, outcome)
	}

	// The write log of the previous block does not affect the next one.
	require.Equal(t, 2, to.sp.runBatch(batch, 0))
	assert.True(t, to.sp.hasValid(*first.tx.ID))
	assert.True(t, to.sp.hasValid(*second.tx.ID))
}

func TestInvokeSpeculatorDropResults(t *testing.T) {
	keyA, _, _ := speculatorTestKeys(t)
	to := newSpeculatorTestObjects(t, 2)
	first := to.addTx(t, speculatorStub{reads: []accessKey{keyA}})
	second := to.addTx(t, speculatorStub{})
	require.Equal(t, 2, to.sp.runBatch([]invokeSpeculationTask{first, second}, 0))
	require.True(t, to.sp.hasValid(first.txID))
	require.True(t, to.sp.hasValid(second.txID))

	to.sp.dropResults()
	for _, task := range []invokeSpeculationTask{first, second} {
		assert.False(t, to.sp.hasValid(task.txID))
		res, outcome := to.sp.take(task.txID)
		assert.Nil(t, res)
		assert.Equal(t, speculationMissing, outcome)
	}
}

func TestInvokeSpeculatorLifecycle(t *testing.T) {
	t.Run("pool is started lazily", func(t *testing.T) {
		to := newSpeculatorTestObjects(t, 3)
		assert.Nil(t, to.sp.jobs)
		assert.Zero(t, to.sp.runBatch(nil, 0))
		assert.Nil(t, to.sp.jobs)
		assert.Equal(t, 1, to.sp.runBatch([]invokeSpeculationTask{to.addTx(t, speculatorStub{})}, 0))
		assert.Nil(t, to.sp.jobs, "single task must not start the pool")
		batch := []invokeSpeculationTask{to.addTx(t, speculatorStub{}), to.addTx(t, speculatorStub{})}
		assert.Equal(t, 2, to.sp.runBatch(batch, 0))
		assert.NotNil(t, to.sp.jobs)
	})
	t.Run("close without start", func(t *testing.T) {
		to := newSpeculatorTestObjects(t, 3)
		to.sp.close()
		to.sp.close()
	})
	t.Run("closed speculator produces nothing", func(t *testing.T) {
		to := newSpeculatorTestObjects(t, 3)
		batch := []invokeSpeculationTask{to.addTx(t, speculatorStub{}), to.addTx(t, speculatorStub{})}
		require.Equal(t, 2, to.sp.runBatch(batch, 0))
		to.sp.close()
		to.sp.close()
		to.sp.resetBlock()
		calls := to.calls.Load()
		assert.Zero(t, to.sp.runBatch(batch, 0))
		assert.Zero(t, to.sp.runBatch(batch[:1], 0))
		assert.Equal(t, calls, to.calls.Load())
		assert.Empty(t, to.sp.results)
	})
	t.Run("close waits for the running batch", func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		evaluate := func(types.EnrichedSmartState, invokeSpeculationTask) (
			invokeEvaluation, proto.WavesAddress, error,
		) {
			started <- struct{}{}
			<-release
			return invokeEvaluation{}, proto.WavesAddress{}, nil
		}
		sp := newInvokeSpeculatorWithEvaluator(2, nil, evaluate)
		fc := proto.NewFunctionCall("call", proto.Arguments{})
		batch := []invokeSpeculationTask{
			speculationTestTask(createInvokeScriptWithProofs(t, nil, fc, feeAsset, invokeFee), nil),
			speculationTestTask(createInvokeScriptWithProofs(t, nil, fc, feeAsset, invokeFee+1), nil),
		}
		var wg sync.WaitGroup
		var produced int
		wg.Go(func() { produced = sp.runBatch(batch, 0) })
		<-started
		<-started
		// The batch holds the lock of the lifecycle until all its tasks are finished, so close has to wait.
		require.False(t, sp.mu.TryLock(), "running batch must hold the lifecycle lock")
		closed := make(chan struct{})
		wg.Go(func() {
			sp.close()
			close(closed)
		})
		select {
		case <-closed:
			t.Fatal("speculator closed while the batch is running")
		case <-time.After(50 * time.Millisecond):
		}
		close(release)
		wg.Wait()
		<-closed
		assert.Equal(t, 2, produced, "results of the running batch must not be lost")
		assert.True(t, sp.closed)
	})
	t.Run("invalid workers number", func(t *testing.T) {
		assert.Panics(t, func() { newInvokeSpeculatorWithEvaluator(0, nil, nil) })
	})
}

// TestInvokeSpeculatorOverRealState runs batches of real script evaluations over a test state repeatedly.
// Results must be the same as the results of inline evaluations. The test is meaningful under -race.
func TestInvokeSpeculatorOverRealState(t *testing.T) {
	const (
		workersNum = 4
		txsNum     = 16
		rounds     = 5
	)
	for _, test := range []struct {
		name            string
		dApp            string
		rideV5Activated bool
		tx              func(t *testing.T, i int) *proto.InvokeScriptWithProofs
	}{
		{
			name: "plain state before RideV5",
			dApp: "dapp.base64",
			tx: func(t *testing.T, i int) *proto.InvokeScriptWithProofs {
				if i%4 == 3 { // Fails by script error: nothing is deposited yet.
					fc := proto.NewFunctionCall("withdraw", proto.Arguments{&proto.IntegerArgument{Value: int64(i)}})
					return createInvokeScriptWithProofs(t, nil, fc, feeAsset, invokeFee)
				}
				payments := proto.ScriptPayments{{Amount: uint64(i + 1)}}
				fc := proto.NewFunctionCall("deposit", proto.Arguments{})
				return createInvokeScriptWithProofs(t, payments, fc, feeAsset, invokeFee)
			},
		},
		{
			name:            "wrapped state after RideV5",
			dApp:            "ride5_recursive_invoke.base64",
			rideV5Activated: true,
			tx: func(t *testing.T, i int) *proto.InvokeScriptWithProofs {
				size := 10 + i
				if i%4 == 3 { // Fails by script error: too much data is written.
					size = 150 + i // Sizes differ to make transactions unique.
				}
				fc := proto.NewFunctionCall("keyvalue", proto.Arguments{
					&proto.IntegerArgument{Value: 99},
					&proto.StringArgument{Value: strings.Repeat("0", size)},
				})
				return createInvokeScriptWithProofs(t, nil, fc, feeAsset, invokeFee)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			to := createInvokeApplierTestObjects(t)
			info := to.fallibleValidationParams(t)
			info.acceptFailed = test.rideV5Activated
			info.blockV5Activated = test.rideV5Activated
			info.rideV5Activated = test.rideV5Activated
			to.setDApp(t, test.dApp, testGlobal.recipientInfo)
			to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, 1000*invokeFee)

			dApp := testGlobal.recipientInfo.addr
			stngs := to.state.appender.ia.sc.settings
			tree, err := to.state.stor.scriptsStorage.newestScriptByAddr(dApp)
			require.NoError(t, err)

			tasks := make([]invokeSpeculationTask, txsNum)
			expected := make([]invokeEvaluation, txsNum)
			failed := 0
			for i := range tasks {
				tx := test.tx(t, i)
				tasks[i] = speculationTestTask(tx, info)
				expected[i], err = evaluateInvokeScriptWithProofs(to.state, stngs, tree, tx, info, dApp)
				require.NoError(t, err)
				if expected[i].err != nil {
					failed++
				}
			}
			require.Equal(t, txsNum/4, failed, "every fourth transaction must fail by script error")

			sp := newInvokeSpeculator(workersNum, to.state, stngs)
			defer sp.close()
			for round := range rounds {
				sp.resetBlock()
				require.Equal(t, txsNum, sp.runBatch(tasks, round))
				for i, task := range tasks {
					res, outcome := sp.take(*task.tx.ID)
					require.Equal(t, speculationReused, outcome)
					assert.Equal(t, *task.tx.ID, res.txID)
					assert.Equal(t, dApp, res.scriptAddress)
					assert.Equal(t, round, res.baseIndex)
					assertEqualInvokeEvaluations(t, expected[i], res.evaluation)
					assert.True(t, res.reads.has(accountScriptAccessKey(dApp.ID())))
				}
			}
		})
	}
}

// TestInvokeSpeculatorResolvesDApp checks that the default evaluation records the reads made to find the
// script and that a failure to find it results in no speculative result.
func TestInvokeSpeculatorResolvesDApp(t *testing.T) {
	to := createInvokeApplierTestObjects(t)
	info := to.fallibleValidationParams(t)
	to.setDApp(t, "dapp.base64", testGlobal.recipientInfo)
	to.setAndCheckInitialWavesBalance(t, testGlobal.senderInfo.addr, 1000*invokeFee)
	sp := newInvokeSpeculator(2, to.state, to.state.appender.ia.sc.settings)
	defer sp.close()

	newTx := func(recipient proto.Recipient, fee uint64) *proto.InvokeScriptWithProofs {
		tx := proto.NewUnsignedInvokeScriptWithProofs(1, testGlobal.senderInfo.pk, recipient,
			proto.NewFunctionCall("deposit", proto.Arguments{}), proto.ScriptPayments{{Amount: 1}}, feeAsset, fee,
			defaultTimestamp)
		require.NoError(t, tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk))
		return tx
	}
	alias := proto.NewAlias(proto.MainNetScheme, "unknown-alias")
	scripted := newTx(proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), invokeFee)
	notScripted := newTx(proto.NewRecipientFromAddress(testGlobal.minerInfo.addr), invokeFee+1)
	byUnknownAlias := newTx(proto.NewRecipientFromAlias(*alias), invokeFee+2)
	tasks := []invokeSpeculationTask{
		speculationTestTask(scripted, info), speculationTestTask(notScripted, info),
		speculationTestTask(byUnknownAlias, info),
	}
	assert.Equal(t, 1, sp.runBatch(tasks, 0))

	res, outcome := sp.take(*scripted.ID)
	require.Equal(t, speculationReused, outcome)
	require.NoError(t, res.evaluation.err)
	dAppID := testGlobal.recipientInfo.addr.ID()
	assert.True(t, res.reads.has(accountScriptAccessKey(dAppID)))
	assert.True(t, res.reads.has(accountDataAccessKey(dAppID)), "deposit reads the data of the dApp")
	for _, tx := range []*proto.InvokeScriptWithProofs{notScripted, byUnknownAlias} {
		res, outcome = sp.take(*tx.ID)
		assert.Nil(t, res)
		assert.Equal(t, speculationMissing, outcome)
	}
}
