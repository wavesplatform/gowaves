// Benchmarks of the Ride tree evaluator over the corpus of testData scripts.
//
// Run the benchmarks and save the results for comparison with benchstat:
//
//	go test -run '^$' -bench 'Benchmark(CallVerifier|CallFunction|NewEvaluationScope|NewEnvironmentWithWrappedState)' \
//		-benchmem -count 10 ./pkg/ride/ > new.txt
//	go run golang.org/x/perf/cmd/benchstat@latest old.txt new.txt
//
// Collect CPU and allocation profiles and open the flame graphs in a browser (View -> Flame Graph):
//
//	go test -run '^$' -bench BenchmarkCallFunction -benchmem -cpuprofile cpu.out -memprofile mem.out ./pkg/ride/
//	go tool pprof -http=:8080 cpu.out
//	go tool pprof -http=:8081 -sample_index=alloc_space mem.out

package ride //nolint:testpackage // White-box benchmarks of the unexported evaluator internals.

import (
	stderrs "errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	ridec "github.com/wavesplatform/gowaves/pkg/ride/compiler"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	evaluatorBenchScheme = proto.TestNetScheme
	evaluatorBenchHeight = 1_000_000
	evaluatorBenchLimit  = 52_000
)

// evaluatorBenchState is a minimal types.EnrichedSmartState for the evaluator benchmarks.
// It embeds the interface and overrides only the methods used by the corpus, so mock reflection
// doesn't show up in the profiles. Calls of other methods panic on the nil embedded interface.
type evaluatorBenchState struct {
	types.EnrichedSmartState

	integers map[string]int64
	strings  map[string]string
}

func newEvaluatorBenchState() *evaluatorBenchState {
	return &evaluatorBenchState{
		integers: map[string]int64{"counter": 12345},
		strings:  map[string]string{"name": "evaluator benchmark"},
	}
}

func (s *evaluatorBenchState) AddingBlockHeight() (uint64, error) { return evaluatorBenchHeight, nil }

func (s *evaluatorBenchState) IsNotFound(err error) bool { return errors.Is(err, proto.ErrNotFound) }

func (s *evaluatorBenchState) NewestRecipientToAddress(recipient proto.Recipient) (proto.WavesAddress, error) {
	if a := recipient.Address(); a != nil {
		return *a, nil
	}
	return proto.WavesAddress{}, proto.ErrNotFound
}

func (s *evaluatorBenchState) RetrieveNewestIntegerEntry(
	_ proto.Recipient, key string,
) (*proto.IntegerDataEntry, error) {
	if v, ok := s.integers[key]; ok {
		return &proto.IntegerDataEntry{Key: key, Value: v}, nil
	}
	return nil, proto.ErrNotFound
}

func (s *evaluatorBenchState) RetrieveNewestStringEntry(_ proto.Recipient, key string) (*proto.StringDataEntry, error) {
	if v, ok := s.strings[key]; ok {
		return &proto.StringDataEntry{Key: key, Value: v}, nil
	}
	return nil, proto.ErrNotFound
}

func (s *evaluatorBenchState) RetrieveNewestBooleanEntry(proto.Recipient, string) (*proto.BooleanDataEntry, error) {
	return nil, proto.ErrNotFound
}

func (s *evaluatorBenchState) RetrieveNewestBinaryEntry(proto.Recipient, string) (*proto.BinaryDataEntry, error) {
	return nil, proto.ErrNotFound
}

func (s *evaluatorBenchState) WavesBalanceProfile(proto.AddressID) (*types.WavesBalanceProfile, error) {
	return &types.WavesBalanceProfile{Balance: 100_000_000_000, Generating: 100_000_000_000}, nil
}

func (s *evaluatorBenchState) NewestAssetBalanceByAddressID(proto.AddressID, crypto.Digest) (uint64, error) {
	return 100_000_000_000, nil
}

// evaluatorBenchAccounts are the deterministic accounts of the evaluator benchmarks.
type evaluatorBenchAccounts struct {
	senderPK crypto.PublicKey
	senderSK crypto.SecretKey
	sender   proto.WavesAddress
	dApp     proto.WavesAddress
	asset    crypto.Digest
}

func newEvaluatorBenchAccounts(tb testing.TB) evaluatorBenchAccounts {
	tb.Helper()
	sk, pk, err := crypto.GenerateKeyPair([]byte("evaluator benchmark sender"))
	require.NoError(tb, err)
	sender, err := proto.NewAddressFromPublicKey(evaluatorBenchScheme, pk)
	require.NoError(tb, err)
	_, dAppPK, err := crypto.GenerateKeyPair([]byte("evaluator benchmark dApp"))
	require.NoError(tb, err)
	dApp, err := proto.NewAddressFromPublicKey(evaluatorBenchScheme, dAppPK)
	require.NoError(tb, err)
	return evaluatorBenchAccounts{
		senderPK: pk,
		senderSK: sk,
		sender:   sender,
		dApp:     dApp,
		asset:    crypto.MustDigestFromBase58("BrjUWjndUanm5VsJkbUip8VRYy6LWJePtxya3FNv4TQa"),
	}
}

func (a evaluatorBenchAccounts) transfer(tb testing.TB) *proto.TransferWithProofs {
	tb.Helper()
	tx := proto.NewUnsignedTransferWithProofs(3, a.senderPK, proto.NewOptionalAssetWaves(),
		proto.NewOptionalAssetWaves(), 1_700_000_000_000, 12_345_678, 500_000,
		proto.NewRecipientFromAddress(a.dApp), proto.Attachment("evaluator benchmark attachment"))
	require.NoError(tb, tx.Sign(evaluatorBenchScheme, a.senderSK))
	return tx
}

func (a evaluatorBenchAccounts) invoke(tb testing.TB, call proto.FunctionCall) *proto.InvokeScriptWithProofs {
	tb.Helper()
	tx := proto.NewUnsignedInvokeScriptWithProofs(2, a.senderPK, proto.NewRecipientFromAddress(a.dApp), call,
		a.payments(), proto.NewOptionalAssetWaves(), 900_000, 1_700_000_000_000)
	require.NoError(tb, tx.Sign(evaluatorBenchScheme, a.senderSK))
	return tx
}

func (a evaluatorBenchAccounts) payments() proto.ScriptPayments {
	return proto.ScriptPayments{
		{Amount: 1_000_000, Asset: proto.NewOptionalAssetWaves()},
		{Amount: 2_000_000, Asset: *proto.NewOptionalAssetFromDigest(a.asset)},
	}
}

// newEvaluatorBenchEnvironment creates a real *EvaluationEnvironment the same way the script caller does it.
// For the DApp callables of library version 5 and later the environment wraps the state with the payments
// attached, also like in the script caller.
func newEvaluatorBenchEnvironment(
	tb testing.TB, rideV6 bool, tree *ast.Tree, tx proto.Transaction,
) *EvaluationEnvironment {
	tb.Helper()
	a := newEvaluatorBenchAccounts(tb)
	env, err := NewEnvironment(evaluatorBenchScheme, newEvaluatorBenchState(), 0, 0, true, rideV6, true, true, true)
	require.NoError(tb, err)
	env.SetThisFromAddress(a.dApp)
	env.ChooseSizeCheck(tree.LibVersion)
	env.SetTimestamp(1_700_000_000_000)
	env.ChooseTakeString(true)
	env.ChooseMaxDataEntriesSize(true)
	env.SetLimit(evaluatorBenchLimit)
	require.NoError(tb, env.SetTransaction(tx))
	invoke, ok := tx.(*proto.InvokeScriptWithProofs)
	if !ok {
		return env
	}
	require.NoError(tb, env.SetInvoke(invoke, tree.LibVersion))
	if tree.LibVersion < ast.LibV5 {
		return env
	}
	env, err = NewEnvironmentWithWrappedState(env, newEvaluatorBenchState(), invoke.Payments, a.sender,
		false, tree.LibVersion, true)
	require.NoError(tb, err)
	return env
}

// resetEvaluatorBenchEnvironment sets a fresh complexity calculator, so every evaluation starts from zero.
func resetEvaluatorBenchEnvironment(env *EvaluationEnvironment, rideV6 bool) {
	env.setComplexityCalculator(newComplexityCalculatorByRideV6Activation(rideV6))
	env.SetLimit(evaluatorBenchLimit)
}

// evaluatorCorpusEntry is a single evaluation of the evaluator corpus: a verifier call when call is nil,
// otherwise a DApp callable call with fixed arguments.
type evaluatorCorpusEntry struct {
	name string
	tree *ast.Tree
	call *proto.FunctionCall
}

func (c evaluatorCorpusEntry) isVerifier() bool { return c.call == nil }

func (c evaluatorCorpusEntry) transaction(tb testing.TB) proto.Transaction {
	tb.Helper()
	a := newEvaluatorBenchAccounts(tb)
	if c.isVerifier() {
		return a.transfer(tb)
	}
	return a.invoke(tb, *c.call)
}

// evaluatorRideV6Variants are the RideV6 activation variants, they select the complexity calculator.
var evaluatorRideV6Variants = []struct { //nolint:gochecknoglobals // Read-only list of the benchmark variants.
	name   string
	rideV6 bool
}{
	{name: "CalculatorV1", rideV6: false},
	{name: "CalculatorV2", rideV6: true},
}

func compileEvaluatorScript(tb testing.TB, name string) *ast.Tree {
	tb.Helper()
	src, err := os.ReadFile(filepath.Join("testData", "evaluator", name+".ride"))
	require.NoError(tb, err)
	tree, errs := ridec.CompileToTree(string(src))
	require.NoError(tb, stderrs.Join(errs...), name)
	return tree
}

// loadEvaluatorCorpus builds the evaluator corpus from the Ride sources in testData/evaluator.
// The order of the entries is stable, because the golden test relies on it.
func loadEvaluatorCorpus(tb testing.TB) []evaluatorCorpusEntry {
	tb.Helper()
	call := func(name string, args ...proto.Argument) *proto.FunctionCall {
		fc := proto.NewFunctionCall(name, args)
		return &fc
	}
	list := make(proto.Arguments, 100)
	for i := range list {
		list[i] = proto.NewIntegerArgument(int64(i * 3))
	}
	dAppV3 := compileEvaluatorScript(tb, "dapp_v3")
	dAppV6 := compileEvaluatorScript(tb, "dapp_v6")
	return []evaluatorCorpusEntry{
		{name: "expression_v3", tree: compileEvaluatorScript(tb, "expression_v3")},
		{name: "expression_v6", tree: compileEvaluatorScript(tb, "expression_v6")},
		{name: "dapp_v3_verifier", tree: dAppV3},
		{name: "dapp_v3_store", tree: dAppV3,
			call: call("store", proto.NewIntegerArgument(42), proto.NewStringArgument("legacy"))},
		{name: "dapp_v6_verifier", tree: dAppV6},
		{name: "dapp_v6_nested", tree: dAppV6, call: call("nested", proto.NewIntegerArgument(3))},
		{name: "dapp_v6_matches", tree: dAppV6,
			call: call("matches", proto.NewIntegerArgument(7), proto.NewStringArgument("beta"))},
		{name: "dapp_v6_lets", tree: dAppV6, call: call("lets", proto.NewIntegerArgument(5))},
		{name: "dapp_v6_fold", tree: dAppV6, call: call("fold", &proto.ListArgument{Items: list})},
		{name: "dapp_v6_actions", tree: dAppV6, call: call("actions")},
	}
}

// evaluateCorpusEntry runs a single evaluation of the corpus entry in the given environment.
func evaluateCorpusEntry(env *EvaluationEnvironment, c evaluatorCorpusEntry) (Result, error) {
	if c.isVerifier() {
		return CallVerifier(env, c.tree)
	}
	return CallFunction(env, c.tree, *c.call)
}

// evaluatorBenchmarkResult keeps the benchmark results reachable, so the compiler can't drop the evaluation.
var evaluatorBenchmarkResult Result //nolint:gochecknoglobals // Benchmark sink.

func TestEvaluatorCorpus(t *testing.T) {
	for _, c := range loadEvaluatorCorpus(t) {
		for _, v := range evaluatorRideV6Variants {
			t.Run(c.name+"/"+v.name, func(t *testing.T) {
				env := newEvaluatorBenchEnvironment(t, v.rideV6, c.tree, c.transaction(t))
				res, err := evaluateCorpusEntry(env, c)
				require.NoError(t, err)
				require.NotNil(t, res)
				assert := require.New(t)
				assert.Positive(res.Complexity())
				if c.isVerifier() {
					assert.True(res.Result())
				} else {
					assert.NotEmpty(res.ScriptActions())
				}
				// Repeated evaluation in the reset environment must give the same result.
				resetEvaluatorBenchEnvironment(env, v.rideV6)
				again, err := evaluateCorpusEntry(env, c)
				require.NoError(t, err)
				assert.Equal(res, again)
			})
		}
	}
}

func benchmarkEvaluatorCorpus(b *testing.B, verifiers bool) {
	for _, c := range loadEvaluatorCorpus(b) {
		if c.isVerifier() != verifiers {
			continue
		}
		for _, v := range evaluatorRideV6Variants {
			b.Run(c.name+"/"+v.name, func(b *testing.B) {
				env := newEvaluatorBenchEnvironment(b, v.rideV6, c.tree, c.transaction(b))
				b.ReportAllocs()
				for b.Loop() {
					resetEvaluatorBenchEnvironment(env, v.rideV6)
					evaluatorBenchmarkResult, _ = evaluateCorpusEntry(env, c)
				}
			})
		}
	}
}

func BenchmarkCallVerifier(b *testing.B) { benchmarkEvaluatorCorpus(b, true) }

func BenchmarkCallFunction(b *testing.B) { benchmarkEvaluatorCorpus(b, false) }

// evaluationScopeBenchmarkResult keeps the scope reachable, so the compiler can't drop its construction.
var evaluationScopeBenchmarkResult evaluationScope //nolint:gochecknoglobals // Benchmark sink.

func BenchmarkNewEvaluationScope(b *testing.B) {
	a := newEvaluatorBenchAccounts(b)
	for v := ast.LibV1; v <= ast.CurrentMaxLibraryVersion(); v++ {
		for _, rv := range evaluatorRideV6Variants {
			b.Run(fmt.Sprintf("V%d/%s", v, rv.name), func(b *testing.B) {
				tree := &ast.Tree{LibVersion: v}
				env := newEvaluatorBenchEnvironment(b, rv.rideV6, tree, a.transfer(b))
				b.ReportAllocs()
				for b.Loop() {
					s, err := newEvaluationScope(v, env, true, nil, 0)
					if err != nil {
						b.Fatal(err)
					}
					evaluationScopeBenchmarkResult = s
				}
			})
		}
	}
}

// environmentBenchmarkResult keeps the environment reachable, so the compiler can't drop its construction.
var environmentBenchmarkResult *EvaluationEnvironment //nolint:gochecknoglobals // Benchmark sink.

func BenchmarkNewEnvironmentWithWrappedState(b *testing.B) {
	a := newEvaluatorBenchAccounts(b)
	tree := &ast.Tree{LibVersion: ast.LibV6}
	for _, rv := range evaluatorRideV6Variants {
		b.Run(rv.name, func(b *testing.B) {
			env := newEvaluatorBenchEnvironment(b, rv.rideV6, tree, a.transfer(b))
			st := newEvaluatorBenchState()
			payments := a.payments()
			b.ReportAllocs()
			for b.Loop() {
				e, err := NewEnvironmentWithWrappedState(env, st, payments, a.sender, false, tree.LibVersion, true)
				if err != nil {
					b.Fatal(err)
				}
				environmentBenchmarkResult = e
			}
		})
	}
}
