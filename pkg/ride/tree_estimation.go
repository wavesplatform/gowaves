package ride

import (
	"encoding/base64"
	"fmt"
	"maps"
	"slices"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type TreeEstimation struct {
	Estimation int            `cbor:"0,keyasint"`
	Verifier   int            `cbor:"1,keyasint,omitempty"`
	Functions  map[string]int `cbor:"2,keyasint,omitempty"`
}

func EstimateTree(tree *ast.Tree, v int) (TreeEstimation, error) {
	switch v {
	case 1:
		te, err := newTreeEstimatorV1(tree)
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		e, verifier, functions, err := te.estimate()
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		return TreeEstimation{Estimation: e, Verifier: verifier, Functions: functions}, nil
	case 2:
		if te, ok := treeEstimationV2Override(tree); ok {
			return te, nil
		}
		te, err := newTreeEstimatorV2(tree)
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		e, verifier, functions, err := te.estimate()
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		return TreeEstimation{Estimation: e, Verifier: verifier, Functions: functions}, nil
	case 3:
		te, err := newTreeEstimatorV3(tree)
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		max, verifier, functions, err := te.estimate()
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		return TreeEstimation{Estimation: max, Verifier: verifier, Functions: functions}, nil
	case 4:
		te, err := newTreeEstimatorV4(tree)
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		e, verifier, functions, err := te.estimate()
		if err != nil {
			return TreeEstimation{}, errors.Wrapf(err, "failed to estimate with tree estimator V%d", v)
		}
		return TreeEstimation{Estimation: e, Verifier: verifier, Functions: functions}, nil
	default:
		return TreeEstimation{}, errors.Errorf("unsupported version of tree estimator '%d'", v)
	}
}

// treeEstimationV2Override returns the hard-coded V2 estimations of the scripts on which the V2 estimator
// overflows the stack. The values are consensus-critical and must not be changed.
func treeEstimationV2Override(tree *ast.Tree) (TreeEstimation, bool) {
	// The raw digests are compared, so no base64 encoding is needed on every call.
	for i := range treeEstimationV2Overrides {
		if o := &treeEstimationV2Overrides[i]; o.digest == tree.Digest {
			return TreeEstimation{Estimation: o.estimation, Functions: maps.Clone(o.functions)}, true
		}
	}
	return TreeEstimation{}, false
}

type treeEstimationV2OverrideEntry struct {
	digest     [32]byte
	estimation int
	functions  map[string]int // Cloned on return, callers get their own copy.
}

//nolint:gochecknoglobals,mnd,goconst // Consensus-critical constants, decoded once.
var treeEstimationV2Overrides = []treeEstimationV2OverrideEntry{
	{mustDecodeTreeDigest("WX51srRAh4WN7dzlOu5EDXYm1vbR9x+gzvd8r2NeXKI="), 3418, map[string]int{"random": 3418}},
	{mustDecodeTreeDigest("iCUS2gce1wBLSNEE//ehvTvBaoZ6aLtdLxyj1k76yjk="), 3223, map[string]int{"random": 3223}},
	{mustDecodeTreeDigest("x932L0eSYHVgTMIYKVVDKXm67RFFUrshgsKYYjFVLfg="), 3089, map[string]int{"random": 3089}},
	{mustDecodeTreeDigest("VPRjsCLrvwS8pKPk78vzZ2VplwlKWBbRs4M/KhZGpCU="), 3128,
		map[string]int{"setOrder": 3128, "cancelOrder": 1143, "executeOrder": 2794}},
	{mustDecodeTreeDigest("fjng1PLxQ1nXYlubqOuS8FLxzIaY92AheqsNfWuEmgE="), 3128,
		map[string]int{"setOrder": 3128, "cancelOrder": 1143, "executeOrder": 2791}},
	{mustDecodeTreeDigest("hBqgd3xz9N3dwfVBMQ5uba3l9Bo/+grSpZsCCyuefJs="), 3118,
		map[string]int{"setOrder": 3118, "cancelOrder": 1133, "executeOrder": 1887}},
	{mustDecodeTreeDigest("jLb7Zt9mH8cjIVhnyVYVXkATgHrNzqdEMF/PGG/tpV8="), 3128,
		map[string]int{"setOrder": 3128, "cancelOrder": 1143, "executeOrder": 2794}},
	{mustDecodeTreeDigest("oUYWZD6AgjbrJIlq9X89KkkbGn6yWRlao4+JcPVyMvM="), 3118,
		map[string]int{"setOrder": 3118, "cancelOrder": 1133, "executeOrder": 1875}},
}

func mustDecodeTreeDigest(s string) [32]byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) != 32 {
		panic(fmt.Sprintf("invalid tree digest %q", s))
	}
	return [32]byte(b)
}

// wrappedCallable holds the nodes created by wrapCallable, so they take one allocation. The arguments slice
// is allocated separately: keeping room for the arguments inline makes the common callables without or with
// few arguments allocate more bytes and run slower.
type wrappedCallable struct {
	trueNode   ast.BooleanNode
	call       ast.FunctionCallNode
	invocation ast.AssignmentNode
}

// wrapCallable turns a callable or the verifier into an expression that the estimators can walk:
// the global declarations followed by the invocation parameter and a call of the function with constant
// arguments. The estimators only read the constant, so one node is shared by all arguments.
// Like before, the function's block and the blocks of the global declarations are set in the tree.
//
// ATTENTION: this is the only place that modifies nodes of an existing tree (SetBlock), so the estimation of a
// tree must not run concurrently with another estimation of the same tree. The evaluation never modifies a tree
// and does not read the Block fields set here (global declarations are declared by name and expression, callables
// and the verifier are entered by Body), so trees are safe to evaluate from several goroutines
// (see TestConcurrentTreeEvaluation).
func wrapCallable(declarations []ast.Node, node *ast.FunctionDeclarationNode) ast.Node {
	w := &wrappedCallable{trueNode: ast.BooleanNode{Value: true}}
	var args []ast.Node
	if n := len(node.Arguments); n > 0 {
		args = make([]ast.Node, n)
	}
	for i := range args {
		args[i] = &w.trueNode
	}
	// It's definitely user function as node is sent from FunctionDeclarationNode
	w.call = ast.FunctionCallNode{Function: ast.UserFunction(node.Name), Arguments: args}
	node.SetBlock(&w.call)
	w.invocation = ast.AssignmentNode{Name: node.InvocationParameter, Expression: &w.trueNode, Block: node}
	var block ast.Node = &w.invocation
	for i := range slices.Backward(declarations) {
		declarations[i].SetBlock(block)
		block = declarations[i]
	}
	return block
}
