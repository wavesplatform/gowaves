package ride

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type esConstant struct {
	value rideType
	c     rideConstructor
}

type esValue struct {
	id         string
	value      rideType
	expression ast.Node
}

type esFunction struct {
	fn *ast.FunctionDeclarationNode
	sp int
}

type evaluationScope struct {
	env       environment
	constants map[string]esConstant
	cs        [][]esValue
	system    func(string) (rideFunction, bool)
	user      []esFunction
	cl        int
	costs     map[string]int
}

func (s *evaluationScope) declare(n ast.Node) error {
	switch d := n.(type) {
	case *ast.FunctionDeclarationNode:
		s.pushUserFunction(d)
		return nil
	case *ast.AssignmentNode:
		s.pushExpression(d.Name, d.Expression)
		return nil
	default:
		return EvaluationFailure.Errorf("not a declaration '%T'", n)
	}
}

func (s *evaluationScope) pushExpression(id string, n ast.Node) {
	s.cs[len(s.cs)-1] = append(s.cs[len(s.cs)-1], esValue{id: id, expression: n})
}

func (s *evaluationScope) pushValue(id string, v rideType) {
	s.cs[len(s.cs)-1] = append(s.cs[len(s.cs)-1], esValue{id: id, value: v})
}

func (s *evaluationScope) updateValue(frame, pos int, id string, v rideType) {
	if ev := s.cs[frame][pos]; ev.id == id && ev.value == nil {
		s.cs[frame][pos] = esValue{id: id, value: v}
	}
}

func (s *evaluationScope) popValue() {
	s.cs[len(s.cs)-1] = s.cs[len(s.cs)-1][:len(s.cs[len(s.cs)-1])-1]
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

func lookup(s []esValue, id string) (esValue, bool, int) {
	for i := len(s) - 1; i >= 0; i-- {
		if v := s[i]; v.id == id {
			return v, true, i
		}
	}
	return esValue{}, false, 0
}

func (s *evaluationScope) value(id string) (esValue, bool, int, int) {
	if i := len(s.cs) - 1; i >= 0 {
		v, ok, p := lookup(s.cs[i], id)
		if ok {
			return v, true, i, p
		}
	}
	for i := s.cl - 1; i >= 0; i-- {
		v, ok, p := lookup(s.cs[i], id)
		if ok {
			return v, true, i, p
		}
	}
	return esValue{}, false, 0, 0
}

func (s *evaluationScope) pushUserFunction(uf *ast.FunctionDeclarationNode) {
	s.user = append(s.user, esFunction{fn: uf, sp: len(s.cs)})
}

func (s *evaluationScope) popUserFunction() error {
	l := len(s.user)
	if l == 0 {
		return EvaluationFailure.New("empty user functions scope")
	}
	s.user = s.user[:l-1]
	return nil
}

func (s *evaluationScope) userFunction(id string) (*ast.FunctionDeclarationNode, int, bool) {
	for i := len(s.user) - 1; i >= 0; i-- {
		uf := s.user[i]
		if uf.fn.Name == id {
			return uf.fn, uf.sp, true
		}
	}
	return nil, 0, false
}

func newEvaluationScope(v ast.LibraryVersion, env environment, enableInvocation bool) (evaluationScope, error) {
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
			return evaluationScope{}, EvaluationFailure.Errorf("unknown constant '%s'", n)
		}
		cs[n] = esConstant{c: constantProvider(int(id))}
	}
	fs, err := selectFunctionsByName(v, enableInvocation)
	if err != nil {
		return evaluationScope{}, err
	}
	ev := 1
	if env.rideV6Activated() {
		ev = 2
	}
	costs, err := selectEvaluationCostsProvider(v, ev)
	if err != nil {
		return evaluationScope{}, err
	}
	return evaluationScope{
		constants: cs,
		system:    fs,
		cs:        [][]esValue{make([]esValue, 0)},
		env:       env,
		costs:     costs,
	}, nil
}

func selectConstantNames(v ast.LibraryVersion) ([]string, error) {
	switch v {
	case ast.LibV1:
		return ConstantsV1, nil
	case ast.LibV2:
		return ConstantsV2, nil
	case ast.LibV3:
		return ConstantsV3, nil
	case ast.LibV4:
		return ConstantsV4, nil
	case ast.LibV5:
		return ConstantsV5, nil
	case ast.LibV6:
		return ConstantsV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version %d", v)
	}
}

type treeEvaluator struct {
	dapp bool
	f    ast.Node
	s    evaluationScope
	env  environment
}

func (e *treeEvaluator) complexity() int {
	return e.env.complexityCalculator().complexity()
}

func (e *treeEvaluator) evaluate() (Result, error) {
	r, err := e.walk(e.f)
	if err != nil {
		return nil, err // Evaluation failed somehow, then result just an error
	}

	switch res := r.(type) {
	case rideBoolean:
		return ScriptResult{res: bool(res), complexity: e.complexity()}, nil
	case rideScriptResult, rideWriteSet, rideTransferSet:
		a, err := objectToActions(e.env, res)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "failed to convert evaluation result")
		}
		return DAppResult{actions: a, complexity: e.complexity()}, nil
	case rideList:
		var actions []proto.ScriptAction
		for _, item := range res {
			a, err := convertToAction(e.env, item)
			if err != nil {
				return nil, EvaluationFailure.Wrap(err, "failed to convert evaluation result")
			}
			actions = append(actions, a)
		}
		return DAppResult{actions: actions, complexity: e.complexity()}, nil
	case tuple2:
		var actions []proto.ScriptAction
		switch resAct := res.el1.(type) {
		case rideList:
			for _, item := range resAct {
				a, err := convertToAction(e.env, item)
				if err != nil {
					return nil, EvaluationFailure.Wrap(err, "failed to convert evaluation result")
				}
				actions = append(actions, a)
			}
		default:
			return nil, EvaluationFailure.Errorf("unexpected result type '%T'", r)
		}
		return DAppResult{actions: actions, param: res.el2, complexity: e.complexity()}, nil
	default:
		return nil, EvaluationFailure.Errorf("unexpected result type '%T'", r)
	}
}

func (e *treeEvaluator) materializeArguments(arguments []ast.Node) ([]rideType, error) {
	args := make([]rideType, len(arguments))
	for i, arg := range arguments {
		a, err := e.walk(arg)
		if err != nil {
			return nil, EvaluationErrorPush(err, "failed to materialize argument %d", i+1)
		}
		args[i] = a
	}
	return args, nil
}

func (e *treeEvaluator) evaluateNativeFunction(name string, arguments []ast.Node) (rideType, error) {
	f, ok := e.s.system(name)
	if !ok {
		return nil, EvaluationFailure.Errorf("failed to find system function '%s'", name)
	}
	cost, ok := e.s.costs[name]
	if !ok {
		return nil, EvaluationFailure.Errorf("failed to get cost of system function '%s'", name)
	}
	args, err := e.materializeArguments(arguments)
	if err != nil {
		return nil, EvaluationErrorPush(err, "failed to call system function '%s'", name)
	}
	if ok, nc := e.env.complexityCalculator().testNativeFunctionComplexity(cost); !ok {
		return nil, ComplexityLimitExceed.Errorf("evaluation complexity %d exceeds the limit %d",
			nc, e.env.complexityCalculator().limit())
	}
	defer func() {
		e.env.complexityCalculator().addNativeFunctionComplexity(cost)
	}()
	r, err := f(e.env, args...)
	if err != nil {
		return nil, EvaluationErrorPush(err, "failed to call system function '%s'", name)
	}
	return r, nil
}

func (e *treeEvaluator) evaluateUserFunction(name string, args []rideType) (rideType, error) {
	initialComplexity := e.env.complexityCalculator().complexity()
	defer func() {
		e.env.complexityCalculator().addAdditionalUserFunctionComplexity(initialComplexity)
	}()
	uf, cl, found := e.s.userFunction(name)
	if !found {
		return nil, RuntimeError.Errorf("user function '%s' not found", name)
	}
	if len(args) != len(uf.Arguments) {
		return nil, RuntimeError.Errorf("mismatched arguments number of user function '%s'", name)
	}
	avs := make([]esValue, len(args))
	for i, arg := range args {
		an := uf.Arguments[i]
		avs[i] = esValue{id: an, value: arg}
	}
	e.s.cs = append(e.s.cs, avs)
	var tmp int
	tmp, e.s.cl = e.s.cl, cl

	r, err := e.walk(uf.Body)
	if err != nil {
		return nil, EvaluationErrorPush(err, "failed to evaluate function '%s' body", name)
	}
	e.s.cs = e.s.cs[:len(e.s.cs)-1]
	e.s.cl = tmp

	if ok, nc := e.env.complexityCalculator().testAdditionalUserFunctionComplexity(initialComplexity); !ok {
		return nil, ComplexityLimitExceed.Errorf("evaluation complexity %d exceeds the limit %d",
			nc, e.env.complexityCalculator().limit())
	}
	return r, nil
}

func (e *treeEvaluator) walk(node ast.Node) (rideType, error) {
	if e.env.complexityCalculator().overflow() {
		return nil, RuntimeError.New("evaluation complexity overflow")
	}
	switch n := node.(type) {
	case *ast.LongNode:
		return rideInt(n.Value), nil

	case *ast.BytesNode:
		return rideBytes(n.Value), nil

	case *ast.BooleanNode:
		return rideBoolean(n.Value), nil

	case *ast.StringNode:
		return rideString(n.Value), nil

	case *ast.ConditionalNode:
		if ok, nc := e.env.complexityCalculator().testConditionalComplexity(); !ok {
			return nil, ComplexityLimitExceed.Errorf("evaluation complexity %d exceeds the limit %d",
				nc, e.env.complexityCalculator().limit())
		}
		defer func() {
			e.env.complexityCalculator().addConditionalComplexity()
		}()
		ce, err := e.walk(n.Condition)
		if err != nil {
			return nil, EvaluationErrorPush(err, "failed to estimate the condition of if")
		}
		cr, ok := ce.(rideBoolean)
		if !ok {
			return nil, RuntimeError.New("conditional is not a boolean")
		}
		if cr {
			return e.walk(n.TrueExpression)
		} else {
			return e.walk(n.FalseExpression)
		}

	case *ast.AssignmentNode:
		id := n.Name
		e.s.pushExpression(id, n.Expression)
		r, err := e.walk(n.Block)
		if err != nil {
			return nil, EvaluationErrorPush(err, "failed to evaluate block after declaration of variable '%s'", id)
		}
		e.s.popValue()
		return r, nil

	case *ast.ReferenceNode:
		if ok, nc := e.env.complexityCalculator().testReferenceComplexity(); !ok {
			return nil, ComplexityLimitExceed.Errorf("evaluation complexity %d exceeds the limit %d",
				nc, e.env.complexityCalculator().limit())
		}
		defer func() {
			e.env.complexityCalculator().addReferenceComplexity()
		}()
		id := n.Name
		v, ok, f, p := e.s.value(id)
		if !ok {
			if v, ok := e.s.constant(id); ok {
				return v, nil
			}
			return nil, RuntimeError.Errorf("value '%s' not found", id)
		}
		if v.value == nil {
			if v.expression == nil {
				return nil, RuntimeError.Errorf("scope value '%s' is empty", id)
			}
			r, err := e.walk(v.expression)
			if err != nil {
				return nil, EvaluationErrorPush(err, "failed to evaluate expression of scope value '%s'", id)
			}
			e.s.updateValue(f, p, id, r)
			return r, nil
		}
		return v.value, nil

	case *ast.FunctionDeclarationNode:
		id := n.Name
		e.s.pushUserFunction(n)
		r, err := e.walk(n.Block)
		if err != nil {
			return nil, EvaluationErrorPush(err, "failed to evaluate block after declaration of function '%s'", id)
		}
		err = e.s.popUserFunction()
		if err != nil {
			return nil, EvaluationErrorPush(err, "failed to evaluate declaration of function '%s'", id)
		}
		return r, nil

	case *ast.FunctionCallNode:
		name := n.Function.Name()
		switch n.Function.(type) {
		case ast.NativeFunction:
			return e.evaluateNativeFunction(name, n.Arguments)
		case ast.UserFunction:
			_, _, found := e.s.userFunction(name)
			if !found {
				return e.evaluateNativeFunction(name, n.Arguments)
			}
			args, err := e.materializeArguments(n.Arguments)
			if err != nil {
				return nil, EvaluationErrorPush(err, "failed to evaluate function '%s' body", name)
			}
			return e.evaluateUserFunction(name, args)
		default:
			return nil, RuntimeError.Errorf("unknown function type: %s", n.Function.Type())
		}

	case *ast.PropertyNode:
		if ok, nc := e.env.complexityCalculator().testPropertyComplexity(); !ok {
			return nil, ComplexityLimitExceed.Errorf("evaluation complexity %d exceeds the limit %d",
				nc, e.env.complexityCalculator().limit())
		}
		defer func() {
			e.env.complexityCalculator().addPropertyComplexity()
		}()
		name := n.Name
		obj, err := e.walk(n.Object)
		if err != nil {
			return nil, EvaluationErrorPush(err, "failed to evaluate an object to get property '%s' on it", name)
		}
		v, err := obj.get(name)

		if err != nil {
			return nil, EvaluationErrorPush(err, "failed to get property '%s'", name)
		}
		return v, nil

	default:
		return nil, EvaluationFailure.Errorf("unsupported type of node '%T'", node)
	}
}

func treeVerifierEvaluator(env environment, tree *ast.Tree) (*treeEvaluator, error) {
	s, err := newEvaluationScope(tree.LibVersion, env, false) // Invocation is disabled for expression calls
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to create scope")
	}
	if tree.IsDApp() {
		if tree.HasVerifier() {
			verifier, ok := tree.Verifier.(*ast.FunctionDeclarationNode)
			if !ok {
				return nil, EvaluationFailure.New("invalid verifier declaration")
			}
			for _, declaration := range tree.Declarations {
				err = s.declare(declaration)
				if err != nil {
					return nil, EvaluationFailure.Wrap(err, "invalid declaration")
				}
			}
			s.constants[verifier.InvocationParameter] = esConstant{c: newTx}
			return &treeEvaluator{
				dapp: tree.IsDApp(),
				f:    verifier.Body, // In DApp verifier is a function, so we have to pass its body
				s:    s,
				env:  env,
			}, nil
		}
		return nil, EvaluationFailure.New("no verifier declaration")
	}
	return &treeEvaluator{
		dapp: tree.IsDApp(),
		f:    tree.Verifier, // In simple script verifier is an expression itself
		s:    s,
		env:  env,
	}, nil
}

func treeFunctionEvaluator(env environment, tree *ast.Tree, name string, args []rideType) (*treeEvaluator, error) {
	s, err := newEvaluationScope(tree.LibVersion, env, true)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to create scope")
	}
	for _, declaration := range tree.Declarations {
		err = s.declare(declaration)
		if err != nil {
			return nil, EvaluationFailure.Wrap(err, "invalid declaration")
		}
	}
	if !tree.IsDApp() {
		return nil, EvaluationFailure.Errorf("unable to call function '%s' on simple script", name)
	}
	for i := 0; i < len(tree.Functions); i++ {
		function, ok := tree.Functions[i].(*ast.FunctionDeclarationNode)
		if !ok {
			return nil, EvaluationFailure.New("invalid callable declaration")
		}
		if function.Name == name {
			s.constants[function.InvocationParameter] = esConstant{c: newInvocation}
			if l := len(args); l != len(function.Arguments) {
				return nil, EvaluationFailure.Errorf("invalid arguments count %d for function '%s'", l, name)
			}
			for i, arg := range args {
				s.pushValue(function.Arguments[i], arg)
			}
			return &treeEvaluator{dapp: true, f: function.Body, s: s, env: env}, nil
		}
	}
	return nil, EvaluationFailure.Errorf("function '%s' not found", name)
}
