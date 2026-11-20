package ride

import (
	"slices"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type scopeValue struct {
	id string
	n  ast.Node
}

// estimationScopeV1 keeps the declared values as a stack, so save and restore are plain truncations and the last
// declaration wins on lookup.
type estimationScopeV1 struct {
	// constants are the predefined values of the library version. They lie below all declared values, never
	// change and have no expression, so they are not copied into values.
	constants []string
	values    []scopeValue
	// branches holds the values declared in the true branches of the conditionals in progress.
	branches  []scopeValue
	functions map[string]int
	builtin   map[string]int
	// trueNode is the shared constant expression of the function arguments, the estimator only reads it.
	trueNode ast.BooleanNode
}

func newEstimationScopeV1(values []string, functions map[string]int) *estimationScopeV1 {
	return &estimationScopeV1{
		constants: values,
		builtin:   functions,
		trueNode:  ast.BooleanNode{Value: true},
	}
}

func (s *estimationScopeV1) setValue(id string, n ast.Node) {
	if s.values == nil {
		const initialValuesCapacity = 16
		s.values = make([]scopeValue, 0, initialValuesCapacity)
	}
	s.values = append(s.values, scopeValue{id: id, n: n})
}

func (s *estimationScopeV1) value(id string) (ast.Node, error) {
	for _, v := range slices.Backward(s.values) {
		if v.id == id {
			return v.n, nil
		}
	}
	if slices.Contains(s.constants, id) {
		return nil, nil //nolint:nilnil // A constant has no node, callers expect nil without an error.
	}
	return nil, errors.Errorf("value '%s' not found", id)
}

func (s *estimationScopeV1) save() int {
	return len(s.values)
}

func (s *estimationScopeV1) restore(p int) {
	if p < len(s.values) {
		s.values = s.values[:p]
	}
}

// stashBranch moves the values declared after the position p to the branches stack and returns the position of
// the moved values there.
func (s *estimationScopeV1) stashBranch(p int) int {
	b := len(s.branches)
	if p < len(s.values) {
		s.branches = append(s.branches, s.values[p:]...)
		s.values = s.values[:p]
	}
	return b
}

// dropBranch removes the values stashed at the position b from the branches stack.
func (s *estimationScopeV1) dropBranch(b int) {
	s.branches = s.branches[:b]
}

// unstashBranch replaces the values declared after the position p with the values stashed at the position b.
func (s *estimationScopeV1) unstashBranch(p, b int) {
	s.restore(p)
	if b < len(s.branches) {
		s.values = append(s.values, s.branches[b:]...)
	}
	s.dropBranch(b)
}

func (s *estimationScopeV1) setFunction(id string, cost int) {
	if s.functions == nil {
		s.functions = make(map[string]int)
	}
	s.functions[id] = cost
}

func (s *estimationScopeV1) function(function ast.Function) (int, error) {
	id := function.Name()
	switch function.(type) {
	case ast.UserFunction:
		if c, ok := s.functions[id]; ok {
			return c, nil
		}
		if c, ok := s.builtin[id]; ok {
			return c, nil
		}
		return 0, errors.Errorf("function '%s' not found", id)
	case ast.NativeFunction:
		if c, ok := s.builtin[id]; ok {
			return c, nil
		}
		return 0, errors.Errorf("system function '%s' not found", id)
	default:
		return 0, errors.Errorf("function '%s' not found", id)
	}
}

type treeEstimatorV1 struct {
	tree  *ast.Tree
	scope *estimationScopeV1
}

func newTreeEstimatorV1(tree *ast.Tree) (*treeEstimatorV1, error) {
	r := &treeEstimatorV1{tree: tree}
	switch tree.LibVersion {
	case ast.LibV1:
		r.scope = newEstimationScopeV1(ConstantsV1, CatalogueV2)
	case ast.LibV2:
		r.scope = newEstimationScopeV1(ConstantsV2, CatalogueV2)
	case ast.LibV3:
		r.scope = newEstimationScopeV1(ConstantsV3, CatalogueV3)
	case ast.LibV4:
		r.scope = newEstimationScopeV1(ConstantsV4, CatalogueV4)
	case ast.LibV5:
		r.scope = newEstimationScopeV1(ConstantsV5, CatalogueV5)
	case ast.LibV6:
		r.scope = newEstimationScopeV1(ConstantsV6, CatalogueV6)
	case ast.LibV7:
		r.scope = newEstimationScopeV1(ConstantsV7, CatalogueV7)
	case ast.LibV8:
		r.scope = newEstimationScopeV1(ConstantsV8, CatalogueV8)
	case ast.LibV9:
		r.scope = newEstimationScopeV1(ConstantsV9, CatalogueV9)
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
		function, ok := e.tree.Functions[i].(*ast.FunctionDeclarationNode)
		if !ok {
			return 0, 0, nil, errors.New("invalid callable declaration")
		}
		c, err := e.walk(wrapCallable(e.tree.Declarations, function))
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
		verifier, ok := e.tree.Verifier.(*ast.FunctionDeclarationNode)
		if !ok {
			return 0, 0, nil, errors.New("invalid verifier declaration")
		}
		c, err := e.walk(wrapCallable(e.tree.Declarations, verifier))
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

func (e *treeEstimatorV1) walk(node ast.Node) (int, error) {
	switch n := node.(type) {
	case *ast.LongNode, *ast.BytesNode, *ast.BooleanNode, *ast.StringNode:
		return 1, nil

	case *ast.ConditionalNode:
		ce, err := e.walk(n.Condition)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the condition of if")
		}
		csp := e.scope.save()
		te, err := e.walk(n.TrueExpression)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the true branch of if")
		}
		tsp := e.scope.stashBranch(csp)
		fe, err := e.walk(n.FalseExpression)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the false branch of if")
		}
		if te > fe {
			e.scope.unstashBranch(csp, tsp)
			return ce + te + 1, nil
		}
		e.scope.dropBranch(tsp)
		return ce + fe + 1, nil

	case *ast.AssignmentNode:
		id := n.Name
		e.scope.setValue(id, n.Expression)
		c, err := e.walk(n.Block)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate block after declaration of variable '%s'", id)
		}
		return c + 5, nil

	case *ast.ReferenceNode:
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

	case *ast.FunctionDeclarationNode:
		id := n.Name
		tmp := e.scope.save()
		for _, a := range n.Arguments {
			e.scope.setValue(a, &e.scope.trueNode)
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

	case *ast.FunctionCallNode:
		id := n.Function.Name()
		function := n.Function
		fc, err := e.scope.function(function)
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

	case *ast.PropertyNode:
		c, err := e.walk(n.Object)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate getter '%s'", n.Name)
		}
		return c + 2, nil

	default:
		return 0, errors.Errorf("unsupported type of node '%T'", node)
	}
}
