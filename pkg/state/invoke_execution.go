package state

import (
	"log/slog"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// Parallel execution of invokes during the application of a block.
//
// invokeExecutor ties together the pieces of the optimistic execution: statistics (invoke_stats.go), access
// prediction and batch selection (invoke_scheduler.go) and the speculative evaluation with its validation
// (invoke_speculator.go). The block appender drives it:
//
//	for every transaction i of the block:
//	    beforeTx(i)    - if i is an invoke without a valid speculative result, a batch is selected and evaluated
//	    appendTx(i)    - the script caller asks evaluation() for the result of the script instead of evaluating it
//	    afterTx(i)     - the write set of the applied transaction goes to the write log, statistics are updated
//
// Only the block application path uses it. The executor exists only if the feature is enabled, so with the
// feature disabled nothing here is reachable.

// invokeExecutionCounters are the counters of invokes of a block.
type invokeExecutionCounters struct {
	// speculated is the number of speculative results produced by batches.
	speculated int
	// reused is the number of invokes which script evaluation was replaced by a speculative result.
	reused int
	// mispredicted is the number of speculative results invalidated by earlier transactions of the block.
	mispredicted int
	// inline is the number of invokes evaluated in order by the appender goroutine.
	inline int
}

func (c invokeExecutionCounters) isZero() bool { return c == invokeExecutionCounters{} }

// invokeExecution is the record of the invoke being applied by the block appender.
type invokeExecution struct {
	tx   *proto.InvokeScriptWithProofs
	txID crypto.Digest
	// evaluated is set when the script caller has asked for the evaluation of the script. It stays false if
	// the transaction was rejected before the script is called.
	evaluated bool
	// scriptAddress is the address of the dApp the script caller has evaluated the script for.
	scriptAddress proto.WavesAddress
	// reads is the actual read set of the evaluation.
	reads accessSet
	// mispredicted is set if the speculative result of the transaction was invalidated.
	mispredicted bool
}

// cachedPrediction is the predicted access of the transaction of the block being applied.
type cachedPrediction struct {
	tx     proto.Transaction // Nil if there is no prediction.
	access predictedAccess
}

type invokeExecutor struct {
	scheme     proto.Scheme
	settings   *settings.BlockchainSettings
	state      types.EnrichedSmartState
	speculator *invokeSpeculator
	stats      *invokeStatsStorage
	predictor  accessPredictor
	// quiescent reports whether the state holds no changes of a partially applied transaction, so it can be
	// read by a batch.
	quiescent func() bool

	// predictions caches the predicted accesses of the transactions of the block by their indexes. Without it
	// the whole lookahead window is predicted again for every invoke that has no valid speculative result.
	// The cache is valid for predictionsRevision of statistics.
	predictions         []cachedPrediction
	predictionsRevision uint64

	counters invokeExecutionCounters
	// total accumulates the counters of all blocks, it is used by tests.
	total invokeExecutionCounters
	// current is the invoke being applied, nil if the transaction being applied is not a speculatable invoke.
	current *invokeExecution
	// miner is the address of the generator of the block being applied, hasMiner is false if it is unknown.
	miner    proto.AddressID
	hasMiner bool
}

func newInvokeExecutor(
	workersNum int,
	state types.EnrichedSmartState,
	stor *blockchainEntitiesStorage,
	stngs *settings.BlockchainSettings,
	quiescent func() bool,
) *invokeExecutor {
	e := &invokeExecutor{
		scheme:     stngs.AddressSchemeCharacter,
		settings:   stngs,
		state:      state,
		speculator: newInvokeSpeculator(workersNum, state, stngs),
		stats:      stor.invokeStats,
		quiescent:  quiescent,
	}
	e.predictor = accessPredictor{
		scheme: stngs.AddressSchemeCharacter,
		stats: func(key invokeStatsKey) *invokeStats {
			stats, ok, err := e.stats.stats(key)
			if err != nil || !ok { // Statistics are advisory: a failure to read them means no statistics.
				return nil
			}
			return stats
		},
		resolveAlias: func(alias proto.Alias) (proto.AddressID, bool) {
			addr, err := stor.aliases.newestAddrByAlias(alias.Alias)
			if err != nil {
				return proto.AddressID{}, false
			}
			return addr.ID(), true
		},
	}
	return e
}

func (e *invokeExecutor) close() { e.speculator.close() }

// resetBlock drops everything that belongs to the block being applied. It must be called at the boundaries
// of a block.
func (e *invokeExecutor) resetBlock() {
	e.speculator.resetBlock()
	e.predictions = nil
	e.current = nil
	e.hasMiner = false
}

// startBlock sets the generator of the block which transactions are going to be applied.
func (e *invokeExecutor) startBlock(minerPK crypto.PublicKey) {
	e.miner, e.hasMiner = e.predictor.senderID(minerPK)
}

// finishBlock logs and resets the counters of the block.
func (e *invokeExecutor) finishBlock(height proto.Height) {
	if e.counters.isZero() {
		return
	}
	c := e.counters
	slog.Debug("Parallel invoke execution", slog.Uint64("height", height),
		slog.Int("speculated", c.speculated), slog.Int("reused", c.reused),
		slog.Int("mispredicted", c.mispredicted), slog.Int("inline", c.inline))
	e.total.speculated += c.speculated
	e.total.reused += c.reused
	e.total.mispredicted += c.mispredicted
	e.total.inline += c.inline
	e.counters = invokeExecutionCounters{}
}

// beforeTx prepares the application of transaction i of the block. If it is an invoke that has no valid
// speculative result, a batch of invokes starting from it is selected and evaluated in parallel.
// It must be called between transactions: the state is read by the batch.
func (e *invokeExecutor) beforeTx(txs []proto.Transaction, i int, params *appendTxParams) {
	e.current = nil
	tx, ok := txs[i].(*proto.InvokeScriptWithProofs)
	if !ok {
		return
	}
	txID, ok := e.invokeID(tx)
	if !ok {
		return
	}
	e.current = &invokeExecution{tx: tx, txID: txID}
	if !e.quiescent() {
		// Must not happen: the state holds changes of a partially applied transaction. A batch must not read
		// such state, and the evaluation in order sees these changes while the speculative results do not.
		// So the results are dropped and the invoke is evaluated in order, exactly as without the executor.
		slog.Warn("Speculative invoke evaluation skipped: state is not between transactions",
			slog.String("tx", txID.String()))
		e.speculator.dropResults()
		return
	}
	if e.speculator.hasValid(txID) {
		return
	}
	if _, outcome := e.speculator.take(txID); outcome == speculationMispredicted {
		// The invalidated result is dropped here, because the batch may replace it with a new one.
		e.counters.mispredicted++
		e.current.mispredicted = true
	}
	tasks := e.selectTasks(txs, i, params)
	e.counters.speculated += e.speculator.runBatch(tasks, i)
}

// invokeID returns the ID of the invoke, generating it if necessary.
func (e *invokeExecutor) invokeID(tx *proto.InvokeScriptWithProofs) (crypto.Digest, bool) {
	id, err := tx.GetID(e.scheme)
	if err != nil {
		return crypto.Digest{}, false
	}
	txID, err := crypto.NewDigestFromBytes(id)
	if err != nil {
		return crypto.Digest{}, false
	}
	return txID, true
}

// predict returns the predicted access of transaction i of the block. Predictions are cached until statistics
// change. Cached access sets are shared, they must not be modified.
func (e *invokeExecutor) predict(txs []proto.Transaction, i int) predictedAccess {
	if revision := e.stats.revision; revision != e.predictionsRevision || len(e.predictions) != len(txs) {
		e.predictions = make([]cachedPrediction, len(txs))
		e.predictionsRevision = revision
	}
	if c := e.predictions[i]; c.tx == txs[i] {
		return c.access
	}
	access := e.predictor.predictTxAccess(txs[i])
	e.predictions[i] = cachedPrediction{tx: txs[i], access: access}
	return access
}

// selectTasks predicts the accesses of the upcoming transactions of the block, starting from transaction i,
// and returns the invokes to evaluate speculatively. Transaction i is the invoke to be applied next.
//
// An invoke which speculative result was invalidated is not selected again until it becomes the next one:
// it depends on something the prediction does not see, the balance of the generator of the block for example,
// so it would be evaluated in vain by every batch launched before it is applied.
//
// Nothing is returned if only the next invoke is selected: there is nothing to run in parallel, so it is
// evaluated in order as usual.
func (e *invokeExecutor) selectTasks(
	txs []proto.Transaction, i int, params *appendTxParams,
) []invokeSpeculationTask {
	upcoming := txs[i:]
	n := min(len(upcoming), invokeSpeculationLookahead)
	window := make([]predictedAccess, 0, n)
	skip := make([]bool, 0, n)
	ids := make([]crypto.Digest, 0, n)
	for j, tx := range upcoming[:n] {
		access := e.predict(txs, i+j)
		skipped := false
		var txID crypto.Digest
		if access.speculatable {
			var ok bool
			txID, ok = e.invokeID(tx.(*proto.InvokeScriptWithProofs)) //nolint:errcheck // Only invokes are speculatable.
			if ok {
				// The next invoke holds no result at all: it is either reused or dropped by beforeTx.
				skipped = e.speculator.has(txID)
			} else {
				access = barrierAccess()
			}
		}
		window = append(window, access)
		skip = append(skip, skipped)
		ids = append(ids, txID)
		if access.barrier { // Nothing is selected after the first barrier.
			break
		}
	}
	batch := selectBatch(window, skip)
	if len(batch) == 0 || (len(batch) == 1 && batch[0] == 0) {
		return nil
	}
	// The parameters are the same for all transactions of the block as far as the evaluation is concerned.
	info := &fallibleValidationParams{appendTxParams: params}
	tasks := make([]invokeSpeculationTask, 0, len(batch))
	for _, j := range batch {
		tx := upcoming[j].(*proto.InvokeScriptWithProofs) //nolint:errcheck // Only invokes are selected.
		tasks = append(tasks, invokeSpeculationTask{tx: tx, txID: ids[j], info: info})
	}
	return tasks
}

// evaluation returns the evaluation of the script of the invoke being applied: the speculative one if the
// transaction holds a valid speculative result, otherwise the script is evaluated in place over a recording
// view of the state, so the actual read set is known either way.
// The second result is false if the transaction is not the one being applied by the block appender, the caller
// has to evaluate the script itself then.
func (e *invokeExecutor) evaluation(
	tree *ast.Tree, tx *proto.InvokeScriptWithProofs, info *fallibleValidationParams,
	scriptAddress proto.WavesAddress,
) (invokeEvaluation, bool, error) {
	cur := e.current
	if cur == nil || cur.tx != tx || cur.evaluated {
		return invokeEvaluation{}, false, nil
	}
	cur.evaluated = true
	cur.scriptAddress = scriptAddress
	res, outcome := e.speculator.take(cur.txID)
	if outcome == speculationReused && res.scriptAddress != scriptAddress {
		// Must not happen: a valid result has read the alias and the script of the same dApp.
		outcome = speculationMispredicted
	}
	switch outcome {
	case speculationReused:
		e.counters.reused++
		cur.reads = res.reads
		return res.evaluation, true, nil
	case speculationMispredicted:
		e.counters.mispredicted++
		cur.mispredicted = true
	case speculationMissing:
	}
	e.counters.inline++
	view := newRecordingSmartState(e.state)
	ev, err := evaluateInvokeScriptWithProofs(view, e.settings, tree, tx, info, scriptAddress)
	cur.reads = view.readSet()
	return ev, true, err
}

// afterTx registers the application of transaction i of the block with the given snapshot: the actual write
// set of the transaction is put to the write log and, if the transaction is the invoke that was evaluated,
// the statistics of the called function are updated. applied is false if the transaction was elided.
func (e *invokeExecutor) afterTx(i int, txID []byte, snapshot txSnapshot, applied bool) {
	cur := e.current
	e.current = nil
	writes, err := writeSetFromSnapshot(txID, snapshot, e.scheme)
	if err != nil {
		// The write set is unknown, so nothing evaluated before this transaction can be trusted.
		slog.Warn("Failed to get write set of transaction", slog.Int("index", i), logging.Error(err))
		e.speculator.dropResults()
	} else {
		// The transaction itself becomes visible to scripts whatever its snapshot is.
		txKey := txAccessKey(txID)
		writes.add(txKey)
		e.speculator.observe(i, writes)
		delete(writes, txKey) // The key is unique, it is useless for statistics.
		for k := range writes {
			if k.kind == accessKindAlias { // Predictions depend on the resolution of aliases.
				e.predictions = nil
				break
			}
		}
	}
	if cur == nil {
		return
	}
	if !cur.evaluated {
		// The transaction was rejected before the script is called, its speculative result is not needed.
		e.speculator.take(cur.txID)
		return
	}
	if !applied || err != nil {
		return
	}
	sender, ok := e.predictor.senderID(cur.tx.SenderPK)
	if !ok {
		return
	}
	dApp := cur.scriptAddress.ID()
	if e.hasMiner && e.miner != sender && e.miner != dApp {
		// The generator of the block receives a part of the fee of every transaction. If this write gets to
		// statistics, all invokes are predicted to depend on each other, while scripts hardly ever read
		// the balance of the generator. Those that do are still protected by the validation of results.
		delete(writes, accountBalanceAccessKey(e.miner))
	}
	key := invokeStatsKey{dApp: dApp, function: cur.tx.FunctionCall.Name()}
	if uErr := e.stats.update(key, sender, cur.reads, writes, cur.mispredicted); uErr != nil {
		slog.Warn("Failed to update invoke statistics", logging.Error(uErr))
	}
}
