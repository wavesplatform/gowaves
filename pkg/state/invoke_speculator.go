package state

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// Speculative evaluation of invokes.
//
// A batch of InvokeScript transactions is evaluated in parallel over the current state, which must not be
// modified while the batch runs. Every evaluation reads the state through its own recordingSmartState, so its
// actual read set is known. Transactions are still applied in the block order, and the actual write set of
// every applied transaction is put to the write log. A speculative result may replace the evaluation of the
// script only if nothing it has read was written since the batch was launched.

// invokeSpeculationTask is an invoke to evaluate speculatively.
type invokeSpeculationTask struct {
	tx *proto.InvokeScriptWithProofs
	// txID is the ID of the transaction. It is generated before the batch is launched, because workers only
	// read transactions.
	txID crypto.Digest
	// info is shared by the tasks of a batch and is only read by workers.
	info *fallibleValidationParams
}

// speculativeResult is the outcome of a speculative evaluation of an invoke.
type speculativeResult struct {
	txID crypto.Digest
	// scriptAddress is the address of the dApp the evaluation was performed for.
	scriptAddress proto.WavesAddress
	evaluation    invokeEvaluation
	// reads is the actual read set of the evaluation, including the reads made to resolve the dApp address
	// and its script.
	reads accessSet
	// baseIndex is the number of transactions of the block applied when the batch was launched. The result
	// is based on the state left by the transactions with indexes less than baseIndex.
	baseIndex int
}

// speculativeEvaluator evaluates the invoke over the given state view and returns the evaluation along with
// the address of the dApp. An error means that there is no speculative result for the transaction.
type speculativeEvaluator func(
	state types.EnrichedSmartState, task invokeSpeculationTask,
) (invokeEvaluation, proto.WavesAddress, error)

// speculationOutcome tells what invokeSpeculator.take has found for a transaction.
type speculationOutcome byte

const (
	// speculationMissing means that the transaction has no speculative result.
	speculationMissing speculationOutcome = iota
	// speculationReused means that the speculative result is valid and is handed over to the caller.
	speculationReused
	// speculationMispredicted means that the speculative result was invalidated by an earlier transaction of
	// the block and is dropped.
	speculationMispredicted
)

// invokeSpeculator evaluates batches of invokes speculatively on a fixed pool of worker goroutines, keeps the
// results and validates them against the write log of the block.
//
// The pool is started lazily by the first batch that needs it and is stopped by close. Results and the write
// log are per block state and must be used from the appender goroutine only; runBatch and close are
// additionally serialized with each other, so the speculator can be closed from another goroutine.
type invokeSpeculator struct {
	workersNum int
	state      types.EnrichedSmartState
	evaluate   speculativeEvaluator

	mu      sync.Mutex // Guards the lifecycle of the pool and is held while a batch runs.
	jobs    chan func()
	workers sync.WaitGroup
	closed  bool

	results map[crypto.Digest]*speculativeResult
	// lastWriter holds the index of the last transaction of the block that has written the key.
	lastWriter map[accessKey]int
}

// newInvokeSpeculator creates the speculator that evaluates invokes over the given state with workersNum
// goroutines. The state must be safe for concurrent readers.
func newInvokeSpeculator(
	workersNum int, state types.EnrichedSmartState, stngs *settings.BlockchainSettings,
) *invokeSpeculator {
	evaluate := func(view types.EnrichedSmartState, task invokeSpeculationTask) (
		invokeEvaluation, proto.WavesAddress, error,
	) {
		return evaluateInvokeSpeculatively(view, stngs, task)
	}
	return newInvokeSpeculatorWithEvaluator(workersNum, state, evaluate)
}

func newInvokeSpeculatorWithEvaluator(
	workersNum int, state types.EnrichedSmartState, evaluate speculativeEvaluator,
) *invokeSpeculator {
	if workersNum <= 0 {
		panic("invoke speculator created with negative or zero goroutines number")
	}
	return &invokeSpeculator{
		workersNum: workersNum,
		state:      state,
		evaluate:   evaluate,
		results:    make(map[crypto.Digest]*speculativeResult),
		lastWriter: make(map[accessKey]int),
	}
}

// evaluateInvokeSpeculatively resolves the dApp address and its script through the given view, so the reads of
// the alias and of the account script get recorded, and runs the pure evaluation of the invoke.
func evaluateInvokeSpeculatively(
	view types.EnrichedSmartState, stngs *settings.BlockchainSettings, task invokeSpeculationTask,
) (invokeEvaluation, proto.WavesAddress, error) {
	scriptAddress, err := view.NewestRecipientToAddress(task.tx.ScriptRecipient)
	if err != nil {
		return invokeEvaluation{}, proto.WavesAddress{}, err
	}
	tree, err := view.NewestScriptByAccount(proto.NewRecipientFromAddress(scriptAddress))
	if err != nil {
		return invokeEvaluation{}, proto.WavesAddress{}, err
	}
	ev, err := evaluateInvokeScriptWithProofs(view, stngs, tree, task.tx, task.info, scriptAddress)
	if err != nil {
		return invokeEvaluation{}, proto.WavesAddress{}, err
	}
	return ev, scriptAddress, nil
}

// start launches the pool of workers if it is not running yet. It must be called with mu held.
func (s *invokeSpeculator) start() {
	if s.jobs != nil {
		return
	}
	jobs := make(chan func())
	s.jobs = jobs
	for range s.workersNum {
		s.workers.Go(func() {
			for job := range jobs {
				job()
			}
		})
	}
}

// close stops the pool of workers and waits for them to finish. A closed speculator produces no new results.
// It is safe to call close more than once.
func (s *invokeSpeculator) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if s.jobs != nil {
		close(s.jobs)
		s.workers.Wait()
	}
}

// speculate evaluates the task over its own recording view of the state. It returns nil if there is no
// speculative result for the transaction: the evaluation can not be set up or it panics.
// Such transaction is evaluated inline, where the failure is reproduced and reported in the regular way.
func (s *invokeSpeculator) speculate(task invokeSpeculationTask, baseIndex int) *speculativeResult {
	var res *speculativeResult // Stays nil if the evaluation panics.
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Warn("Speculative invoke evaluation panicked, transaction will be evaluated in order",
					slog.String("panic", fmt.Sprint(r)))
			}
		}()
		res = s.speculateUnsafe(task, baseIndex)
	}()
	return res
}

// speculateUnsafe does the job of speculate without containing panics.
func (s *invokeSpeculator) speculateUnsafe(task invokeSpeculationTask, baseIndex int) *speculativeResult {
	view := newRecordingSmartState(s.state)
	ev, scriptAddress, err := s.evaluate(view, task)
	if err != nil {
		return nil
	}
	return &speculativeResult{
		txID:          task.txID,
		scriptAddress: scriptAddress,
		evaluation:    ev,
		reads:         view.readSet(),
		baseIndex:     baseIndex,
	}
}

// runBatch evaluates the tasks in parallel and stores their speculative results, replacing the results the
// transactions already have. It blocks until all the tasks are finished and returns the number of results
// produced. baseIndex is the number of transactions of the block applied so far.
//
// The state must not be modified while the batch runs.
func (s *invokeSpeculator) runBatch(tasks []invokeSpeculationTask, baseIndex int) int {
	if len(tasks) == 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0
	}
	out := make([]*speculativeResult, len(tasks))
	if len(tasks) == 1 { // Nothing to parallelize, don't bother the pool.
		out[0] = s.speculate(tasks[0], baseIndex)
	} else {
		s.start()
		var batch sync.WaitGroup
		for i := range tasks {
			batch.Add(1)
			s.jobs <- func() {
				defer batch.Done()
				out[i] = s.speculate(tasks[i], baseIndex)
			}
		}
		batch.Wait()
	}
	produced := 0
	for _, res := range out {
		if res != nil {
			s.results[res.txID] = res
			produced++
		}
	}
	return produced
}

// observe puts the actual write set of the applied transaction with the given index in the block to the
// write log. Transactions must be observed in the order of application.
func (s *invokeSpeculator) observe(txIndex int, writes accessSet) {
	for k := range writes {
		s.lastWriter[k] = txIndex
	}
}

// dropResults drops all speculative results. It is used when the write set of an applied transaction is
// unknown, so nothing evaluated before it can be validated.
func (s *invokeSpeculator) dropResults() {
	clear(s.results)
}

// valid reports whether the speculative result is still valid: nothing it has read was written by the
// transactions applied since its batch was launched.
func (s *invokeSpeculator) valid(res *speculativeResult) bool {
	for k := range res.reads {
		if w, ok := s.lastWriter[k]; ok && w >= res.baseIndex {
			return false
		}
	}
	return true
}

// hasValid reports whether the transaction holds a speculative result that is valid at the moment.
func (s *invokeSpeculator) hasValid(txID crypto.Digest) bool {
	res, ok := s.results[txID]
	return ok && s.valid(res)
}

// has reports whether the transaction holds a speculative result, valid or invalidated.
func (s *invokeSpeculator) has(txID crypto.Digest) bool {
	_, ok := s.results[txID]
	return ok
}

// take removes the speculative result of the transaction and returns it if it is still valid. If the result
// was invalidated, it is dropped and the misprediction is reported.
func (s *invokeSpeculator) take(txID crypto.Digest) (*speculativeResult, speculationOutcome) {
	res, ok := s.results[txID]
	if !ok {
		return nil, speculationMissing
	}
	delete(s.results, txID)
	if !s.valid(res) {
		return nil, speculationMispredicted
	}
	return res, speculationReused
}

// resetBlock drops all speculative results and clears the write log. It must be called at the boundaries of
// a block: results never survive the block they were computed for.
func (s *invokeSpeculator) resetBlock() {
	clear(s.results)
	clear(s.lastWriter)
}
