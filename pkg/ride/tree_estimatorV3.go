package ride

import (
	"github.com/pkg/errors"
)

type fd struct {
	cost   int
	usages []string
}
type fsV3 struct {
	parent    *fsV3
	functions map[string]fd
}

func newFsV3() *fsV3 {
	return &fsV3{parent: nil, functions: make(map[string]fd)}
}

func (f *fsV3) spawn() *fsV3 {
	return &fsV3{
		parent:    f,
		functions: make(map[string]fd),
	}
}

func (f *fsV3) set(key string, cost int, usages []string) {
	f.functions[key] = fd{cost, usages}
}

func (f *fsV3) get(key string) (int, []string, bool) {
	fd, ok := f.functions[key]
	if !ok {
		if f.parent == nil {
			return 0, nil, false
		}
		return f.parent.get(key)
	}
	return fd.cost, fd.usages, true
}

type estimationScopeV3 struct {
	usages    [][]string
	functions *fsV3
	builtin   map[string]int
}

func newEstimationScopeV3(functions map[string]int) *estimationScopeV3 {
	return &estimationScopeV3{
		usages:    make([][]string, 0),
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

func (s *estimationScopeV3) setFunction(id string, cost int, usages []string) {
	s.functions.set(id, cost, usages)
}

func (s *estimationScopeV3) function(function function, enableInvocation bool) (int, []string, error) {
	id := function.Name()
	switch function.(type) {
	case userFunction:
		cost, usages, found := s.functions.get(id)
		if found {
			return cost, usages, nil
		}
		if c, ok := s.builtin[id]; ok {
			if (id == "1020" || id == "1021") && !enableInvocation {
				return 0, nil, errors.Errorf("user function '%s' not found", id)
			}
			return c, nil, nil
		}
		return 0, nil, errors.Errorf("user function '%s' not found", id)
	case nativeFunction:
		if c, ok := s.builtin[id]; ok {
			if (id == "1020" || id == "1021") && !enableInvocation {
				return 0, nil, errors.Errorf("function '%s' not found", id)
			}
			return c, nil, nil
		}
		return 0, nil, errors.Errorf("native function '%s' not found", id)
	default:
		return 0, nil, errors.Errorf("unknown type of function '%s'", id)
	}
}

func (s *estimationScopeV3) used(id string) bool {
	if l := len(s.usages); l > 0 {
		for i := len(s.usages[l-1]) - 1; i >= 0; i-- {
			if s.usages[l-1][i] == id {
				return true
			}
		}
	}
	return false
}

func (s *estimationScopeV3) remove(id string) {
	if l := len(s.usages); l > 0 {
		for i := len(s.usages[l-1]) - 1; i >= 0; i-- {
			if s.usages[l-1][i] == id {
				s.usages[l-1] = append(s.usages[l-1][:i], s.usages[l-1][i+1:]...)
			}
		}
	}
}

func (s *estimationScopeV3) use(id string) {
	if l := len(s.usages); l > 0 {
		s.usages[l-1] = append(s.usages[l-1], id)
	}
}

func (s *estimationScopeV3) submerge() {
	s.usages = append(s.usages, make([]string, 0))
}

func (s *estimationScopeV3) emerge() []string {
	if l := len(s.usages); l > 0 {
		var r []string
		r, s.usages = s.usages[l-1], s.usages[:l-1]
		return r
	}
	return nil
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
	case 5:
		r.scope = newEstimationScopeV3(CatalogueV5)
	default:
		return nil, errors.Errorf("unsupported library version %d", tree.LibVersion)
	}
	return r, nil
}

func (e *treeEstimatorV3) estimate() (int, int, map[string]int, error) {
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

func (e *treeEstimatorV3) wrapFunction(node *FunctionDeclarationNode) Node {
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

func (e *treeEstimatorV3) walk(node Node, enableInvocation bool) (int, error) {
	switch n := node.(type) {
	case *LongNode, *BytesNode, *BooleanNode, *StringNode:
		return 1, nil

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
			return ce + le + 1, nil
		}
		return ce + re + 1, nil

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
			c = c + le
		}
		if overlapped {
			e.scope.use(id)
		} else {
			e.scope.remove(id)
		}
		return c, nil

	case *ReferenceNode:
		e.scope.use(n.Name)
		return 1, nil

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
		e.scope.setFunction(id, fc, bodyUsages)
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
			ac += c
		}
		return fc + ac, nil

	case *PropertyNode:
		c, err := e.walk(n.Object, enableInvocation)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to estimate getter '%s'", n.Name)
		}
		return c + 1, nil

	default:
		return 0, errors.Errorf("unsupported type of node '%T'", node)
	}
}
