package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type treeEstimatorV4 struct {
	tree  *Tree
	scope *estimationScopeV3
}

func newTreeEstimatorV4(tree *Tree) (*treeEstimatorV4, error) {
	r := &treeEstimatorV4{tree: tree}
	switch tree.LibVersion {
	case 1, 2:
		r.scope = newEstimationScopeV3(CatalogueV2)
	case 3:
		r.scope = newEstimationScopeV3(CatalogueV3)
	case 4:
		r.scope = newEstimationScopeV3(CatalogueV4)
	case 5:
		r.scope = newEstimationScopeV3(CatalogueV5)
	case 6:
		r.scope = newEstimationScopeV3(CatalogueV6)
	default:
		return nil, errors.Errorf("unsupported library version %d", tree.LibVersion)
	}
	return r, nil
}

func (e *treeEstimatorV4) estimate() (int, int, map[string]int, error) {
	if !e.tree.IsDApp() {
		e.scope.submerge()
		c, err := e.walk(e.tree.Verifier, false)
		if err != nil {
			return 0, 0, nil, err
		}
		e.scope.emerge()
		return c, c, nil, nil
	}
	max := 0
	m := make(map[string]int)
	for i := 0; i < len(e.tree.Functions); i++ {
		e.scope.resetFunctions()
		function, ok := e.tree.Functions[i].(*FunctionDeclarationNode)
		if !ok {
			return 0, 0, nil, errors.New("invalid callable declaration")
		}
		e.scope.submerge()
		c, err := e.walk(e.wrapFunction(function), true)
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
		verifier, ok := e.tree.Verifier.(*FunctionDeclarationNode)
		if !ok {
			return 0, 0, nil, errors.New("invalid verifier declaration")
		}
		e.scope.submerge()
		c, err := e.walk(e.wrapFunction(verifier), false)
		if err != nil {
			return 0, 0, nil, err
		}
		e.scope.emerge()
		vc = c
		if c > max {
			max = c
		}
	}
	return max, vc, m, nil
}

func (e *treeEstimatorV4) wrapFunction(node *FunctionDeclarationNode) Node {
	args := make([]Node, len(node.Arguments))
	for i := range node.Arguments {
		args[i] = NewBooleanNode(true)
	}
	// It's definitely user function as node is sent from FunctionDeclarationNode
	node.SetBlock(NewFunctionCallNode(userFunction(node.Name), args))
	var block Node
	block = NewAssignmentNode(node.invocationParameter, NewBooleanNode(true), node)
	for i := len(e.tree.Declarations) - 1; i >= 0; i-- {
		e.tree.Declarations[i].SetBlock(block)
		block = e.tree.Declarations[i]
	}
	return block
}

func (e *treeEstimatorV4) walk(node Node, enableInvocation bool) (int, error) {
	switch n := node.(type) {
	case *LongNode, *BytesNode, *BooleanNode, *StringNode:
		return 0, nil

	case *ConditionalNode:
		ce, err := e.walk(n.Condition, enableInvocation)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the condition of if")
		}
		cs := e.scope.save()
		le, err := e.walk(n.TrueExpression, enableInvocation)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the true branch of if")
		}
		ls := e.scope.save()
		e.scope.restore(cs)
		re, err := e.walk(n.FalseExpression, enableInvocation)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the false branch of if")
		}
		if le > re {
			e.scope.restore(ls)
			res, err := common.AddInt(ce, le)
			if err != nil {
				return 0, err
			}
			return res, nil
		}
		res, err := common.AddInt(ce, re)
		if err != nil {
			return 0, err
		}
		return res, nil

	case *AssignmentNode:
		id := n.Name
		overlapped := e.scope.used(id)
		e.scope.remove(id)
		c, err := e.walk(n.Block, enableInvocation)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate block after declaration of variable '%s'", id)
		}
		if e.scope.used(id) {
			tmp := e.scope.save()
			le, err := e.walk(n.Expression, enableInvocation)
			if err != nil {
				return 0, errors.Wrap(err, "failed to estimate let expression")
			}
			e.scope.restore(tmp)
			c, err = common.AddInt(c, le)
			if err != nil {
				return 0, err
			}
		}
		if overlapped {
			e.scope.use(id)
		} else {
			e.scope.remove(id)
		}
		return c, nil

	case *ReferenceNode:
		e.scope.use(n.Name)
		return 0, nil

	case *FunctionDeclarationNode:
		id := n.Name
		tmp := e.scope.save()
		e.scope.submerge()
		fc, err := e.walk(n.Body, enableInvocation)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate cost of function '%s'", id)
		}
		bodyUsages := e.scope.emerge()
		e.scope.restore(tmp)
		if e.scope.setFunction(id, fc, bodyUsages) {
			return 0, errors.Errorf("function '%s' already declared", id)
		}
		bc, err := e.walk(n.Block, enableInvocation)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate block after declaration of function '%s'", id)
		}
		return bc, nil

	case *FunctionCallNode:
		name := n.Function.Name()
		fc, bu, err := e.scope.function(n.Function, enableInvocation)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate the call of function '%s'", name)
		}
		for _, u := range bu {
			e.scope.use(u)
		}
		ac := 0
		for i, a := range n.Arguments {
			tmp := e.scope.save()
			c, err := e.walk(a, enableInvocation)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to estimate parameter %d of function call '%s'", i, name)
			}
			e.scope.restore(tmp)
			ac, err = common.AddInt(ac, c)
			if err != nil {
				return 0, err
			}
		}
		if fc == 0 {
			fc = 1
		}
		res, err := common.AddInt(fc, ac)
		if err != nil {
			return 0, err
		}
		return res, nil

	case *PropertyNode:
		res, err := e.walk(n.Object, enableInvocation)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate getter '%s'", n.Name)
		}
		return res, nil

	default:
		return 0, errors.Errorf("unsupported type of node '%T'", node)
	}
}
