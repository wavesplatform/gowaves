package ride

import "github.com/pkg/errors"

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
		f = f.Call(n.Name, uint16(len(n.Arguments)))
		for i := range n.Arguments {
			f, err := ccc(f, n.Arguments[i])
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
		// TODO
		fsm, err := ccc(f.Assigment(n.Name), n.Body)
		if err != nil {
			return fsm, err
		}
		return ccc(fsm.Return(), n.Block)
	default:
		return f, errors.Errorf("unknown type %T", node)
	}
}

func CompileSimpleScript(t *Tree) (*Executable, error) {
	return compileSimpleScript(t.LibVersion, t.Verifier)
}

func compileSimpleScript(libVersion int, node Node) (*Executable, error) {
	fCheck, err := selectFunctionChecker(libVersion)
	if err != nil {
		return nil, err
	}
	b := newBuilder()
	c := newConstants()
	r := newReferences(nil)

	f := NewDefinitionsFsm(b, c, r, fCheck)
	f, err = ccc(f, node)
	if err != nil {
		return nil, err
	}
	// Just to write `OpReturn` to bytecode.
	f = f.Return()

	return f.(BuildExecutable).BuildExecutable(libVersion), nil

}
