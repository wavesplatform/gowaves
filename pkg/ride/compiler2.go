package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

func ccc(f Fsm, node Node) (Fsm, error) {
	switch n := node.(type) {
	case *AssignmentNode:
		fsm, err := ccc(f.Assigment(n.Name), n.Expression)
		if err != nil {
			return fsm, err
		}
		return ccc(fsm.Return(), n.Block)
	case *LongNode:
		return f.Long(n.Value), nil
	case *FunctionCallNode:
		var err error
		f = f.Call(n.Name, uint16(len(n.Arguments)))
		for i := range n.Arguments {
			f, err = ccc(f, n.Arguments[i])
			if err != nil {
				return f, err
			}
		}
		return f.Return(), nil
	case *ReferenceNode:
		return f.Reference(n.Name), nil
	case *BooleanNode:
		return f.Boolean(n.Value), nil
	case *StringNode:
		return f.String(n.Value), nil
	case *BytesNode:
		return f.Bytes(n.Value), nil
	case *ConditionalNode:
		f, err := ccc(f.Condition(), n.Condition)
		if err != nil {
			return f, err
		}
		f, err = ccc(f.TrueBranch(), n.TrueExpression)
		if err != nil {
			return f, err
		}
		f, err = ccc(f.FalseBranch(), n.FalseExpression)
		if err != nil {
			return f, err
		}
		return f.Return(), nil
	case *FunctionDeclarationNode:
		fsm, err := ccc(f.Func(n.Name, n.Arguments, n.invocationParameter), n.Body)
		if err != nil {
			return fsm, err
		}
		return ccc(fsm.Return(), n.Block)
	case *PropertyNode:
		f, err := ccc(f.Property(n.Name), n.Object)
		if err != nil {
			return f, err
		}
		return f.Return(), nil
	case nil:
		// it should be dapp
		return f, nil
	default:
		return f, errors.Errorf("unknown type %T", node)
	}
}

func CompileVerifier(txID string, tree *Tree) (*Executable, error) {
	//s, err := newEvaluationScope(tree.LibVersion, env)
	//if err != nil {
	//	return nil, errors.Wrap(err, "failed to create scope")
	//}
	if tree.IsDApp() {
		if tree.HasVerifier() {
			_, ok := tree.Verifier.(*FunctionDeclarationNode)
			if !ok {
				return nil, errors.New("invalid verifier declaration")
			}
			//for _, declaration := range tree.Declarations {
			//	err = s.declare(declaration)
			//	if err != nil {
			//		return nil, errors.Wrap(err, "invalid declaration")
			//	}
			//}
			return compileFunction(txID, tree.LibVersion, append(tree.Declarations, tree.Verifier))
			//s.constants[verifier.invocationParameter] = esConstant{c: newTx}
			//return &treeEvaluator{
			//	dapp: tree.IsDApp(),
			//	f:    verifier.Body, // In DApp verifier is a function, so we have to pass its body
			//	s:    s,
			//	env:  env,
			//}, nil
		}
		return nil, errors.New("no verifier declaration")
	}
	return compileFunction(txID, tree.LibVersion, []Node{tree.Verifier})
}

type namedArgument struct {
	name string
	arg  rideType
}

type functionArgumentsCount = int

func CompileFunction(txID string, tree *Tree, name string, args proto.Arguments) (*Executable, functionArgumentsCount, error) {
	//s, err := newEvaluationScope(tree.LibVersion, env)
	//if err != nil {
	//	return nil, errors.Wrap(err, "failed to create scope")
	//}
	//for i, declaration := range tree.Declarations {
	//	//err = s.declare(declaration)
	//	//if err != nil {
	//	//	return nil, errors.Wrap(err, "invalid declaration")
	//	//}
	//	zap.S().Errorf("decl #%d ?? %s", i, Decompiler(declaration))
	//}
	if !tree.IsDApp() {
		return nil, 0, errors.Errorf("unable to call function '%s' on simple script", name)
	}
	for i := 0; i < len(tree.Functions); i++ {
		function, ok := tree.Functions[i].(*FunctionDeclarationNode)
		if !ok {
			return nil, 0, errors.New("invalid callable declaration")
		}
		if function.Name == name {
			//s.constants[function.invocationParameter] = esConstant{c: newInvocation}
			//if l := len(args); l != len(function.Arguments) {
			//	return nil, errors.Errorf("invalid arguments count %d for function '%s'", l, name)
			//}
			//applyArgs := make([]rideType, 0, len(args))
			//for _, arg := range args {
			//	a, err := convertArgument(arg)
			//	if err != nil {
			//		return nil, errors.Wrapf(err, "failed to call function '%s'", name)
			//	}
			//	//s.pushValue(function.Arguments[i], a)
			//	applyArgs = append(applyArgs, a)
			//	//namedArgument{
			//	//	name: function.Arguments[i],
			//	//	arg:  a,
			//	//})
			//}

			rs, err := compileFunction(txID, tree.LibVersion, append(tree.Declarations, function))
			if err != nil {
				return rs, 0, err
			}
			return rs, len(function.Arguments), nil
			//return &treeEvaluator{dapp: true, f: function.Body, s: s, env: env}, nil
		}
	}
	return nil, 0, errors.Errorf("function '%s' not found", name)

	//return compileFunction(t.LibVersion, t)
}

/*
func compileVerifier(libVersion int, node Node) (*Executable, error) {
	fCheck, err := selectFunctionChecker(libVersion)
	if err != nil {
		return nil, err
	}
	u := &uniqid{}
	b := newBuilder()
	r := newReferences(nil)
	c := newCell()
	b.writeByte(OpReturn)

	params := params{
		b: b,
		r: r,
		f: fCheck,
		u: u,
		c: c,
	}
	for k, v := range predefinedFunctions {
		params.addPredefined(k, v.id, v.f)
	}

	f := NewMain(params)
	f, err = ccc(f, node)
	if err != nil {
		return nil, err
	}
	// Just to write `OpReturn` to bytecode.
	f = f.Return()

	return f.(BuildExecutable).BuildExecutable(libVersion), nil
}
*/

func compileFunction(txID string, libVersion int, nodes []Node) (*Executable, error) {
	fCheck, err := selectFunctionChecker(libVersion)
	if err != nil {
		return nil, err
	}
	u := &uniqid{}
	b := newBuilder()
	r := newReferences(nil)
	c := newCell()
	b.writeByte(OpReturn)

	params := params{
		b:    b,
		r:    r,
		f:    fCheck,
		u:    u,
		c:    c,
		txID: txID,
	}
	for k, v := range predefinedFunctions {
		params.addPredefined(k, v.id, v.f)
	}

	//invokParam := nodes[len(nodes)-1].(*FunctionDeclarationNode).invocationParameter
	//if invokParam != "" {
	//	params.r.set(
	//		invokParam,
	//		math.MaxUint16,
	//	)
	//}

	//for _, arg := range args {
	//	params.r.set(
	//		arg.name,
	//		params.constant(arg.arg),
	//	)
	////}
	//args2 := make([]rideType, len(args))
	//for i := range args {
	//	args2[i] = args[i].arg
	//}

	f := NewMain(params)
	for _, node := range nodes {
		zap.S().Error(Decompiler(node))
		f, err = ccc(f, node)
		if err != nil {
			return nil, err
		}
	}
	// Just to write `OpReturn` to bytecode.
	f = f.Return()

	return f.(BuildExecutable).BuildExecutable(libVersion), nil
}
