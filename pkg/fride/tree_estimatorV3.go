package fride

import (
	"github.com/pkg/errors"
)

type fsV3 struct {
	parent    *fsV3
	functions map[string]int
}

func newFsV3() *fsV3 {
	return &fsV3{parent: nil, functions: make(map[string]int)}
}

func (f *fsV3) spawn() *fsV3 {
	return &fsV3{
		parent:    f,
		functions: make(map[string]int),
	}
}

func (f *fsV3) set(key string, cost int) {
	f.functions[key] = cost
}

func (f *fsV3) get(key string) (int, error) {
	function, ok := f.functions[key]
	if !ok {
		if f.parent == nil {
			return 0, errors.Errorf("user function '%s' not found", key)
		}
		return f.parent.get(key)
	}
	return function, nil
}

type estimationScopeV3 struct {
	used      map[string]struct{}
	functions *fsV3
	builtin   map[string]int
}

func newEstimationScopeV3(functions map[string]int) *estimationScopeV3 {
	return &estimationScopeV3{
		used:      make(map[string]struct{}),
		functions: newFsV3(),
		builtin:   functions,
	}
}

func (s *estimationScopeV3) save() *fsV3 {
	r := s.functions
	s.functions = s.functions.spawn()
	return r
}

func (s *estimationScopeV3) restore(fs *fsV3) {
	s.functions = fs
}

func (s *estimationScopeV3) setFunction(id string, cost int) {
	s.functions.set(id, cost)
}

func (s *estimationScopeV3) function(id string) (int, error) {
	if c, ok := s.builtin[id]; ok {
		return c, nil
	}
	return s.functions.get(id)
}

type treeEstimatorV3 struct {
	tree  *Tree
	scope *estimationScopeV3
}

func newTreeEstimatorV3(tree *Tree) (*treeEstimatorV3, error) {
	r := &treeEstimatorV3{tree: tree}
	switch tree.LibVersion {
	case 1, 2:
		r.scope = newEstimationScopeV3(CatalogueV2)
	case 3:
		r.scope = newEstimationScopeV3(CatalogueV3)
	case 4:
		r.scope = newEstimationScopeV3(CatalogueV4)
	default:
		return nil, errors.Errorf("unsupported library version %d", tree.LibVersion)
	}
	return r, nil
}

func (e *treeEstimatorV3) estimate() (int, int, map[string]int, error) {
	if !e.tree.IsDApp() {
		c, err := e.walk(e.tree.Verifier)
		if err != nil {
			return 0, 0, nil, err
		}
		return c, c, nil, nil
	}
	max := 0
	m := make(map[string]int)
	for i := 0; i < len(e.tree.Functions); i++ {
		function, ok := e.tree.Functions[i].(*FunctionDeclarationNode)
		if !ok {
			return 0, 0, nil, errors.New("invalid callable declaration")
		}
		c, err := e.walk(e.wrapFunction(function))
		if err != nil {
			return 0, 0, nil, err
		}
		m[function.Name] = c
		if c > max {
			max = c
		}
	}
	vc := 0
	if e.tree.HasVerifier() {
		verifier, ok := e.tree.Verifier.(*FunctionDeclarationNode)
		if !ok {
			return 0, 0, nil, errors.New("invalid verifier declaration")
		}
		c, err := e.walk(e.wrapFunction(verifier))
		if err != nil {
			return 0, 0, nil, err
		}
		vc = c
		if c > max {
			max = c
		}
	}
	return max, vc, m, nil
}

func (e *treeEstimatorV3) wrapFunction(node *FunctionDeclarationNode) Node {
	args := make([]Node, len(node.Arguments))
	for i := range node.Arguments {
		args[i] = NewBooleanNode(true)
	}
	node.SetBlock(NewFunctionCallNode(node.Name, args))
	var block Node
	block = NewAssignmentNode(node.invocationParameter, NewBooleanNode(true), node)
	for i := len(e.tree.Declarations) - 1; i >= 0; i-- {
		e.tree.Declarations[i].SetBlock(block)
		block = e.tree.Declarations[i]
	}
	return block
}

func (e *treeEstimatorV3) walk(node Node) (int, error) {
	switch n := node.(type) {
	case *LongNode, *BytesNode, *BooleanNode, *StringNode:
		return 1, nil

	case *ConditionalNode:
		ce, err := e.walk(n.Condition)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the condition of if")
		}
		cs := e.scope.save()
		le, err := e.walk(n.TrueExpression)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the true branch of if")
		}
		ls := e.scope.save()
		e.scope.restore(cs)
		re, err := e.walk(n.FalseExpression)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the false branch of if")
		}
		if le > re {
			e.scope.restore(ls)
			return ce + le + 1, nil
		}
		return ce + re + 1, nil

	case *AssignmentNode:
		id := n.Name
		_, overlapped := e.scope.used[id]
		delete(e.scope.used, id)
		c, err := e.walk(n.Block)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate block after declaration of variable '%s'", id)
		}
		if _, used := e.scope.used[id]; used {
			tmp := e.scope.save()
			le, err := e.walk(n.Expression)
			if err != nil {
				return 0, errors.Wrap(err, "failed to estimate let expression")
			}
			e.scope.restore(tmp)
			c = c + le
		}
		if overlapped {
			e.scope.used[id] = struct{}{}
		} else {
			delete(e.scope.used, id)
		}
		return c, nil

	case *ReferenceNode:
		e.scope.used[n.Name] = struct{}{}
		return 1, nil

	case *FunctionDeclarationNode:
		id := n.Name
		tmp := e.scope.save()
		fc, err := e.walk(n.Body)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate cost of function '%s'", id)
		}
		e.scope.restore(tmp)
		e.scope.setFunction(id, fc)
		bc, err := e.walk(n.Block)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate block after declaration of function '%s'", id)
		}
		return bc, nil

	case *FunctionCallNode:
		id := n.Name
		fc, err := e.scope.function(id)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate the call of function '%s'", id)
		}
		ac := 0
		for i, a := range n.Arguments {
			tmp := e.scope.save()
			c, err := e.walk(a)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to estimate parameter %d of function call '%s'", i, id)
			}
			e.scope.restore(tmp)
			ac += c
		}
		return fc + ac, nil

	case *PropertyNode:
		c, err := e.walk(n.Object)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate getter '%s'", n.Name)
		}
		return c + 1, nil

	default:
		return 0, errors.Errorf("unsupported type of node '%T'", node)
	}
}
