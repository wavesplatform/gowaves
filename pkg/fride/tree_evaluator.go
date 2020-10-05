package fride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type esConstant struct {
	value rideType
	c     rideConstructor
}

type esValue struct {
	id         string
	value      rideType
	expression Node
}

type esFunction struct {
	fn *FunctionDeclarationNode
	fs []esValue
}

type evaluationScope struct {
	env       RideEnvironment
	constants map[string]esConstant
	values    []esValue
	sfs       map[string]rideFunction
	user      []esFunction
}

func (s *evaluationScope) declare(n Node) error {
	switch d := n.(type) {
	case *FunctionDeclarationNode:
		s.pushUserFunction(d)
		return nil
	case *AssignmentNode:
		s.pushExpression(d.Name, d.Expression)
		return nil
	default:
		return errors.Errorf("not a declaration '%T'", n)
	}
}

func (s *evaluationScope) pushExpression(id string, n Node) {
	s.values = append(s.values, esValue{id: id, expression: n})
}

func (s *evaluationScope) pushValue(id string, v rideType) {
	s.values = append(s.values, esValue{id: id, value: v})
}

func (s *evaluationScope) popValue() error {
	l := len(s.values)
	if l == 0 {
		return errors.New("empty value scope")
	}
	s.values = s.values[:l-1]
	return nil
}

func (s *evaluationScope) branch() []esValue {
	vs := make([]esValue, len(s.values))
	for i, v := range s.values {
		vs[i] = v
	}
	return vs
}

func (s *evaluationScope) constant(id string) (rideType, bool) {
	if c, ok := s.constants[id]; ok {
		if c.value == nil {
			v := c.c(s.env)
			s.constants[id] = esConstant{value: v, c: c.c}
			return v, true
		}
		return c.value, true
	}
	return nil, false
}

func (s *evaluationScope) value(id string) (esValue, error) {
	for i := len(s.values) - 1; i >= 0; i-- {
		if s.values[i].id == id {
			return s.values[i], nil
		}
	}
	return esValue{}, errors.Errorf("value '%s' not found", id)
}

func (s *evaluationScope) pushUserFunction(uf *FunctionDeclarationNode) {
	s.user = append(s.user, esFunction{
		fn: uf,
		fs: s.branch(),
	})
}

func (s *evaluationScope) popUserFunction() error {
	l := len(s.user)
	if l == 0 {
		return errors.New("empty user functions scope")
	}
	s.user = s.user[:l-1]
	return nil
}

func (s *evaluationScope) userFunction(id string) (*FunctionDeclarationNode, []esValue, error) {
	for i := len(s.user) - 1; i >= 0; i-- {
		f := s.user[i]
		if f.fn.Name == id {
			return f.fn, f.fs, nil
		}
	}
	return nil, nil, errors.Errorf("user function '%s' is not found", id)
}

func newEvaluationScope(v int, env RideEnvironment) (evaluationScope, error) {
	constants, err := selectConstantNames(v)
	if err != nil {
		return evaluationScope{}, err
	}
	constantChecker, err := selectConstantsChecker(v)
	if err != nil {
		return evaluationScope{}, err
	}
	constantProvider, err := selectConstants(v)
	if err != nil {
		return evaluationScope{}, err
	}
	cs := make(map[string]esConstant, len(constants))
	for _, n := range constants {
		id, ok := constantChecker(n)
		if !ok {
			return evaluationScope{}, errors.Errorf("unknown constant '%s'", n)
		}
		cs[n] = esConstant{c: constantProvider(int(id))}
	}
	functions, err := selectFunctionNames(v)
	if err != nil {
		return evaluationScope{}, err
	}
	functionChecker, err := selectFunctionChecker(v)
	if err != nil {
		return evaluationScope{}, err
	}
	functionProvider, err := selectFunctions(v)
	if err != nil {
		return evaluationScope{}, err
	}
	fs := make(map[string]rideFunction, len(functions))
	for _, fn := range functions {
		id, ok := functionChecker(fn)
		if !ok {
			return evaluationScope{}, errors.Errorf("unknown function '%s'", fn)
		}
		fs[fn] = functionProvider(int(id))
	}
	return evaluationScope{
		constants: cs,
		sfs:       fs,
		env:       env,
	}, nil
}

func selectConstantNames(v int) ([]string, error) {
	switch v {
	case 1:
		return ConstantsV1, nil
	case 2:
		return ConstantsV2, nil
	case 3:
		return ConstantsV3, nil
	case 4:
		return ConstantsV4, nil
	default:
		return nil, errors.Errorf("unsupported library version %d", v)
	}
}

func keys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func selectFunctionNames(v int) ([]string, error) {
	switch v {
	case 1, 2:
		return keys(CatalogueV2), nil
	case 3:
		return keys(CatalogueV3), nil
	case 4:
		return keys(CatalogueV4), nil
	default:
		return nil, errors.Errorf("unsupported library version %d", v)
	}
}

type treeEvaluator struct {
	f   Node
	s   evaluationScope
	env RideEnvironment
}

func (e *treeEvaluator) evaluate() (RideResult, error) {
	r, err := e.walk(e.f)
	if err != nil {
		return nil, err
	}
	switch res := r.(type) {
	case rideBoolean:
		return ScriptResult(res), nil
	case rideObject:
		actions, err := objectToActions(e.env, res)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert evaluation result")
		}
		return DAppResult(actions), nil
	case rideList:
		actions := make([]proto.ScriptAction, 0, len(res))
		for i, item := range res {
			a, err := convertToAction(e.env, item)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert evaluation result")
			}
			actions[i] = a
		}
		return DAppResult(actions), nil
	default:
		return nil, errors.Errorf("unexpected result type '%T'", r)
	}
}

func (e *treeEvaluator) walk(node Node) (rideType, error) {
	switch n := node.(type) {
	case *LongNode:
		return rideInt(n.Value), nil

	case *BytesNode:
		return rideBytes(n.Value), nil

	case *BooleanNode:
		return rideBoolean(n.Value), nil

	case *StringNode:
		return rideString(n.Value), nil

	case *ConditionalNode:
		ce, err := e.walk(n.Condition)
		if err != nil {
			return nil, errors.Wrap(err, "failed to estimate the condition of if")
		}
		cr, ok := ce.(rideBoolean)
		if !ok {
			return nil, errors.Errorf("not a boolean")
		}
		if cr {
			return e.walk(n.TrueExpression)
		} else {
			return e.walk(n.FalseExpression)
		}

	case *AssignmentNode:
		id := n.Name
		e.s.pushExpression(id, n.Expression)
		r, err := e.walk(n.Block)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to evaluate block after declaration of variable '%s'", id)
		}
		err = e.s.popValue()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to evaluate declaration of variable '%s'", id)
		}
		return r, nil

	case *ReferenceNode:
		id := n.Name
		if v, ok := e.s.constant(id); ok {
			return v, nil
		}
		v, err := e.s.value(id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get reference '%s'", id)
		}
		if v.value == nil {
			if v.expression == nil {
				return nil, errors.Errorf("scope value '%s' is empty", id)
			}
			r, err := e.walk(v.expression)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to evaluate expression of scope value '%s'", id)
			}
			e.s.pushValue(id, r)
			return r, nil
		}
		return v.value, nil

	case *FunctionDeclarationNode:
		id := n.Name
		e.s.pushUserFunction(n)
		r, err := e.walk(n.Block)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to evaluate block after declaration of function '%s'", id)
		}
		err = e.s.popUserFunction()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to evaluate declaration of function '%s'", id)
		}
		return r, nil

	case *FunctionCallNode:
		id := n.Name
		f, ok := e.s.sfs[id]
		if ok { // System function
			args := make([]rideType, len(n.Arguments))
			for i, arg := range n.Arguments {
				a, err := e.walk(arg) // materialize argument
				if err != nil {
					return nil, errors.Wrapf(err, "failed to materialize argument %d of system function '%s'", i+1, id)
				}
				args[i] = a
			}
			r, err := f(e.env, args...)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to call system function '%s'", id)
			}
			return r, nil
		}
		uf, fs, err := e.s.userFunction(id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to call function '%s'", id)
		}
		if len(n.Arguments) != len(uf.Arguments) {
			return nil, errors.Errorf("mismatched arguments number of user function '%s'", id)
		}
		tmp := e.s.values
		for i, arg := range n.Arguments {
			an := uf.Arguments[i]
			av, err := e.walk(arg) // materialize argument
			if err != nil {
				return nil, errors.Wrapf(err, "failed to materialize argument '%s' of user function '%s", an, id)
			}
			e.s.pushValue(an, av)
			fs = append(fs, esValue{id: an, value: av})
		}
		e.s.values = fs
		r, err := e.walk(uf.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to evaluate function '%s' body", id)
		}
		e.s.values = tmp
		return r, nil

	case *PropertyNode:
		name := n.Name
		obj, err := e.walk(n.Object)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to evaluate an object to get property '%s' on it", name)
		}
		v, err := obj.get(name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get property '%s'", name)
		}
		return v, nil

	default:
		return nil, errors.Errorf("unsupported type of node '%T'", node)
	}
}

func treeVerifierEvaluator(env RideEnvironment, tree *Tree) (*treeEvaluator, error) {
	s, err := newEvaluationScope(tree.LibVersion, env)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create scope")
	}
	if tree.IsDApp() {
		if tree.HasVerifier() {
			verifier, ok := tree.Verifier.(*FunctionDeclarationNode)
			if !ok {
				return nil, errors.New("invalid verifier declaration")
			}
			for _, declaration := range tree.Declarations {
				err = s.declare(declaration)
				if err != nil {
					return nil, errors.Wrap(err, "invalid declaration")
				}
			}
			s.constants[verifier.invocationParameter] = esConstant{c: newInvocation}
			return &treeEvaluator{
				f:   verifier.Body, // In DApp verifier is a function, so we have to pass its body
				s:   s,
				env: env,
			}, nil
		}
		return nil, errors.Wrap(err, "no verifier declaration")
	}
	return &treeEvaluator{
		f:   tree.Verifier, // In simple scripts verifier is an expression itself
		s:   s,
		env: env,
	}, nil
}

func treeFunctionEvaluator(env RideEnvironment, tree *Tree, name string, args proto.Arguments) (*treeEvaluator, error) {
	s, err := newEvaluationScope(tree.LibVersion, env)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create scope")
	}
	for _, declaration := range tree.Declarations {
		err = s.declare(declaration)
		if err != nil {
			return nil, errors.Wrap(err, "invalid declaration")
		}
	}
	if !tree.IsDApp() {
		return nil, errors.Errorf("unable to call function '%s' on simple script", name)
	}
	for i := 0; i < len(tree.Functions); i++ {
		function, ok := tree.Functions[i].(*FunctionDeclarationNode)
		if !ok {
			return nil, errors.New("invalid callable declaration")
		}
		if function.Name == name {
			s.constants[function.invocationParameter] = esConstant{c: newInvocation}
			if l := len(args); l != len(function.Arguments) {
				return nil, errors.Errorf("invalid arguments count %d for function '%s'", l, name)
			}
			for i, arg := range args {
				a, err := convertArgument(arg)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to call function '%s'", name)
				}
				s.pushValue(function.Arguments[i], a)
			}
			return &treeEvaluator{f: function.Body, s: s, env: env}, nil
		}
	}
	return nil, errors.Errorf("function '%s' not found", name)
}
