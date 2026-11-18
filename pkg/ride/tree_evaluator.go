package ride

import (
	"slices"
	"sync"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

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
	env     environment
	consts  *evaluationScopeConstants
	values  []rideType // Lazily created values of the library constants, indexed as consts.constructors.
	inv     esInvocationParameter
	cs      [][]esValue
	natives map[string]esNative
	user    []esFunction
	cl      int
}

// esNative is a system function of a library version together with its evaluation cost.
type esNative struct {
	f       rideFunction
	cost    int
	hasF    bool
	hasCost bool
}

// esInvocationParameter is the invocation parameter (tx or invocation) of a DApp verifier or callable.
// It overlays the library constants with the same name.
type esInvocationParameter struct {
	set   bool
	name  string
	value rideType
	c     rideConstructor
}

// evaluationScopeConstants is the read-only constants data of a library version shared by all scopes.
type evaluationScopeConstants struct {
	ids          map[string]int
	constructors []rideConstructor
}

// evaluationScopeTable is the read-only per-version data of an evaluation scope.
type evaluationScopeTable struct {
	constants *evaluationScopeConstants
	natives   map[string]esNative
	err       error
}

func evaluationScopeTableIndex(v ast.LibraryVersion, enableInvocation bool, ev evaluatorVersion) int {
	i := int(v-ast.LibV1) * 4 //nolint:mnd // Two invocation variants by two evaluator versions.
	if enableInvocation {
		i += 2
	}
	if ev == evaluatorV2 {
		i++
	}
	return i
}

// evaluationScopeTables returns the scope tables of all supported library versions, built once.
var evaluationScopeTables = sync.OnceValue(func() []evaluationScopeTable { //nolint:gochecknoglobals // Read-only cache.
	maxV := ast.CurrentMaxLibraryVersion()
	tables := make([]evaluationScopeTable, evaluationScopeTableIndex(maxV, true, evaluatorV2)+1)
	for v := ast.LibV1; v <= maxV; v++ {
		constants, err := buildEvaluationScopeConstants(v)
		for _, inv := range []bool{false, true} {
			for _, ev := range []evaluatorVersion{evaluatorV1, evaluatorV2} {
				t := &tables[evaluationScopeTableIndex(v, inv, ev)]
				if err != nil {
					t.err = err
					continue
				}
				*t = buildEvaluationScopeTable(v, inv, ev)
				t.constants = constants
			}
		}
	}
	return tables
})

func buildEvaluationScopeConstants(v ast.LibraryVersion) (*evaluationScopeConstants, error) {
	constants, err := selectConstantNames(v)
	if err != nil {
		return nil, err
	}
	constantChecker, err := selectConstantsChecker(v)
	if err != nil {
		return nil, err
	}
	constantProvider, err := selectConstants(v)
	if err != nil {
		return nil, err
	}
	r := &evaluationScopeConstants{
		ids:          make(map[string]int, len(constants)),
		constructors: make([]rideConstructor, 0, len(constants)),
	}
	for _, n := range constants {
		id, ok := constantChecker(n)
		if !ok {
			return nil, EvaluationFailure.Errorf("unknown constant '%s'", n)
		}
		r.ids[n] = len(r.constructors)
		r.constructors = append(r.constructors, constantProvider(int(id)))
	}
	return r, nil
}

func buildEvaluationScopeTable(v ast.LibraryVersion, enableInvocation bool, ev evaluatorVersion) evaluationScopeTable {
	fs, err := selectFunctionsByName(v, enableInvocation)
	if err != nil {
		return evaluationScopeTable{err: err}
	}
	costs, err := selectEvaluationCostsProvider(v, ev)
	if err != nil {
		return evaluationScopeTable{err: err}
	}
	names, err := selectFunctionNameProvider(v)
	if err != nil {
		return evaluationScopeTable{err: err}
	}
	natives := make(map[string]esNative, len(costs))
	add := func(name string) {
		if _, ok := natives[name]; ok {
			return
		}
		f, hasF := fs(name)
		cost, hasCost := costs[name]
		if hasF || hasCost {
			natives[name] = esNative{f: f, cost: cost, hasF: hasF, hasCost: hasCost}
		}
	}
	for i := 0; ; i++ {
		name := names(i)
		if name == "" {
			break
		}
		add(name)
	}
	for name := range costs {
		add(name)
	}
	return evaluationScopeTable{natives: natives}
}

func lookupEvaluationScopeTable(v ast.LibraryVersion, enableInvocation bool, ev evaluatorVersion) evaluationScopeTable {
	if v < ast.LibV1 || v > ast.CurrentMaxLibraryVersion() {
		return evaluationScopeTable{err: unsupportedLibraryVersionEvaluationError(v)}
	}
	return evaluationScopeTables()[evaluationScopeTableIndex(v, enableInvocation, ev)]
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

func (s *evaluationScope) setInvocationParameter(name string, c rideConstructor) {
	s.inv = esInvocationParameter{set: true, name: name, c: c}
}

func (s *evaluationScope) constant(id string) (rideType, bool) {
	if s.inv.set && s.inv.name == id {
		if s.inv.value == nil {
			s.inv.value = s.inv.c(s.env)
		}
		return s.inv.value, true
	}
	i, ok := s.consts.ids[id]
	if !ok {
		return nil, false
	}
	if s.values == nil {
		s.values = make([]rideType, len(s.consts.constructors))
	}
	v := s.values[i]
	if v == nil {
		v = s.consts.constructors[i](s.env)
		s.values[i] = v
	}
	return v, true
}

func lookup(s []esValue, id string) (esValue, bool, int) {
	for i, v := range slices.Backward(s) {
		if v.id == id {
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

// pushFrame pushes a new call frame, reusing the backing array of a frame popped before at the same depth.
func (s *evaluationScope) pushFrame(size int) []esValue {
	l := len(s.cs)
	var frame []esValue
	if l < cap(s.cs) {
		frame = s.cs[:l+1][l][:0]
	}
	if cap(frame) < size {
		frame = make([]esValue, 0, size)
	}
	s.cs = append(s.cs, frame)
	return frame
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
	for _, uf := range slices.Backward(s.user) {
		if uf.fn.Name == id {
			return uf.fn, uf.sp, true
		}
	}
	return nil, 0, false
}

// newEvaluationScope creates a scope with room reserved for the given declarations and the given number of
// additional values in the root frame.
func newEvaluationScope(
	v ast.LibraryVersion, env environment, enableInvocation bool, declarations []ast.Node, values int,
) (evaluationScope, error) {
	ev := evaluatorV1
	if env.rideV6Activated() {
		ev = evaluatorV2
	}
	t := lookupEvaluationScopeTable(v, enableInvocation, ev)
	if t.err != nil {
		return evaluationScope{}, t.err
	}
	functions := 0
	for _, d := range declarations {
		switch d.(type) {
		case *ast.FunctionDeclarationNode:
			functions++
		case *ast.AssignmentNode:
			values++
		}
	}
	// Room for the frames of top-level user function calls (recursion is impossible), nested declarations may grow it.
	cs := make([][]esValue, 1, 1+functions)
	if values > 0 {
		cs[0] = make([]esValue, 0, values)
	}
	var user []esFunction
	if functions > 0 {
		user = make([]esFunction, 0, functions)
	}
	return evaluationScope{
		consts:  t.constants,
		natives: t.natives,
		cs:      cs,
		user:    user,
		env:     env,
	}, nil
}

type treeEvaluator struct {
	dapp  bool
	f     ast.Node
	fName string
	s     evaluationScope
	env   environment
	cc    complexityCalculator // Cached env.complexityCalculator(), refreshed after each system function call.
	args  []rideType           // Stack of materialized arguments of the system function calls in progress.
}

func (e *treeEvaluator) complexity() int {
	return e.env.complexityCalculator().complexity()
}

func errTypeFromComplexityCalcErr(err error) EvaluationError {
	// Fast path: complexity calculators return their errors unwrapped.
	if ccErr, ok := err.(complexityCalculatorError); ok { //nolint:errorlint // errors.As is the fallback below
		return ccErr.EvaluationErrorWrapType()
	}
	eet := Undefined
	if ccErr := complexityCalculatorError(nil); errors.As(err, &ccErr) {
		eet = ccErr.EvaluationErrorWrapType()
	}
	return eet
}

func (e *treeEvaluator) handleZeroComplexityCall(
	initialComplexity int,
	res Result, // if runErr != nil, then res is nil, otherwise res must not be nil
	runErr error,
) (Result, error) {
	if runErr != nil {
		return res, runErr // nothing to do in case of error
	}
	if !e.dapp {
		return res, nil // nothing to do in case of expression script call
	}
	// check any error from complexity calculator
	if err := e.env.complexityCalculator().error(); err != nil {
		eet := errTypeFromComplexityCalcErr(err)
		return res, eet.Wrapf(err,
			"complexity calculator error detected at the end of evaluation in '%s'", e.fName,
		)
	}
	cErr := e.env.complexityCalculator().testAdditionalUserFunctionComplexity(e.fName, initialComplexity)
	if cErr != nil {
		eet := errTypeFromComplexityCalcErr(cErr)
		return res, eet.Wrapf(cErr, "failed to add complexity of user script invocation '%s'", e.fName)
	}
	// can safely call addAdditionalUserFunctionComplexity because we tested it above
	e.env.complexityCalculator().addAdditionalUserFunctionComplexity(e.fName, initialComplexity)
	return res.withComplexity(e.complexity()), nil
}

func (e *treeEvaluator) evaluate() (res Result, runErr error) { //nolint:nonamedreturns // necessary to handle results
	defer func(initialComplexity int) {
		res, runErr = e.handleZeroComplexityCall(initialComplexity, res, runErr)
	}(e.complexity())

	e.cc = e.env.complexityCalculator()
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
		var actions []proto.ScriptAction // Stays nil for an empty list.
		if len(res) > 0 {
			actions = make([]proto.ScriptAction, 0, len(res))
		}
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
			if len(resAct) > 0 {
				actions = make([]proto.ScriptAction, 0, len(resAct))
			}
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

// materializeArguments evaluates the arguments and pushes their values on top of the arguments stack.
// It returns the position of the first argument. The caller must pop the arguments with popArguments.
func (e *treeEvaluator) materializeArguments(arguments []ast.Node) (int, error) {
	base := len(e.args)
	for i, arg := range arguments {
		a, err := e.walk(arg)
		if err != nil {
			e.popArguments(base)
			return 0, EvaluationErrorPushf(err, "failed to materialize argument %d", i+1)
		}
		e.args = append(e.args, a)
	}
	return base, nil
}

func (e *treeEvaluator) popArguments(base int) {
	clear(e.args[base:])
	e.args = e.args[:base]
}

func (e *treeEvaluator) evaluateNativeFunction(name string, arguments []ast.Node) (rideType, error) {
	nf, ok := e.s.natives[name]
	if !ok || !nf.hasF {
		return nil, EvaluationFailure.Errorf("failed to find system function '%s'", name)
	}
	if !nf.hasCost {
		return nil, EvaluationFailure.Errorf("failed to get cost of system function '%s'", name)
	}
	base, err := e.materializeArguments(arguments)
	if err != nil {
		return nil, EvaluationErrorPushf(err, "failed to call system function '%s'", name)
	}
	if tErr := e.cc.testNativeFunctionComplexity(name, nf.cost); tErr != nil {
		e.popArguments(base)
		eet := errTypeFromComplexityCalcErr(tErr)
		return nil, eet.Wrap(tErr, "failed to test complexity of system function")
	}
	// System functions don't keep the arguments slice, so it is safe to pass a part of the arguments stack.
	// System functions never call back into this evaluator, so the stack doesn't change during the call.
	r, err := nf.f(e.env, e.args[base:]...)
	e.popArguments(base)
	e.cc = e.env.complexityCalculator() // Invoke functions may replace the complexity calculator.
	e.cc.addNativeFunctionComplexity(name, nf.cost)
	if err != nil {
		return nil, EvaluationErrorPushf(err, "failed to call system function '%s'", name)
	}
	return r, nil
}

func (e *treeEvaluator) evaluateUserFunction(
	name string, uf *ast.FunctionDeclarationNode, cl int, arguments []ast.Node,
) (rideType, error) {
	base, err := e.materializeArguments(arguments)
	if err != nil {
		return nil, EvaluationErrorPushf(err, "failed to evaluate function '%s' body", name)
	}
	initialComplexity := e.cc.complexity()
	if len(arguments) != len(uf.Arguments) {
		e.popArguments(base)
		e.cc.addAdditionalUserFunctionComplexity(name, initialComplexity)
		return nil, RuntimeError.Errorf("mismatched arguments number of user function '%s'", name)
	}
	avs := e.s.pushFrame(len(arguments))
	for i, arg := range e.args[base:] {
		avs = append(avs, esValue{id: uf.Arguments[i], value: arg})
	}
	e.s.cs[len(e.s.cs)-1] = avs
	e.popArguments(base)
	var tmp int
	tmp, e.s.cl = e.s.cl, cl

	r, err := e.walk(uf.Body)
	if err != nil {
		e.cc.addAdditionalUserFunctionComplexity(name, initialComplexity)
		return nil, EvaluationErrorPushf(err, "failed to evaluate function '%s' body", name)
	}
	e.s.cs = e.s.cs[:len(e.s.cs)-1]
	e.s.cl = tmp

	tErr := e.cc.testAdditionalUserFunctionComplexity(name, initialComplexity)
	e.cc.addAdditionalUserFunctionComplexity(name, initialComplexity)
	if tErr != nil {
		eet := errTypeFromComplexityCalcErr(tErr)
		return nil, eet.Wrap(tErr, "failed to test complexity of user function")
	}
	return r, nil
}

func (e *treeEvaluator) walk(node ast.Node) (rideType, error) {
	if err := e.cc.error(); err != nil {
		eet := errTypeFromComplexityCalcErr(err)
		return nil, eet.Wrapf(err, "failed to walk node '%T'", node)
	}
	switch n := node.(type) {
	case *ast.LongNode:
		return rideInt(n.Value), nil

	case *ast.BytesNode:
		return rideByteVector(n.Value), nil

	case *ast.BooleanNode:
		return rideBoolean(n.Value), nil

	case *ast.StringNode:
		return rideString(n.Value), nil

	case *ast.ConditionalNode:
		return e.walkConditional(n)

	case *ast.AssignmentNode:
		id := n.Name
		e.s.pushExpression(id, n.Expression)
		r, err := e.walk(n.Block)
		if err != nil {
			return nil, EvaluationErrorPushf(err, "failed to evaluate block after declaration of variable '%s'", id)
		}
		e.s.popValue()
		return r, nil

	case *ast.ReferenceNode:
		if tErr := e.cc.testReferenceComplexity(); tErr != nil {
			eet := errTypeFromComplexityCalcErr(tErr)
			return nil, eet.Wrap(tErr, "failed to test reference complexity")
		}
		r, err := e.walkReference(n.Name)
		e.cc.addReferenceComplexity()
		return r, err

	case *ast.FunctionDeclarationNode:
		id := n.Name
		e.s.pushUserFunction(n)
		r, err := e.walk(n.Block)
		if err != nil {
			return nil, EvaluationErrorPushf(err, "failed to evaluate block after declaration of function '%s'", id)
		}
		err = e.s.popUserFunction()
		if err != nil {
			return nil, EvaluationErrorPushf(err, "failed to evaluate declaration of function '%s'", id)
		}
		return r, nil

	case *ast.FunctionCallNode:
		name := n.Function.Name()
		switch n.Function.(type) {
		case ast.NativeFunction:
			return e.evaluateNativeFunction(name, n.Arguments)
		case ast.UserFunction:
			uf, cl, found := e.s.userFunction(name)
			if !found {
				return e.evaluateNativeFunction(name, n.Arguments)
			}
			return e.evaluateUserFunction(name, uf, cl, n.Arguments)
		default:
			return nil, RuntimeError.Errorf("unknown function type: %s", n.Function.Type())
		}

	case *ast.PropertyNode:
		if tErr := e.cc.testPropertyComplexity(); tErr != nil {
			eet := errTypeFromComplexityCalcErr(tErr)
			return nil, eet.Wrap(tErr, "failed to test property complexity")
		}
		r, err := e.walkProperty(n)
		e.cc.addPropertyComplexity()
		return r, err

	default:
		return nil, EvaluationFailure.Errorf("unsupported type of node '%T'", node)
	}
}

// walkConditional evaluates the conditional node. The conditional complexity is added after the evaluation of
// the chosen branch on every path, the same way as the deferred call did before.
func (e *treeEvaluator) walkConditional(n *ast.ConditionalNode) (rideType, error) {
	if tErr := e.cc.testConditionalComplexity(); tErr != nil {
		eet := errTypeFromComplexityCalcErr(tErr)
		return nil, eet.Wrap(tErr, "failed to test conditional complexity")
	}
	ce, err := e.walk(n.Condition)
	if err != nil {
		e.cc.addConditionalComplexity()
		return nil, EvaluationErrorPushf(err, "failed to estimate the condition of if")
	}
	cr, ok := ce.(rideBoolean)
	if !ok {
		e.cc.addConditionalComplexity()
		return nil, RuntimeError.New("conditional is not a boolean")
	}
	var r rideType
	if cr {
		r, err = e.walk(n.TrueExpression)
	} else {
		r, err = e.walk(n.FalseExpression)
	}
	e.cc.addConditionalComplexity()
	return r, err
}

func (e *treeEvaluator) walkReference(id string) (rideType, error) {
	v, ok, f, p := e.s.value(id)
	if !ok {
		if c, found := e.s.constant(id); found {
			return c, nil
		}
		return nil, RuntimeError.Errorf("value '%s' not found", id)
	}
	if v.value == nil {
		if v.expression == nil {
			return nil, RuntimeError.Errorf("scope value '%s' is empty", id)
		}
		r, err := e.walk(v.expression)
		if err != nil {
			return nil, EvaluationErrorPushf(err, "failed to evaluate expression of scope value '%s'", id)
		}
		e.s.updateValue(f, p, id, r)
		return r, nil
	}
	return v.value, nil
}

func (e *treeEvaluator) walkProperty(n *ast.PropertyNode) (rideType, error) {
	name := n.Name
	obj, err := e.walk(n.Object)
	if err != nil {
		return nil, EvaluationErrorPushf(err, "failed to evaluate an object to get property '%s' on it", name)
	}
	v, err := obj.get(name)
	if err != nil {
		return nil, EvaluationErrorPushf(err, "failed to get property '%s'", name)
	}
	return v, nil
}

func treeVerifierEvaluator(env environment, tree *ast.Tree) (*treeEvaluator, error) {
	// Invocation is disabled for expression calls.
	s, err := newEvaluationScope(tree.LibVersion, env, false, tree.Declarations, 0)
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
			s.setInvocationParameter(verifier.InvocationParameter, newTx)
			return &treeEvaluator{
				dapp:  tree.IsDApp(),
				fName: verifier.Name,
				f:     verifier.Body, // In DApp verifier is a function, so we have to pass its body
				s:     s,
				env:   env,
			}, nil
		}
		return nil, EvaluationFailure.New("no verifier declaration")
	}
	return &treeEvaluator{
		dapp:  tree.IsDApp(),
		fName: "verifier",
		f:     tree.Verifier, // In simple script verifier is an expression itself
		s:     s,
		env:   env,
	}, nil
}

func treeFunctionEvaluator(env environment, tree *ast.Tree, name string, args []rideType) (*treeEvaluator, error) {
	s, err := newEvaluationScope(tree.LibVersion, env, true, tree.Declarations, len(args))
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
			s.setInvocationParameter(function.InvocationParameter, newInvocation)
			if l := len(args); l != len(function.Arguments) {
				return nil, EvaluationFailure.Errorf("invalid arguments count %d for function '%s'", l, name)
			}
			for i, arg := range args {
				s.pushValue(function.Arguments[i], arg)
			}
			return &treeEvaluator{dapp: true, fName: name, f: function.Body, s: s, env: env}, nil
		}
	}
	return nil, EvaluationFailure.Errorf("function '%s' not found", name)
}
