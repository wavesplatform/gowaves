package ride //nolint:testpackage // Uses the unexported evaluator corpus and complexity calculators.

import (
	stdsha256 "crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

const evaluatorWalkHeader = `{-# STDLIB_VERSION %d #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}
`

// evaluatorWalkFrames calls user functions with lets in their bodies many times at the same depth,
// so the call frames are pushed and popped repeatedly.
const evaluatorWalkFrames = `
func g(x: Int) = { let y = x + 1; let w = y * 2; w - x }
func h(x: Int) = { let z = g(x); let q = g(z); q + g(x) + z }
let a = h(1)
let b = h(a)
let c = [a, b, h(2), g(g(g(3)))]
size(c) == 4 && a + b > 0 && c[3] == g(g(g(3)))
`

// evaluatorWalkNested has nested system function calls, so the arguments stack holds the arguments of
// several calls at once, conditionals, references and properties.
const evaluatorWalkNested = `
let s = (1 + 2) + (3 + (4 + (5 + 6)))
let fee = match tx {
  case t: TransferTransaction => t.fee
  case _ => 0
}
let l = [s, s + 1, if (s > 20) then s * 2 else s - 2, fee]
let t = (l[0], l[1], tx.sender.bytes.size())
if (s + t._3 > 0 && isDefined(getInteger(tx.sender, "k"))) then false
else valueOrElse(getInteger(tx.sender, "k"), 0) == 0 && t._3 == 26 && l[2] == 42 && size(l) == 4 && t._2 == s + 1
`

// evaluatorWalkTrees are the hand-built trees with the user function shadowing and the system functions
// called through the user function nodes.
func evaluatorWalkTrees() []evaluatorCorpusEntry {
	long := func(v int64) ast.Node { return ast.NewLongNode(v) }
	call := func(f ast.Function, args ...ast.Node) ast.Node { return ast.NewFunctionCallNode(f, args) }
	eq := func(a, b ast.Node) ast.Node { return call(ast.NativeFunction("0"), a, b) }
	and := func(a, b ast.Node) ast.Node { return ast.NewConditionalNode(a, b, ast.NewBooleanNode(false)) }

	// func f() = 1; func f() = 2; f() == 2
	shadow := ast.NewTree(ast.ContentTypeExpression, ast.LibV6)
	shadow.Verifier = ast.NewFunctionDeclarationNode("f", nil, long(1),
		ast.NewFunctionDeclarationNode("f", nil, long(2), eq(call(ast.UserFunction("f")), long(2))))

	// let r = { func f(x) = x + 10; f(1) }; func f(x) = x; r == 11 && f(1) == 1
	scoped := ast.NewTree(ast.ContentTypeExpression, ast.LibV3)
	scoped.Verifier = ast.NewAssignmentNode("r",
		ast.NewFunctionDeclarationNode("f", []string{"x"},
			call(ast.NativeFunction("100"), ast.NewReferenceNode("x"), long(10)),
			call(ast.UserFunction("f"), long(1))),
		ast.NewFunctionDeclarationNode("f", []string{"x"}, ast.NewReferenceNode("x"),
			and(eq(ast.NewReferenceNode("r"), long(11)), eq(call(ast.UserFunction("f"), long(1)), long(1)))))

	// The user function isDefined shadows the system function, outside its scope the system one is called.
	// let a = { func isDefined(x) = x == 5; isDefined(5) }; a && isDefined(5) && valueOrElse(unit, 3) == 3
	native := ast.NewTree(ast.ContentTypeExpression, ast.LibV6)
	native.Verifier = ast.NewAssignmentNode("a",
		ast.NewFunctionDeclarationNode("isDefined", []string{"x"}, eq(ast.NewReferenceNode("x"), long(5)),
			call(ast.UserFunction("isDefined"), long(5))),
		and(ast.NewReferenceNode("a"),
			and(call(ast.UserFunction("isDefined"), long(5)),
				eq(call(ast.UserFunction("valueOrElse"), ast.NewReferenceNode("unit"), long(3)), long(3)))))

	return []evaluatorCorpusEntry{
		{name: "walk_shadow", tree: shadow},
		{name: "walk_scoped", tree: scoped},
		{name: "walk_native", tree: native},
	}
}

func evaluatorWalkEntries(tb testing.TB) []evaluatorCorpusEntry {
	tb.Helper()
	src := func(v int, body string) *ast.Tree {
		return compileEvaluatorSource(tb, fmt.Sprintf(evaluatorWalkHeader, v)+body)
	}
	return append(evaluatorWalkTrees(),
		evaluatorCorpusEntry{name: "walk_frames_v3", tree: src(3, evaluatorWalkFrames)},
		evaluatorCorpusEntry{name: "walk_frames_v6", tree: src(6, evaluatorWalkFrames)},
		evaluatorCorpusEntry{name: "walk_nested_v6", tree: src(6, evaluatorWalkNested)},
	)
}

// evaluatorWalkGolden holds the results of the evaluator on the walk cases: the result without the limit
// and the SHA-256 of the results under every limit from zero to the complexity of the successful evaluation.
// The values were captured from the unoptimized evaluator and MUST NOT be changed.
//
//nolint:lll // Golden data, one line per case.
var evaluatorWalkGolden = map[string]struct { //nolint:gochecknoglobals // Read-only golden data.
	res    evaluatorGoldenResult
	digest string
}{
	"walk_shadow/CalculatorV1":    {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 1}, digest: "e6bf4539271ac3f98a24e0a0906325f0c7a27c1ed51b5bce859948441f31d581"},
	"walk_shadow/CalculatorV2":    {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 2}, digest: "506b6566c54a1b1f207db315f6712d21836a3625d7045234d22dcd4e9263c017"},
	"walk_scoped/CalculatorV1":    {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 7}, digest: "6999a03a9c2f98d34c6ecfd5bfb247ae130aacc80ab1c771df849beec1c37ecc"},
	"walk_scoped/CalculatorV2":    {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 4}, digest: "845e43757d408d37854927e474b8bfaa58ebb4575c8d98ad117a4bfb4c36a140"},
	"walk_native/CalculatorV1":    {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 10}, digest: "a7dc8e620a573c08734b3438e0b42d5ca3c40b0079004c73394df315e460ec7f"},
	"walk_native/CalculatorV2":    {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 5}, digest: "2467e7fc5619c5dfc96b4b8f16dc92fbe31b6e8b161c39ce5916794031b13a83"},
	"walk_frames_v3/CalculatorV1": {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 152}, digest: "f48a85b4678c7ef134e88a31beb5d658f9278313641b3777683a785e8ac1b1b3"},
	"walk_frames_v3/CalculatorV2": {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 67}, digest: "6c9f4942aa7e5e824007546a93390c2057d695fdaa54f1ed53092fd16f7fc599"},
	"walk_frames_v6/CalculatorV1": {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 148}, digest: "83325d5b62747f31c3991d8244a16a5b27e7bfe321ca61aad3630da59862896a"},
	"walk_frames_v6/CalculatorV2": {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 63}, digest: "9ecc9027f4d7dc39151c547381ec50c4c1916488ea2e039ffb6852b82e12b332"},
	"walk_nested_v6/CalculatorV1": {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 92}, digest: "532536eccffea92e91242d6a0505ea2b1fc38ae89a9ef3dd77c9995419be3d94"},
	"walk_nested_v6/CalculatorV2": {res: evaluatorGoldenResult{result: "ride.ScriptResult", ok: true, complexity: 54}, digest: "d0a3a18bf5175ff92d7e0112db3caf98d4b5f7c2160bf1ffaf89899690d1c3f2"},
}

// TestEvaluatorWalk checks the evaluation of the user function shadowing, the system functions called through
// the user function nodes, and the complexity errors raised in the middle of every node, by evaluating the cases
// under every complexity limit up to the complexity of the successful evaluation.
func TestEvaluatorWalk(t *testing.T) {
	for _, e := range evaluatorWalkEntries(t) {
		for _, v := range evaluatorRideV6Variants {
			name := e.name + "/" + v.name
			t.Run(name, func(t *testing.T) {
				res := runEvaluatorGoldenCase(t, evaluatorGoldenCase{entry: e, rideV6: v.rideV6})
				require.NotZero(t, res.complexity, "%+v", res)
				var sb strings.Builder
				for l := 0; l <= res.complexity; l++ {
					r := runEvaluatorGoldenCase(t, evaluatorGoldenCase{
						entry: e, rideV6: v.rideV6, prepare: evaluatorGoldenLimit(uint32(l)),
					})
					_, _ = fmt.Fprintf(&sb, "%d:%+v\n", l, r)
				}
				d := stdsha256.Sum256([]byte(sb.String()))
				digest := hex.EncodeToString(d[:])
				expected, ok := evaluatorWalkGolden[name]
				require.True(t, ok, "no golden data")
				require.Equal(t, expected.res, res)
				require.Equal(t, expected.digest, digest, sb.String())
			})
		}
	}
}
