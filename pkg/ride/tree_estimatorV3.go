package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type functionDescriptor struct {
	cost            int
	usages          []string
	callsInvocation bool
}
type fsV3 struct {
	parent    *fsV3
	functions map[string]functionDescriptor
}

func newFsV3() *fsV3 {
	return &fsV3{parent: nil, functions: make(map[string]functionDescriptor)}
}

func (f *fsV3) spawn() *fsV3 {
	return &fsV3{
		parent:    f,
		functions: make(map[string]functionDescriptor),
	}
}

// set adds new function descriptor to context, returns true if new function overwrite old one.
func (f *fsV3) set(key string, cost int, usages []string, callsInvocation bool) bool {
	_, ok := f.functions[key]
	f.functions[key] = functionDescriptor{cost: cost, usages: usages, callsInvocation: callsInvocation}
	return ok
}

func (f *fsV3) get(key string) (functionDescriptor, bool) {
	fd, ok := f.functions[key]
	if !ok {
		if f.parent == nil {
			return functionDescriptor{}, false
		}
		return f.parent.get(key)
	}
	return fd, true
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

func (s *estimationScopeV3) setFunction(id string, cost int, usages []string, invocation bool) bool {
	return s.functions.set(id, cost, usages, invocation)
}

func (s *estimationScopeV3) resetFunctions() {
	s.functions = newFsV3()
}

func (s *estimationScopeV3) nativeFunction(ut, id string) (functionDescriptor, error) {
	if c, ok := s.builtin[id]; ok {
		return functionDescriptor{cost: c, callsInvocation: id == "1020" || id == "1021"}, nil
	}
	return functionDescriptor{}, errors.Errorf("%s function '%s' not found", ut, id)
}

func (s *estimationScopeV3) function(function ast.Function) (functionDescriptor, error) {
	id := function.Name()
	switch function.(type) {
	case ast.UserFunction:
		if fd, ok := s.functions.get(id); ok {
			return fd, nil
		}
		return s.nativeFunction("user", id)
	case ast.NativeFunction:
		return s.nativeFunction("native", id)
	default:
		return functionDescriptor{}, errors.Errorf("unknown type of function '%s'", id)
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
	tree  *ast.Tree
	scope *estimationScopeV3
}

func newTreeEstimatorV3(tree *ast.Tree) (*treeEstimatorV3, error) {
	r := &treeEstimatorV3{tree: tree}
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

func (e *treeEstimatorV3) estimate() (int, int, map[string]int, error) {
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

func (e *treeEstimatorV3) wrapFunction(node *ast.FunctionDeclarationNode) ast.Node {
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
func (e *treeEstimatorV3) walk(node ast.Node) (int, bool, error) {
	switch n := node.(type) {
	case *ast.LongNode, *ast.BytesNode, *ast.BooleanNode, *ast.StringNode:
		return 1, false, nil

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
			sum, err := common.AddInt(ce, le)
			if err != nil {
				return 0, false, err
			}
			res, err := common.AddInt(sum, 1)
			if err != nil {
				return 0, false, err
			}
			return res, ci || li, nil
		}
		sum, err := common.AddInt(ce, re)
		if err != nil {
			return 0, false, err
		}
		res, err := common.AddInt(sum, 1)
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
			inv = inv || li
			e.scope.restore(tmp)
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
		return 1, false, nil

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
		e.scope.setFunction(id, fc, bodyUsages, bi)
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
		res, err := common.AddInt(fd.cost, ac)
		if err != nil {
			return 0, false, err
		}
		return res, inv, nil

	case *ast.PropertyNode:
		c, inv, err := e.walk(n.Object)
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to estimate getter '%s'", n.Name)
		}
		res, err := common.AddInt(c, 1)
		if err != nil {
			return 0, false, err
		}
		return res, inv, nil

	default:
		return 0, false, errors.Errorf("unsupported type of node '%T'", node)
	}
}
