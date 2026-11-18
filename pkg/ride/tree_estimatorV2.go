package ride

import (
	"slices"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

// estimationScopeV2 keeps the declared values, the user functions and the overlapped values as stacks, so
// save and restore are plain truncations and the last declaration wins on lookup.
type estimationScopeV2 struct {
	// constants are the predefined values of the library version. They lie below all declared values, never
	// change and have no expression, so they are not copied into values.
	constants []string
	values    []scopeValue
	stash     []scopeValue
	functions []*ast.FunctionDeclarationNode
	builtin   map[string]int
	// args holds the values overlapped by the arguments of the user function calls in progress.
	args []scopeValue
	// seen is the reusable set of the names re-stacked from the stash.
	seen map[string]struct{}
	// trueNode is the shared constant expression of the function arguments, the estimator only reads it.
	trueNode ast.BooleanNode
}

func newEstimationScopeV2(values []string, functions map[string]int) *estimationScopeV2 {
	return &estimationScopeV2{
		constants: values,
		builtin:   functions,
		trueNode:  ast.BooleanNode{Value: true},
	}
}

func (s *estimationScopeV2) setValue(id string, n ast.Node) {
	if s.values == nil {
		const initialValuesCapacity = 16
		s.values = make([]scopeValue, 0, initialValuesCapacity)
	}
	s.values = append(s.values, scopeValue{id: id, n: n})
}

// lookup returns the expression of the last declared value with the given name.
func (s *estimationScopeV2) lookup(id string) (ast.Node, bool) {
	for _, v := range slices.Backward(s.values) {
		if v.id == id {
			return v.n, true
		}
	}
	if slices.Contains(s.constants, id) {
		return nil, true
	}
	return nil, false
}

func (s *estimationScopeV2) value(id string) (ast.Node, error) {
	if n, ok := s.lookup(id); ok {
		return n, nil
	}
	return nil, errors.Errorf("value '%s' not found", id)
}

func (s *estimationScopeV2) deleteValue(id string, limit int) {
	for i := len(s.values) - 1; i >= limit; i-- {
		if s.values[i].id == id {
			s.values = append(s.values[:i], s.values[i+1:]...)
		}
	}
}

func (s *estimationScopeV2) setFunction(n *ast.FunctionDeclarationNode) {
	s.functions = append(s.functions, n)
}

func (s *estimationScopeV2) function(id string) (*ast.FunctionDeclarationNode, error) {
	for _, f := range slices.Backward(s.functions) {
		if f.Name == id {
			return f, nil
		}
	}
	return nil, errors.Errorf("function '%s' not found", id)
}

// restackOverlaps declares again the last stashed value of every overlapped name. The names are distinct, so
// the order in which they are declared does not change the result of any lookup.
func (s *estimationScopeV2) restackOverlaps() {
	switch len(s.stash) {
	case 0:
		return
	case 1:
		s.setValue(s.stash[0].id, s.stash[0].n)
		return
	}
	if s.seen == nil {
		s.seen = make(map[string]struct{}, len(s.stash))
	} else {
		clear(s.seen)
	}
	for _, o := range slices.Backward(s.stash) {
		if _, ok := s.seen[o.id]; ok {
			continue
		}
		s.seen[o.id] = struct{}{}
		s.setValue(o.id, o.n)
	}
}

func (s *estimationScopeV2) addOverlaps(overlaps []scopeValue) {
	s.stash = append(s.stash, overlaps...)
}

func (s *estimationScopeV2) setOverlap(id string, n ast.Node) {
	s.stash = append(s.stash, scopeValue{id: id, n: n})
}

func (s *estimationScopeV2) save() (int, int, int) {
	return len(s.values), len(s.functions), len(s.stash)
}

func (s *estimationScopeV2) restore(pv, pf, po int) {
	s.values = s.values[:pv]
	s.functions = s.functions[:pf]
	s.stash = s.stash[:po]
}

// isOverlapped reports whether the name is among the overlapped values.
func isOverlapped(overlapped []scopeValue, id string) bool {
	for _, o := range overlapped {
		if o.id == id {
			return true
		}
	}
	return false
}

type treeEstimatorV2 struct {
	tree  *ast.Tree
	scope *estimationScopeV2
}

func newTreeEstimatorV2(tree *ast.Tree) (*treeEstimatorV2, error) {
	r := &treeEstimatorV2{tree: tree}
	switch tree.LibVersion {
	case ast.LibV1:
		r.scope = newEstimationScopeV2(ConstantsV1, CatalogueV2)
	case ast.LibV2:
		r.scope = newEstimationScopeV2(ConstantsV2, CatalogueV2)
	case ast.LibV3:
		r.scope = newEstimationScopeV2(ConstantsV3, CatalogueV3)
	case ast.LibV4:
		r.scope = newEstimationScopeV2(ConstantsV4, CatalogueV4)
	case ast.LibV5:
		r.scope = newEstimationScopeV2(ConstantsV5, CatalogueV5)
	case ast.LibV6:
		r.scope = newEstimationScopeV2(ConstantsV6, CatalogueV6)
	case ast.LibV7:
		r.scope = newEstimationScopeV2(ConstantsV7, CatalogueV7)
	case ast.LibV8:
		r.scope = newEstimationScopeV2(ConstantsV8, CatalogueV8)
	case ast.LibV9:
		r.scope = newEstimationScopeV2(ConstantsV9, CatalogueV9)
	default:
		return nil, errors.Errorf("unsupported library version %d", tree.LibVersion)
	}
	return r, nil
}

func (e *treeEstimatorV2) estimate() (int, int, map[string]int, error) {
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
		vp, fp, op := e.scope.save()
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
		e.scope.restore(vp, fp, op)
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

func (e *treeEstimatorV2) walk(node ast.Node) (int, error) {
	switch n := node.(type) {
	case *ast.LongNode, *ast.BytesNode, *ast.BooleanNode, *ast.StringNode:
		return 1, nil

	case *ast.ConditionalNode:
		ce, err := e.walk(n.Condition)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the condition of if")
		}
		te, err := e.walk(n.TrueExpression)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the true branch of if")
		}
		fe, err := e.walk(n.FalseExpression)
		if err != nil {
			return 0, errors.Wrap(err, "failed to estimate the false branch of if")
		}
		if te > fe {
			return ce + te + 1, nil
		}
		return ce + fe + 1, nil

	case *ast.AssignmentNode:
		id := n.Name
		vp, fp, op := e.scope.save()
		e.scope.setValue(id, n.Expression)
		c, err := e.walk(n.Block)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate block after declaration of variable '%s'", id)
		}
		e.scope.restore(vp, fp, op)
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
		e.scope.setFunction(n)
		bc, err := e.walk(n.Block)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate block after declaration of function '%s'", id)
		}
		return bc + 5, nil

	case *ast.FunctionCallNode:
		id := n.Function.Name()
		ac := 0
		for i, a := range n.Arguments {
			c, err := e.walk(a)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to estimate parameter %d of function call '%s'", i, id)
			}
			ac += c
		}
		fc, ok := e.scope.builtin[id]
		if ok { // For a built-in function return estimation
			return fc + ac, nil
		}

		fn, err := e.scope.function(id)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate the call of function '%s'", id)
		}

		// User function call
		// Values overlapped by function arguments are kept in s.args[ab:] until the body is estimated.
		// The arguments that overlap nothing are deleted after the body is estimated.
		s := e.scope
		ab := len(s.args)
		for _, a := range fn.Arguments {
			if vn, found := s.lookup(a); found {
				s.args = append(s.args, scopeValue{id: a, n: vn})
			}
		}

		// Re stack previous overlaps
		s.restackOverlaps()

		l := len(s.values)
		// Stack arguments
		for _, a := range fn.Arguments {
			s.setValue(a, &s.trueNode)
		}

		// Stash current overlaps
		s.addOverlaps(s.args[ab:])

		fc, err = e.walk(fn.Body)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate cost of function '%s'", id)
		}

		overlapped := s.args[ab:]
		for _, o := range overlapped {
			if vn, found := s.lookup(o.id); found {
				s.setOverlap(o.id, vn)
			}
			s.setValue(o.id, o.n)
		}

		for _, a := range fn.Arguments {
			if !isOverlapped(overlapped, a) {
				s.deleteValue(a, l)
			}
		}
		s.args = s.args[:ab]

		ac += len(fn.Arguments) * 5
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
