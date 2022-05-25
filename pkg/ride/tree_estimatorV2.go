package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type estimationScopeV2 struct {
	values    []scopeValue
	stash     []scopeValue
	functions []*ast.FunctionDeclarationNode
	builtin   map[string]int
}

func newEstimationScopeV2(values []string, functions map[string]int) *estimationScopeV2 {
	initial := make([]scopeValue, len(values))
	for i, id := range values {
		initial[i] = scopeValue{id: id}
	}
	return &estimationScopeV2{
		values:    initial,
		stash:     make([]scopeValue, 0),
		functions: make([]*ast.FunctionDeclarationNode, 0),
		builtin:   functions,
	}
}

func (s *estimationScopeV2) setValue(id string, n ast.Node) {
	s.values = append(s.values, scopeValue{id: id, n: n})
}

func (s *estimationScopeV2) value(id string) (ast.Node, error) {
	for i := len(s.values) - 1; i >= 0; i-- {
		if s.values[i].id == id {
			return s.values[i].n, nil
		}
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
	for i := len(s.functions) - 1; i >= 0; i-- {
		if s.functions[i].Name == id {
			return s.functions[i], nil
		}
	}
	return nil, errors.Errorf("function '%s' not found", id)
}

func (s *estimationScopeV2) overlaps() map[string]ast.Node {
	r := make(map[string]ast.Node)
	for i := 0; i < len(s.stash); i++ {
		r[s.stash[i].id] = s.stash[i].n
	}
	return r
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
		c, err := e.walk(e.wrapFunction(function))
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

func (e *treeEstimatorV2) wrapFunction(node *ast.FunctionDeclarationNode) ast.Node {
	args := make([]ast.Node, len(node.Arguments))
	for i := range node.Arguments {
		args[i] = ast.NewBooleanNode(true)
	}
	node.SetBlock(ast.NewFunctionCallNode(ast.UserFunction(node.Name), args))
	var block ast.Node
	block = ast.NewAssignmentNode(node.InvocationParameter, ast.NewBooleanNode(true), node)
	for i := len(e.tree.Declarations) - 1; i >= 0; i-- {
		e.tree.Declarations[i].SetBlock(block)
		block = e.tree.Declarations[i]
	}
	return block
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
		overlapped := make([]scopeValue, 0) // Temporary storage for the values that was overlapped by function arguments
		newArguments := make([]string, 0)   // Names of function arguments that overlaps nothing
		for _, a := range fn.Arguments {
			if vn, err := e.scope.value(a); err == nil {
				overlapped = append(overlapped, scopeValue{id: a, n: vn})
			} else {
				newArguments = append(newArguments, a)
			}
		}

		// Re stack previous overlaps
		for id, n := range e.scope.overlaps() {
			e.scope.setValue(id, n)
		}

		l := len(e.scope.values)
		// Stack arguments
		for _, a := range fn.Arguments {
			e.scope.setValue(a, ast.NewBooleanNode(true))
		}

		// Stash current overlaps
		e.scope.addOverlaps(overlapped)

		fc, err = e.walk(fn.Body)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate cost of function '%s'", id)
		}

		for _, o := range overlapped {
			if vn, err := e.scope.value(o.id); err == nil {
				e.scope.setOverlap(o.id, vn)
			}
			e.scope.setValue(o.id, o.n)
		}

		for _, a := range newArguments {
			e.scope.deleteValue(a, l)
		}

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
