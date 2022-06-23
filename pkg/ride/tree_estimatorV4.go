package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type treeEstimatorV4 struct {
	tree  *ast.Tree
	scope *estimationScopeV3
}

func newTreeEstimatorV4(tree *ast.Tree) (*treeEstimatorV4, error) {
	r := &treeEstimatorV4{tree: tree}
	switch tree.LibVersion {
	case ast.LibV1, ast.LibV2:
		r.scope = newEstimationScopeV3(CatalogueV2)
	case ast.LibV3:
		r.scope = newEstimationScopeV3(CatalogueV3)
	case ast.LibV4:
		r.scope = newEstimationScopeV3(CatalogueV4)
	case ast.LibV5:
		r.scope = newEstimationScopeV3(CatalogueV5)
	case ast.LibV6:
		r.scope = newEstimationScopeV3(CatalogueV6)
	default:
		return nil, errors.Errorf("unsupported library version %d", tree.LibVersion)
	}
	return r, nil
}

func (e *treeEstimatorV4) estimate() (int, int, map[string]int, error) {
	if !e.tree.IsDApp() {
		e.scope.submerge()
		c, inv, err := e.walk(e.tree.Verifier)
		if err != nil {
			return 0, 0, nil, err
		}
		if inv {
			return 0, 0, nil, errors.New("usage of invocation functions is prohibited in expressions")
		}
		e.scope.emerge()
		return c, c, nil, nil
	}
	max := 0
	m := make(map[string]int)
	for i := 0; i < len(e.tree.Functions); i++ {
		e.scope.resetFunctions()
		function, ok := e.tree.Functions[i].(*ast.FunctionDeclarationNode)
		if !ok {
			return 0, 0, nil, errors.New("invalid callable declaration")
		}
		e.scope.submerge()
		c, _, err := e.walk(e.wrapFunction(function))
		if err != nil {
			return 0, 0, nil, err
		}
		e.scope.emerge()
		m[function.Name] = c
		if c > max {
			max = c
		}
	}
	vc := 0
	if e.tree.HasVerifier() {
		e.scope.resetFunctions()
		verifier, ok := e.tree.Verifier.(*ast.FunctionDeclarationNode)
		if !ok {
			return 0, 0, nil, errors.New("invalid verifier declaration")
		}
		e.scope.submerge()
		c, inv, err := e.walk(e.wrapFunction(verifier))
		if err != nil {
			return 0, 0, nil, err
		}
		if inv {
			return 0, 0, nil, errors.New("usage of invocation functions is prohibited in verifier")
		}
		e.scope.emerge()
		vc = c
		if c > max {
			max = c
		}
	}
	return max, vc, m, nil
}

func (e *treeEstimatorV4) wrapFunction(node *ast.FunctionDeclarationNode) ast.Node {
	args := make([]ast.Node, len(node.Arguments))
	for i := range node.Arguments {
		args[i] = ast.NewBooleanNode(true)
	}
	// It's definitely user function as node is sent from FunctionDeclarationNode
	node.SetBlock(ast.NewFunctionCallNode(ast.UserFunction(node.Name), args))
	var block ast.Node
	block = ast.NewAssignmentNode(node.InvocationParameter, ast.NewBooleanNode(true), node)
	for i := len(e.tree.Declarations) - 1; i >= 0; i-- {
		e.tree.Declarations[i].SetBlock(block)
		block = e.tree.Declarations[i]
	}
	return block
}

// walk function iterates over AST and calculates an estimation of every node.
// Function returns the cumulative cost of a node's subtree,
// the bool indicator of invocation function usage in the node's subtree and error if any.
func (e *treeEstimatorV4) walk(node ast.Node) (int, bool, error) {
	switch n := node.(type) {
	case *ast.LongNode, *ast.BytesNode, *ast.BooleanNode, *ast.StringNode:
		return 0, false, nil

	case *ast.ConditionalNode:
		ce, ci, err := e.walk(n.Condition)
		if err != nil {
			return 0, false, errors.Wrap(err, "failed to estimate the condition of if")
		}
		cs := e.scope.save()
		le, li, err := e.walk(n.TrueExpression)
		if err != nil {
			return 0, false, errors.Wrap(err, "failed to estimate the true branch of if")
		}
		ls := e.scope.save()
		e.scope.restore(cs)
		re, ri, err := e.walk(n.FalseExpression)
		if err != nil {
			return 0, false, errors.Wrap(err, "failed to estimate the false branch of if")
		}
		if le > re {
			e.scope.restore(ls)
			res, err := common.AddInt(ce, le)
			if err != nil {
				return 0, false, err
			}
			return res, ci || li, nil
		}
		res, err := common.AddInt(ce, re)
		if err != nil {
			return 0, false, err
		}
		return res, ci || ri, nil

	case *ast.AssignmentNode:
		id := n.Name
		overlapped := e.scope.used(id)
		e.scope.remove(id)
		c, inv, err := e.walk(n.Block)
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to estimate block after declaration of variable '%s'", id)
		}
		if e.scope.used(id) {
			tmp := e.scope.save()
			le, li, err := e.walk(n.Expression)
			if err != nil {
				return 0, false, errors.Wrap(err, "failed to estimate let expression")
			}
			e.scope.restore(tmp)
			inv = inv || li
			c, err = common.AddInt(c, le)
			if err != nil {
				return 0, false, err
			}
		}
		if overlapped {
			e.scope.use(id)
		} else {
			e.scope.remove(id)
		}
		return c, inv, nil

	case *ast.ReferenceNode:
		e.scope.use(n.Name)
		return 0, false, nil

	case *ast.FunctionDeclarationNode:
		id := n.Name
		tmp := e.scope.save()
		e.scope.submerge()
		fc, bi, err := e.walk(n.Body)
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to estimate cost of function '%s'", id)
		}
		bodyUsages := e.scope.emerge()
		e.scope.restore(tmp)
		if e.scope.setFunction(id, fc, bodyUsages, bi) {
			return 0, false, errors.Errorf("function '%s' already declared", id)
		}
		bc, inv, err := e.walk(n.Block)
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to estimate block after declaration of function '%s'", id)
		}
		return bc, inv, nil

	case *ast.FunctionCallNode:
		name := n.Function.Name()
		fd, err := e.scope.function(n.Function)
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to estimate the call of function '%s'", name)
		}
		for _, u := range fd.usages {
			e.scope.use(u)
		}
		ac := 0
		inv := fd.callsInvocation
		for i, a := range n.Arguments {
			tmp := e.scope.save()
			c, ai, err := e.walk(a)
			if err != nil {
				return 0, false, errors.Wrapf(err, "failed to estimate parameter %d of function call '%s'", i, name)
			}
			inv = inv || ai
			e.scope.restore(tmp)
			ac, err = common.AddInt(ac, c)
			if err != nil {
				return 0, false, err
			}
		}
		fc := fd.cost
		if fc == 0 {
			fc = 1
		}
		res, err := common.AddInt(fc, ac)
		if err != nil {
			return 0, false, err
		}
		return res, inv, nil

	case *ast.PropertyNode:
		res, inv, err := e.walk(n.Object)
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to estimate getter '%s'", n.Name)
		}
		return res, inv, nil

	default:
		return 0, false, errors.Errorf("unsupported type of node '%T'", node)
	}
}
