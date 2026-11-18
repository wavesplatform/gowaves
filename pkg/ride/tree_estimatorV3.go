package ride

import (
	"slices"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type functionDescriptor struct {
	cost            int
	usages          []string
	callsInvocation bool
}

// fsV3 is a scope of user function declarations. Scopes form a chain: a lookup that misses in the scope
// continues in its parent. The map of a scope is allocated lazily on the first set, because most of the scopes
// created by save never get a declaration.
type fsV3 struct {
	parent    *fsV3
	functions map[string]functionDescriptor
}

func newFsV3() *fsV3 {
	return &fsV3{}
}

func (f *fsV3) spawn() *fsV3 {
	return &fsV3{parent: f}
}

// set adds new function descriptor to context, returns true if new function overwrite old one.
func (f *fsV3) set(key string, cost int, usages []string, callsInvocation bool) bool {
	if f.functions == nil {
		f.functions = make(map[string]functionDescriptor)
	}
	_, ok := f.functions[key]
	f.functions[key] = functionDescriptor{cost: cost, usages: usages, callsInvocation: callsInvocation}
	return ok
}

func (f *fsV3) get(key string) (functionDescriptor, bool) {
	for c := f; c != nil; c = c.parent {
		if fd, ok := c.functions[key]; ok {
			return fd, true
		}
	}
	return functionDescriptor{}, false
}

// estimationScopeV3 is the scope shared by the estimators V3 and V4.
// The usages field is a stack of the names referenced in the nested function bodies. The slices above the top
// of the stack (between len and cap) are kept to reuse their backing arrays on the next submerge.
type estimationScopeV3 struct {
	usages    [][]string
	root      *fsV3
	functions *fsV3
	builtin   map[string]int
}

func newEstimationScopeV3(functions map[string]int) *estimationScopeV3 {
	const initialUsagesCapacity = 4
	root := newFsV3()
	return &estimationScopeV3{
		usages:    make([][]string, 0, initialUsagesCapacity),
		root:      root,
		functions: root,
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

// resetFunctions drops all function declarations. The root scope and its map are reused, because no scope
// created before the reset is used after it.
func (s *estimationScopeV3) resetFunctions() {
	clear(s.root.functions)
	s.functions = s.root
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
		u := s.usages[l-1]
		for _, v := range slices.Backward(u) {
			if v == id {
				return true
			}
		}
	}
	return false
}

// remove deletes all occurrences of id from the top usages, keeping the order of the others.
// It returns true if id was there.
func (s *estimationScopeV3) remove(id string) bool {
	l := len(s.usages)
	if l == 0 {
		return false
	}
	u := s.usages[l-1]
	i := slices.Index(u, id)
	if i < 0 {
		return false
	}
	j := i
	for k := i + 1; k < len(u); k++ {
		if u[k] != id {
			u[j] = u[k]
			j++
		}
	}
	clear(u[j:])
	s.usages[l-1] = u[:j]
	return true
}

func (s *estimationScopeV3) use(id string) {
	if l := len(s.usages); l > 0 {
		s.usages[l-1] = append(s.usages[l-1], id)
	}
}

func (s *estimationScopeV3) submerge() {
	if l := len(s.usages); l < cap(s.usages) {
		s.usages = s.usages[:l+1]
		s.usages[l] = s.usages[l][:0]
		return
	}
	s.usages = append(s.usages, nil)
}

// emerge pops the top usages and returns a copy of them. The backing array stays in the stack for reuse.
func (s *estimationScopeV3) emerge() []string {
	if l := len(s.usages); l > 0 {
		r := slices.Clone(s.usages[l-1])
		s.usages = s.usages[:l-1]
		return r
	}
	return nil
}

// drop pops the top usages without returning them.
func (s *estimationScopeV3) drop() {
	if l := len(s.usages); l > 0 {
		s.usages = s.usages[:l-1]
	}
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
	case ast.LibV7:
		r.scope = newEstimationScopeV3(CatalogueV7)
	case ast.LibV8:
		r.scope = newEstimationScopeV3(CatalogueV8)
	case ast.LibV9:
		r.scope = newEstimationScopeV3(CatalogueV9)
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
		e.scope.drop()
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
		c, _, err := e.walk(wrapCallable(e.tree.Declarations, function))
		if err != nil {
			return 0, 0, nil, err
		}
		e.scope.drop()
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
		c, inv, err := e.walk(wrapCallable(e.tree.Declarations, verifier))
		if err != nil {
			return 0, 0, nil, err
		}
		if inv {
			return 0, 0, nil, errors.New("usage of invocation functions is prohibited in verifier")
		}
		e.scope.drop()
		vc = c
		if c > max {
			max = c
		}
	}
	return max, vc, m, nil
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
		overlapped := e.scope.remove(id)
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
