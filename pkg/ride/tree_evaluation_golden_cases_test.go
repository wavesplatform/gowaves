package ride //nolint:testpackage // Uses the unexported evaluator corpus and complexity calculators.

import (
	stdsha256 "crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrs "errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	ridec "github.com/wavesplatform/gowaves/pkg/ride/compiler"
)

// evaluatorGoldenResult is the observable outcome of a single evaluation that must stay the same.
type evaluatorGoldenResult struct {
	result     string // Type of the result, empty if evaluation failed.
	ok         bool   // Boolean result of the verifier.
	actions    int    // Number of the script actions.
	digest     string // SHA-256 of the serialized script actions, empty if there are no actions.
	param      string // String representation of the result param, empty if there is no param.
	complexity int    // Complexity of the successful result.
	errType    EvaluationError
	err        string // Error message, empty if no error is expected.
	callStack  string // Call stack of the evaluation error, joined with " | ".
	spent      int    // Spent complexity reported by the evaluation error.
}

type evaluatorGoldenEntry struct {
	name string
	res  evaluatorGoldenResult
}

// evaluatorGoldenCase is a single evaluation of the golden test: a corpus entry under one of the complexity
// calculators, optionally with the calculator prepared to hit the limit or the integer overflow.
type evaluatorGoldenCase struct {
	name    string
	entry   evaluatorCorpusEntry
	rideV6  bool
	prepare func(env *EvaluationEnvironment, rideV6 bool)
}

func evaluatorGoldenLimit(limit uint32) func(*EvaluationEnvironment, bool) {
	return func(env *EvaluationEnvironment, _ bool) { env.SetLimit(limit) }
}

// evaluatorGoldenOverflow presets the calculator complexity close to the integer limit, so the next additions
// overflow. The calculator limit is set to the maximal integer, so the limit check never fails first.
func evaluatorGoldenOverflow(headroom int) func(*EvaluationEnvironment, bool) {
	return func(env *EvaluationEnvironment, rideV6 bool) {
		if rideV6 {
			env.setComplexityCalculator(&complexityCalculatorV2{c: math.MaxInt - headroom, l: math.MaxInt})
			return
		}
		env.setComplexityCalculator(&complexityCalculatorV1{c: math.MaxInt - headroom, l: math.MaxInt})
	}
}

func compileEvaluatorSource(tb testing.TB, src string) *ast.Tree {
	tb.Helper()
	tree, errs := ridec.CompileToTree(src)
	require.NoError(tb, stderrs.Join(errs...))
	return tree
}

const evaluatorGoldenEdgeDApp = `{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func one() = 1
func pass(x: Int) = x

@Callable(i)
func empty() = []

@Callable(i)
func user() = {
  let a = one()
  let b = pass(a)
  [IntegerEntry("b", b)]
}

@Callable(i)
func constructor() = {
  let a = Address(base58'3P3336rNSSU8')
  [BinaryEntry("a", a.bytes)]
}

@Callable(i)
func failure() = throw("user failure")

@Callable(i)
func overflow(x: Int) = [IntegerEntry("x", x * x)]
`

// evaluatorGoldenEdgeTrees are the hand-built trees of the cases the compiler doesn't allow.
func evaluatorGoldenEdgeTrees() []evaluatorCorpusEntry {
	unknown := ast.NewTree(ast.ContentTypeExpression, ast.LibV6)
	unknown.Verifier = ast.NewReferenceNode("unknown")

	unknownV3 := ast.NewTree(ast.ContentTypeExpression, ast.LibV3)
	unknownV3.Verifier = ast.NewReferenceNode("unknown")

	// func f(a: Int) = a > 0; f()
	mismatch := ast.NewTree(ast.ContentTypeExpression, ast.LibV6)
	mismatch.Verifier = ast.NewFunctionDeclarationNode("f", []string{"a"},
		ast.NewFunctionCallNode(ast.NativeFunction("102"), []ast.Node{ast.NewReferenceNode("a"), ast.NewLongNode(0)}),
		ast.NewFunctionCallNode(ast.UserFunction("f"), nil),
	)

	// func f(a: Int) = a > 0; f(1, 2)
	mismatchMore := ast.NewTree(ast.ContentTypeExpression, ast.LibV3)
	mismatchMore.Verifier = ast.NewFunctionDeclarationNode("f", []string{"a"},
		ast.NewFunctionCallNode(ast.NativeFunction("102"), []ast.Node{ast.NewReferenceNode("a"), ast.NewLongNode(0)}),
		ast.NewFunctionCallNode(ast.UserFunction("f"), []ast.Node{ast.NewLongNode(1), ast.NewLongNode(2)}),
	)

	// Unknown function called through the user function node falls back to the native functions.
	unknownFunction := ast.NewTree(ast.ContentTypeExpression, ast.LibV6)
	unknownFunction.Verifier = ast.NewFunctionCallNode(ast.UserFunction("missing"), nil)

	// Non-boolean condition.
	condition := ast.NewTree(ast.ContentTypeExpression, ast.LibV6)
	condition.Verifier = ast.NewConditionalNode(ast.NewLongNode(1), ast.NewBooleanNode(true),
		ast.NewBooleanNode(false))

	// Property of the transaction that doesn't exist.
	property := ast.NewTree(ast.ContentTypeExpression, ast.LibV6)
	property.Verifier = ast.NewPropertyNode("missing", ast.NewReferenceNode("tx"))

	// Callable that calls the zero cost function HalfDown: zero complexity of a native function is an error
	// since RideV6, but the compiler doesn't produce such calls.
	zeroCost := ast.NewTree(ast.ContentTypeApplication, ast.LibV6)
	halfDown := ast.NewFunctionCallNode(ast.UserFunction("HalfDown"), nil)
	zeroCostCallable := ast.NewFunctionDeclarationNode("zero", nil,
		ast.NewConditionalNode(ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{halfDown, halfDown}),
			ast.NewReferenceNode("nil"), ast.NewReferenceNode("nil")),
		nil,
	)
	zeroCostCallable.InvocationParameter = "i"
	zeroCost.Functions = []ast.Node{zeroCostCallable}
	zeroCostCall := proto.NewFunctionCall("zero", nil)

	return []evaluatorCorpusEntry{
		{name: "edge_unknown_reference_v6", tree: unknown},
		{name: "edge_unknown_reference_v3", tree: unknownV3},
		{name: "edge_user_function_fewer_args", tree: mismatch},
		{name: "edge_user_function_more_args", tree: mismatchMore},
		{name: "edge_unknown_function", tree: unknownFunction},
		{name: "edge_non_boolean_condition", tree: condition},
		{name: "edge_unknown_property", tree: property},
		{name: "edge_zero_cost_function", tree: zeroCost, call: &zeroCostCall},
	}
}

func evaluatorGoldenEdgeEntries(tb testing.TB) []evaluatorCorpusEntry {
	tb.Helper()
	call := func(name string, args ...proto.Argument) *proto.FunctionCall {
		fc := proto.NewFunctionCall(name, args)
		return &fc
	}
	edge := compileEvaluatorSource(tb, evaluatorGoldenEdgeDApp)
	dAppV6 := compileEvaluatorScript(tb, "dapp_v6")
	return append(evaluatorGoldenEdgeTrees(),
		evaluatorCorpusEntry{name: "edge_empty", tree: edge, call: call("empty")},
		evaluatorCorpusEntry{name: "edge_user", tree: edge, call: call("user")},
		evaluatorCorpusEntry{name: "edge_zero_cost_constructor", tree: edge, call: call("constructor")},
		evaluatorCorpusEntry{name: "edge_failure", tree: edge, call: call("failure")},
		evaluatorCorpusEntry{name: "edge_multiplication_overflow", tree: edge,
			call: call("overflow", proto.NewIntegerArgument(math.MaxInt64))},
		evaluatorCorpusEntry{name: "edge_unknown_callable", tree: edge, call: call("missing")},
		evaluatorCorpusEntry{name: "edge_callable_args_count", tree: dAppV6, call: call("nested")},
		evaluatorCorpusEntry{name: "edge_dapp_no_verifier", tree: edge},
	)
}

// loadEvaluatorGoldenCases builds the golden cases. The order of the cases is stable, the golden data relies on it.
func loadEvaluatorGoldenCases(tb testing.TB) []evaluatorGoldenCase {
	tb.Helper()
	type modifier struct {
		name    string
		prepare func(*EvaluationEnvironment, bool)
	}
	corpusModifiers := []modifier{
		{name: "default"},
		{name: "limit_100", prepare: evaluatorGoldenLimit(100)},
		{name: "limit_1000", prepare: evaluatorGoldenLimit(1000)},
		{name: "overflow_100", prepare: evaluatorGoldenOverflow(100)},
	}
	edgeModifiers := []modifier{
		{name: "default"},
		{name: "limit_0", prepare: evaluatorGoldenLimit(0)},
		{name: "limit_1", prepare: evaluatorGoldenLimit(1)},
		{name: "overflow_0", prepare: evaluatorGoldenOverflow(0)},
		{name: "overflow_1", prepare: evaluatorGoldenOverflow(1)},
	}
	var cases []evaluatorGoldenCase
	add := func(entries []evaluatorCorpusEntry, modifiers []modifier) {
		for _, e := range entries {
			for _, v := range evaluatorRideV6Variants {
				for _, m := range modifiers {
					cases = append(cases, evaluatorGoldenCase{
						name:    e.name + "/" + v.name + "/" + m.name,
						entry:   e,
						rideV6:  v.rideV6,
						prepare: m.prepare,
					})
				}
			}
		}
	}
	add(loadEvaluatorCorpus(tb), corpusModifiers)
	add(evaluatorGoldenEdgeEntries(tb), edgeModifiers)
	return cases
}

func serializeEvaluatorActions(tb testing.TB, actions []proto.ScriptAction) string {
	tb.Helper()
	var sb strings.Builder
	for _, a := range actions {
		js, err := json.Marshal(a)
		require.NoError(tb, err)
		_, _ = fmt.Fprintf(&sb, "%T:%s\n", a, js)
	}
	return sb.String()
}

func runEvaluatorGoldenCase(tb testing.TB, c evaluatorGoldenCase) evaluatorGoldenResult {
	tb.Helper()
	env := newEvaluatorBenchEnvironment(tb, c.rideV6, c.entry.tree, c.entry.transaction(tb))
	if c.prepare != nil {
		c.prepare(env, c.rideV6)
	}
	res, err := evaluateCorpusEntry(env, c.entry)
	if err != nil {
		require.Nil(tb, res)
		return evaluatorGoldenResult{
			errType:   GetEvaluationErrorType(err),
			err:       err.Error(),
			callStack: strings.Join(EvaluationErrorCallStack(err), " | "),
			spent:     EvaluationErrorSpentComplexity(err),
		}
	}
	r := evaluatorGoldenResult{
		result:     fmt.Sprintf("%T", res),
		ok:         res.Result(),
		actions:    len(res.ScriptActions()),
		complexity: res.Complexity(),
	}
	if r.actions == 0 {
		require.Nil(tb, res.ScriptActions(), "empty actions must stay nil")
	} else {
		d := stdsha256.Sum256([]byte(serializeEvaluatorActions(tb, res.ScriptActions())))
		r.digest = hex.EncodeToString(d[:])
	}
	if p := res.userResult(); p != nil {
		r.param = p.String()
	}
	return r
}

func TestEvaluatorGolden(t *testing.T) {
	cases := loadEvaluatorGoldenCases(t)
	require.Len(t, evaluatorGolden, len(cases), "golden data doesn't match the golden cases")
	for i, c := range cases {
		expected := evaluatorGolden[i]
		require.Equal(t, expected.name, c.name, "golden data order doesn't match the golden cases")
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, expected.res, runEvaluatorGoldenCase(t, c))
		})
	}
}

// TestEvaluatorGoldenParallel runs the golden cases concurrently to check, under the race detector, that
// evaluations sharing the cached scope tables don't interfere.
func TestEvaluatorGoldenParallel(t *testing.T) {
	t.Parallel()
	cases := loadEvaluatorGoldenCases(t)
	require.Len(t, evaluatorGolden, len(cases), "golden data doesn't match the golden cases")
	for i, c := range cases {
		expected := evaluatorGolden[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expected.res, runEvaluatorGoldenCase(t, c))
		})
	}
}
