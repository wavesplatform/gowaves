package ride

import (
	"github.com/pkg/errors"
)

type scopeValue struct {
	id string
	n  Node
}

type estimationScopeV1 struct {
	values    []scopeValue
	functions map[string]int
	builtin   map[string]int
}

func newEstimationScopeV1(values []string, functions map[string]int) *estimationScopeV1 {
	initial := make([]scopeValue, len(values))
	for i, id := range values {
		initial[i] = scopeValue{id: id}
	}
	return &estimationScopeV1{
		values:    initial,
		functions: make(map[string]int),
		builtin:   functions,
	}
}

func (s *estimationScopeV1) setValue(id string, n Node) {
	s.values = append(s.values, scopeValue{id: id, n: n})
}

func (s *estimationScopeV1) value(id string) (Node, error) {
	for i := len(s.values) - 1; i >= 0; i-- {
		if s.values[i].id == id {
			return s.values[i].n, nil
		}
	}
	return nil, errors.Errorf("value '%s' not found", id)
}

func (s *estimationScopeV1) save() int {
	return len(s.values)
}

func (s *estimationScopeV1) restore(p int) []scopeValue {
	l := len(s.values)
	if p < l {
		r := make([]scopeValue, l-p)
		copy(r, s.values[p:])
		s.values = s.values[:p]
		return r
	}
	return nil
}

func (s *estimationScopeV1) setFunction(id string, cost int) {
	s.functions[id] = cost
}

func (s *estimationScopeV1) function(id string) (int, error) {
	if c, ok := s.builtin[id]; ok {
		return c, nil
	}
	if c, ok := s.functions[id]; ok {
		return c, nil
	}
	return 0, errors.Errorf("function '%s' not found", id)
}

type treeEstimatorV1 struct {
	tree  *Tree
	scope *estimationScopeV1
}

func newTreeEstimatorV1(tree *Tree) (*treeEstimatorV1, error) {
	r := &treeEstimatorV1{tree: tree}
	switch tree.LibVersion {
	case 1:
		r.scope = newEstimationScopeV1(ConstantsV1, CatalogueV2)
	case 2:
		r.scope = newEstimationScopeV1(ConstantsV2, CatalogueV2)
	case 3:
		r.scope = newEstimationScopeV1(ConstantsV3, CatalogueV3)
	case 4:
		r.scope = newEstimationScopeV1(ConstantsV4, CatalogueV4)
	default:
		return nil, errors.Errorf("unsupported library version %d", tree.LibVersion)
	}
	return r, nil
}

func (e *treeEstimatorV1) estimate() (int, int, map[string]int, error) {
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
		s := e.scope.save()
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
		e.scope.restore(s)
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

func (e *treeEstimatorV1) wrapFunction(node *FunctionDeclarationNode) Node {
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

func (e *treeEstimatorV1) walk(node Node) (int, error) {
	switch n := node.(type) {
	case *LongNode, *BytesNode, *BooleanNode, *StringNode:
		return 1, nil

	case *ConditionalNode:
		ce, err := e.walk(n.Condition)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the condition of if")
		}
		csp := e.scope.save()
		te, err := e.walk(n.TrueExpression)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the true branch of if")
		}
		tsi := e.scope.restore(csp)
		fe, err := e.walk(n.FalseExpression)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the false branch of if")
		}
		if te > fe {
			e.scope.restore(csp)
			e.scope.values = append(e.scope.values, tsi...)
			return ce + te + 1, nil
		}
		return ce + fe + 1, nil

	case *AssignmentNode:
		id := n.Name
		e.scope.setValue(id, n.Expression)
		c, err := e.walk(n.Block)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate block after declaration of variable '%s'", id)
		}
		return c + 5, nil

	case *ReferenceNode:
		id := n.Name
		v, err := e.scope.value(id)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate reference to '%s'", id)
		}
		if v != nil {
			c, err := e.walk(v)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to estimate expression of '%s'", id)
			}
			e.scope.setValue(id, nil)
			return 2 + c, nil
		}
		return 2, nil

	case *FunctionDeclarationNode:
		id := n.Name
		tmp := e.scope.save()
		for _, a := range n.Arguments {
			e.scope.setValue(a, NewBooleanNode(true))
		}
		fc, err := e.walk(n.Body)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate function '%s'", id)
		}
		ac := len(n.Arguments) * 5
		e.scope.restore(tmp)
		e.scope.setFunction(id, ac+fc)

		bc, err := e.walk(n.Block)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate block after declaration of function '%s'", id)
		}
		return bc + 5, nil

	case *FunctionCallNode:
		id := n.Name
		fc, err := e.scope.function(id)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate the call of function '%s'", id)
		}
		ac := 0
		for i, a := range n.Arguments {
			c, err := e.walk(a)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to estimate parameter %d of function call '%s'", i, id)
			}
			ac += c
		}
		return fc + ac, nil

	case *PropertyNode:
		c, err := e.walk(n.Object)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate getter '%s'", n.Name)
		}
		return c + 2, nil

	default:
		return 0, errors.Errorf("unsupported type of node '%T'", node)
	}
}
