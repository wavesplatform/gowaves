package ride

import (
	"fmt"
)

func ReverseTree(n []Node) {
	for i := len(n) - 1; i >= 0; i-- {

	}
}

/*
func reverseTree(n Node, r []*RLet) RNode {
	switch v := n.(type) {
	case *FunctionDeclarationNode:
		return &RFunc{
			Invocation: v.invocationParameter,
			Name:       v.Name,
			Arguments:  v.Arguments,
			Body:       reverseTree(v.Body, r),
		}
	case *LongNode:
		return &RLong{Value: v.Value}
	case *FunctionCallNode:
		args := make([]RNode, len(v.Arguments))
		for i := range v.Arguments {
			args[i] = reverseTree(v.Arguments[i], nil)
		}
		return &RCall{Name: v.Name, Arguments: args, Assigments: r}
	case *ReferenceNode:
		return &RRef{Name: v.Name, Assigments: r}
	case *AssignmentNode:
		return reverseTree(v.Block, append(r, &RLet{Name: v.Name, Body: reverseTree(v.Expression, r)}))
	case *ConditionalNode:
		return &RCond{
			Cond:       reverseTree(v.Condition, nil),
			True:       reverseTree(v.TrueExpression, nil),
			False:      reverseTree(v.FalseExpression, nil),
			Assigments: r,
		}
	case *StringNode:
		return &RString{Value: v.Value}
	default:
		panic(fmt.Sprintf("unknown type %T", n))
	}
}
*/

//
//func walkAssigments(n Node, r []*RLet) ([]*RLet, Node) {
//	switch v := n.(type) {
//	case *AssignmentNode:
//		r = append(r, &RLet{
//			Name: v.Name,
//			N:    0,
//			Body: reverseTree(v.Expression),
//		})
//		return walkAssigments(v.Block, r)
//	default:
//		return r, v
//	}
//}

//func compileReverse(r RNode, out *bytes.Buffer) {
//	switch t := r.(type) {
//	case *RCall:
//		for i, v := range reverseCallTree(t.CallTree()) {
//
//		}
//
//		out.Write()
//	}
//
//}

type startpoint struct {
	fns      []func()
	default_ func()
}

func (a *startpoint) push(f func()) {
	a.fns = append(a.fns, f)
}

func (a *startpoint) evalAndPop() {
	if len(a.fns) > 0 {
		a.fns[len(a.fns)-1]()
		a.fns = a.fns[:len(a.fns)-1]
	} else {
		a.default_()
	}

}

//func compileReversedTree(nodes []RNode, libVersion int, isDapp bool, hasVerifier bool) (*Executable, error) {
//	var condPos []uint16
//	refs := newReferences(nil)
//	out := bytes.Buffer{}
//	out.WriteByte(OpReturn)
//	out.WriteByte(OpReturn)
//	c := newCell()
//	u := uniqid{}
//	u.next()
//	entrypoints := make(map[string]Entrypoint)
//
//	st := startpoint{
//		default_: func() {
//			entrypoints[""] = Entrypoint{
//				name: "",
//				at:   uint16(out.Len()),
//				argn: 0,
//			}
//		},
//	}
//	st.evalAndPop()
//
//	for k, v := range predefinedFunctions {
//		id := uint16(math.MaxUint16 - k)
//		refs.set(v.name, id)
//		c.set(id, nil, id, 0, false, v.name)
//	}
//
//	//
//	for _, v := range nodes {
//		switch v := v.(type) {
//		case *RDef:
//			n := u.next()
//			refs.set(v.Name, n)
//			c.set(n, nil, 0, 0, false, fmt.Sprintf("rdef %d, named %s", n, v.Name))
//			for range v.Arguments {
//				u.next()
//			}
//		case *RConst:
//			n := u.next()
//			c.set(n, v.Value, 0, 0, true, fmt.Sprintf("constant %q", v.Value))
//			out.WriteByte(OpRef)
//			out.Write(encode(n))
//		case *RCall:
//			if n, ok := refs.get(v.Name); ok {
//				out.WriteByte(OpRef)
//				out.Write(encode(n))
//				continue
//			}
//			fCheck, err := selectFunctionChecker(libVersion)
//			if err != nil {
//				return nil, err
//			}
//			out.WriteByte(OpExternalCall)
//			id, ok := fCheck(v.Name)
//			if !ok {
//				return nil, errors.Errorf("invalid func name `%s`", v.Name)
//			}
//			out.Write(encode(id))
//			out.Write(encode(v.Argn))
//		case *RReferenceNode:
//			n, ok := refs.get(v.Name)
//			if !ok {
//				return nil, errors.Errorf("reference `%s` not found", v.Name)
//			}
//			out.WriteByte(OpRef)
//			out.Write(encode(n))
//		case *RLet:
//			n := u.next()
//			refs.set(v.Name, n)
//			c.set(n, nil, 0, 0, false, fmt.Sprintf("rdef %d, named %s", n, v.Name))
//			n, ok := refs.get(v.Name)
//			if !ok {
//				return nil, errors.Errorf("reference `%s` not found", v.Name)
//			}
//			e, ok := c.values[n]
//			if !ok {
//				return nil, errors.Errorf("cell `%d` not found", n)
//			}
//			e.position = uint16(out.Len())
//			c.values[n] = e
//		case *RFunc:
//			n := u.next()
//			refs.set(v.Name, n)
//			refs = newReferences(refs)
//			c.set(n, nil, 0, 0, false, fmt.Sprintf("ref %d, func named %s", n, v.Name))
//			for i := range v.ArgumentsWithInvocation {
//				z := u.next()
//				c.set(z, nil, 0, 0, false, fmt.Sprintf("ref %d, func arg #%d %s", z, i, v.Name))
//				refs.set(v.ArgumentsWithInvocation[i], z)
//			}
//			if v.Invocation != "" {
//				st.push(func() {
//					entrypoints[v.Name] = Entrypoint{
//						name: v.Name,
//						at:   uint16(out.Len()),
//						argn: uint16(len(v.Arguments)),
//					}
//					for i := len(v.Arguments) + 1; i > 0; i-- {
//						out.WriteByte(OpCache)
//						out.Write(encode(n + uint16(i)))
//						out.WriteByte(OpPop)
//					}
//				})
//			} else {
//				st.push(func() {
//					c.set(n, nil, 0, uint16(out.Len()), false, fmt.Sprintf("ref %d, func named %s", n, v.Name))
//					for i := len(v.ArgumentsWithInvocation); i > 0; i-- {
//						out.WriteByte(OpCache)
//						out.Write(encode(n + uint16(i)))
//						out.WriteByte(OpPop)
//					}
//				})
//			}
//		case *RFuncEnd:
//			refs = refs.pop()
//
//		case *RRet:
//			out.WriteByte(OpReturn)
//		case *RCond:
//			condPos = append(condPos, uint16(out.Len()))
//			out.WriteByte(OpJumpIfFalse)
//			out.Write(make([]byte, 6))
//		case *RCondTrue:
//			pos := condPos[len(condPos)-1]
//			st.push(func() {
//				patchBuffer(&out, pos+1, encode(uint16(out.Len())))
//			})
//		case *RCondFalse:
//			pos := condPos[len(condPos)-1]
//			st.push(func() {
//				patchBuffer(&out, pos+3, encode(uint16(out.Len())))
//			})
//
//		case *RCondEnd:
//			pos := condPos[len(condPos)-1]
//			patchBuffer(&out, pos+5, encode(uint16(out.Len())))
//			condPos = condPos[:len(condPos)-1]
//		case *RStart:
//			st.evalAndPop()
//
//		case *RProperty:
//			out.WriteByte(OpProperty)
//
//		default:
//			panic(fmt.Sprintf("unknown type %T", v))
//		}
//
//	}
//
//	// Most recent code line.
//	out.WriteByte(OpReturn)
//
//	if isDapp && hasVerifier {
//		entrypoints[""] = entrypoints["verify"]
//	}
//
//	e := Executable{
//		LibVersion:  libVersion,
//		IsDapp:      isDapp,
//		hasVerifier: hasVerifier,
//		ByteCode:    out.Bytes(),
//		EntryPoints: entrypoints,
//		References:  c.values,
//	}
//
//	return &e, nil
//}

//func compileReversedTree(nodes []RNode, libVersion int, isDapp bool, hasVerifier bool) (*Executable, error) {
//	var condPos []uint16
//	refs := newReferences(nil)
//	out := bytes.Buffer{}
//	out.WriteByte(OpReturn)
//	out.WriteByte(OpReturn)
//	c := newCell()
//	u := uniqid{}
//	u.next()
//	entrypoints := make(map[string]Entrypoint)
//
//	st := startpoint{
//		default_: func() {
//			entrypoints[""] = Entrypoint{
//				name: "",
//				at:   uint16(out.Len()),
//				argn: 0,
//			}
//		},
//	}
//	st.evalAndPop()
//
//	for k, v := range predefinedFunctions {
//		id := uint16(math.MaxUint16 - k)
//		refs.setAssigment(v.name, id)
//		c.set(id, nil, id, 0, false, v.name)
//	}
//
//	//
//	for _, v := range nodes {
//		switch v := v.(type) {
//		case *RDef:
//			n := u.next()
//			refs.set(v.Name, n)
//			c.set(n, nil, 0, 0, false, fmt.Sprintf("rdef %d, named %s", n, v.Name))
//			for range v.Arguments {
//				u.next()
//			}
//		//case *RBody:
//		//	n, ok := refs.get(v.Name, n)
//		//	if !ok {
//		//		return errors.Errorf()
//		//	}
//		case *RConst:
//			n := u.next()
//			c.set(n, v.Value, 0, 0, true, fmt.Sprintf("constant %q", v.Value))
//			out.WriteByte(OpRef)
//			out.Write(encode(n))
//		case *RCall:
//			if n, ok := refs.get(v.Name); ok {
//				out.WriteByte(OpRef)
//				out.Write(encode(n))
//				continue
//			}
//			fCheck, err := selectFunctionChecker(libVersion)
//			if err != nil {
//				return nil, err
//			}
//			out.WriteByte(OpExternalCall)
//			id, ok := fCheck(v.Name)
//			if !ok {
//				return nil, errors.Errorf("invalid func name `%s`", v.Name)
//			}
//			out.Write(encode(id))
//			out.Write(encode(v.Argn))
//		case *RReferenceNode:
//			n, ok := refs.get(v.Name)
//			if !ok {
//				return nil, errors.Errorf("reference `%s` not found", v.Name)
//			}
//			out.WriteByte(OpRef)
//			out.Write(encode(n))
//		case *RLet:
//			n := u.next()
//			refs.set(v.Name, n)
//			c.set(n, nil, 0, 0, false, fmt.Sprintf("rdef %d, named %s", n, v.Name))
//			n, ok := refs.get(v.Name)
//			if !ok {
//				return nil, errors.Errorf("reference `%s` not found", v.Name)
//			}
//			e, ok := c.values[n]
//			if !ok {
//				return nil, errors.Errorf("cell `%d` not found", n)
//			}
//			e.position = uint16(out.Len())
//			c.values[n] = e
//		case *RFunc:
//			n := u.next()
//			refs.set(v.Name, n)
//			refs = newReferences(refs)
//			c.set(n, nil, 0, 0, false, fmt.Sprintf("ref %d, func named %s", n, v.Name))
//			for i := range v.ArgumentsWithInvocation {
//				z := u.next()
//				c.set(z, nil, 0, 0, false, fmt.Sprintf("ref %d, func arg #%d %s", z, i, v.Name))
//				refs.set(v.ArgumentsWithInvocation[i], z)
//			}
//			if v.Invocation != "" {
//				st.push(func() {
//					entrypoints[v.Name] = Entrypoint{
//						name: v.Name,
//						at:   uint16(out.Len()),
//						argn: uint16(len(v.Arguments)),
//					}
//					for i := len(v.Arguments) + 1; i > 0; i-- {
//						out.WriteByte(OpCache)
//						out.Write(encode(n + uint16(i)))
//						out.WriteByte(OpPop)
//					}
//				})
//			} else {
//				st.push(func() {
//					c.set(n, nil, 0, uint16(out.Len()), false, fmt.Sprintf("ref %d, func named %s", n, v.Name))
//					for i := len(v.ArgumentsWithInvocation); i > 0; i-- {
//						out.WriteByte(OpCache)
//						out.Write(encode(n + uint16(i)))
//						out.WriteByte(OpPop)
//					}
//				})
//			}
//		case *RFuncEnd:
//			refs = refs.pop()
//
//		case *RRet:
//			out.WriteByte(OpReturn)
//		case *RCond:
//			condPos = append(condPos, uint16(out.Len()))
//			out.WriteByte(OpJumpIfFalse)
//			out.Write(make([]byte, 6))
//		case *RCondTrue:
//			pos := condPos[len(condPos)-1]
//			st.push(func() {
//				patchBuffer(&out, pos+1, encode(uint16(out.Len())))
//			})
//		case *RCondFalse:
//			pos := condPos[len(condPos)-1]
//			st.push(func() {
//				patchBuffer(&out, pos+3, encode(uint16(out.Len())))
//			})
//
//		case *RCondEnd:
//			pos := condPos[len(condPos)-1]
//			patchBuffer(&out, pos+5, encode(uint16(out.Len())))
//			condPos = condPos[:len(condPos)-1]
//		case *RStart:
//			st.evalAndPop()
//
//		case *RProperty:
//			out.WriteByte(OpProperty)
//
//		default:
//			panic(fmt.Sprintf("unknown type %T", v))
//		}
//
//	}
//
//	// Most recent code line.
//	out.WriteByte(OpReturn)
//
//	if isDapp && hasVerifier {
//		entrypoints[""] = entrypoints["verify"]
//	}
//
//	e := Executable{
//		LibVersion:  libVersion,
//		IsDapp:      isDapp,
//		hasVerifier: hasVerifier,
//		ByteCode:    out.Bytes(),
//		EntryPoints: entrypoints,
//		References:  c.values,
//	}
//
//	return &e, nil
//}

type ddfrs struct {
}

/*
func reverseTree2(n Node, out []RNode, deferreds *ddfrs) []RNode {
	switch t := n.(type) {
	case *FunctionDeclarationNode:
		out = append(out, &RDef{Name: t.Name, Arguments: t.Arguments})
		temp := []RNode{&RFunc{Name: t.Name, Arguments: t.Arguments, Invocation: t.invocationParameter}}
		// func body
		d := &ddfrs{}
		temp = append(temp, reverseTree2(t.Body, nil, d)...)
		// end of function
		temp = append(temp, &RFuncEnd{})
		temp = append(temp, &RRet{})
		temp = append(temp, d.deferreds...)
		return reverseTree2(t.Block, out, deferreds)
	case *AssignmentNode:
		out = append([]RNode{&RDef{Name: t.Name}}, out...)
		deferreds.add(&RLet{Name: t.Name})
		//d := &ddfrs{}
		deferreds = reverseTree2(t.Expression, nil, deferreds)
		deferreds = append(deferreds, &RRet{})
		return reverseTree2(t.Block, out, deferreds)
	case *ConditionalNode:
		cond := reverseTree2(t.Condition, nil, nil)
		out = append(out, cond...)
		out = append(out, &RCond{})
		out = append(out, &RCondTrue{})
		out = append(out, reverseTree2(t.TrueExpression, nil, nil)...)
		out = append(out, &RRet{})
		out = append(out, &RCondFalse{})
		out = append(out, reverseTree2(t.FalseExpression, nil, nil)...)
		out = append(out, &RRet{})
		out = append(out, &RCondEnd{})
		//if len(deferreds) > 0 {
		//	out = append(out, &RRet{})
		//}
		//return append(out, deferreds...)
		return out
	case *LongNode:
		return append(out, &RConst{Value: rideInt(t.Value)})
	case *FunctionCallNode:
		out = append(out, flatCall(t)...)
		if len(deferreds) > 0 {
			out = append(out, &RRet{})
		}
		return append(out, deferreds...)
	case *BooleanNode:
		return append(out, &RConst{Value: rideBoolean(t.Value)})
	case *StringNode:
		return append(out, &RConst{Value: rideString(t.Value)})
	case *BytesNode:
		return append(out, &RConst{Value: rideBytes(t.Value)})
	case *ReferenceNode:
		return append(append(out, &RReferenceNode{Name: t.Name}), deferreds...)
	default:
		panic(fmt.Sprintf("unknown type %T", n))
	}
}
*/

// Check is definition.
func isDef(n []RNode) bool {
	switch n[0].(type) {
	case *RLet, *RFunc:
		return true
	default:
		return false
	}
}

func appendInvocation(i string, args []string) []string {
	if i != "" {
		return append([]string{i}, args...)
	}
	return args
}

func reverseTree3(n Node, out []RNode) []RNode {
	switch t := n.(type) {
	case *FunctionDeclarationNode:
		out = append(out, &RFunc{
			Name:                    t.Name,
			Arguments:               t.Arguments,
			Invocation:              t.invocationParameter,
			ArgumentsWithInvocation: appendInvocation(t.invocationParameter, t.Arguments),
		})
		// func body
		temp := reverseTree3(t.Body, nil)
		if !isDef(temp) {
			temp = append([]RNode{&RStart{}}, temp...)
		}
		out = append(out, temp...)
		out = append(out, &RFuncEnd{})
		out = append(out, &RRet{})
		switch t.Block.(type) {
		case *AssignmentNode, *FunctionDeclarationNode, nil:
			return append(out, reverseTree3(t.Block, nil)...)
		default:
			out = append(out, &RStart{})
			return append(out, reverseTree3(t.Block, nil)...)
		}
	case *BooleanNode:
		return append(out, &RConst{Value: rideBoolean(t.Value)})
	case *AssignmentNode:
		out = append(out, &RLet{Name: t.Name})
		out = append(out, reverseTree3(t.Expression, nil)...)
		out = append(out, &RRet{})
		switch t.Block.(type) {
		case *AssignmentNode:
			return append(out, reverseTree3(t.Block, nil)...)
		case *FunctionDeclarationNode:
			return append(out, reverseTree3(t.Block, nil)...)
		default:
			out = append(out, &RStart{})
			return append(out, reverseTree3(t.Block, nil)...)
		}
	case *LongNode:
		return append(out, &RConst{Value: rideInt(t.Value)})
	case *StringNode:
		return append(out, &RConst{Value: rideString(t.Value)})
	case *BytesNode:
		return append(out, &RConst{Value: rideBytes(t.Value)})
	case *ReferenceNode:
		return append(out, &RReferenceNode{Name: t.Name})
	case *FunctionCallNode:
		out = append(out, flatCall2(t.Arguments)...)
		out = append(out, &RCall{
			Name: t.Name,
			Argn: uint16(len(t.Arguments)),
		})
		return out
	case *ConditionalNode:
		out = append(out, flatCondNode(t)...)
		return out
	case *PropertyNode:
		out = append(out, flatProperty(t)...)
		return out
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("unknown type %T", n))
	}
}

//func CompileFlatTree(tree *Tree) (*Executable, error) {
//	if tree.IsDApp() {
//		var rev []RNode
//		for _, v := range tree.Declarations {
//			rev = append(rev, reverseTree3(v, nil)...)
//		}
//		for _, v := range tree.Functions {
//			rev = append(rev, reverseTree3(v, nil)...)
//		}
//		if tree.HasVerifier() {
//			rev = append(rev, reverseTree3(tree.Verifier, nil)...)
//		}
//		return compileReversedTree(rev, tree.LibVersion, tree.IsDApp(), tree.HasVerifier())
//	}
//	return compileReversedTree(reverseTree3(tree.Verifier, nil), tree.LibVersion, tree.IsDApp(), tree.HasVerifier())
//}

func flatCondNode(t *ConditionalNode) []RNode {
	var out []RNode
	cond := reverseTree3(t.Condition, nil)
	out = append(out, cond...)
	out = append(out, &RCond{})
	out = append(out, &RCondTrue{})
	switch t.TrueExpression.(type) {
	case *AssignmentNode:
		out = append(out, reverseTree3(t.TrueExpression, nil)...)
	default:
		out = append(out, &RStart{})
		out = append(out, reverseTree3(t.TrueExpression, nil)...)
	}
	out = append(out, &RRet{})
	out = append(out, &RCondFalse{})
	switch t.FalseExpression.(type) {
	case *AssignmentNode:
		out = append(out, reverseTree3(t.FalseExpression, nil)...)
	default:
		out = append(out, &RStart{})
		out = append(out, reverseTree3(t.FalseExpression, nil)...)
	}
	out = append(out, &RRet{})
	out = append(out, &RCondEnd{})
	return out
}

//func flatCall(call *FunctionCallNode) []RNode {
//	out := []RNode{
//		&RCall{
//			Name: call.Name,
//			Argn: uint16(len(call.Arguments)),
//		},
//	}
//	for i := len(call.Arguments) - 1; i >= 0; i-- {
//		switch t := call.Arguments[i].(type) {
//		case *LongNode:
//			out = append(out, &RConst{Value: rideInt(t.Value)})
//		case *BooleanNode:
//			out = append(out, &RConst{Value: rideBoolean(t.Value)})
//		case *BytesNode:
//			out = append(out, &RConst{Value: rideBytes(t.Value)})
//		case *StringNode:
//			out = append(out, &RConst{Value: rideString(t.Value)})
//		case *FunctionCallNode:
//			out = append(out, flatCall(t)...)
//		case *ReferenceNode:
//			out = append(out, &RReferenceNode{Name: t.Name})
//		case *PropertyNode:
//			out = append(out, reverseRnodes(flatProperty(t))...)
//		case *ConditionalNode:
//			out = append(out, reverseRnodes(flatCondNode(t))...)
//		default:
//			panic(fmt.Sprintf("unknown type %T", call.Arguments[i]))
//		}
//	}
//	return out
//}

func flatCall2(Arguments []Node) []RNode {
	var out = []RNode{}
	for i := len(Arguments) - 1; i >= 0; i-- {
		switch t := Arguments[i].(type) {
		case *LongNode:
			out = append([]RNode{&RConst{Value: rideInt(t.Value)}}, out...)
		case *BooleanNode:
			out = append([]RNode{&RConst{Value: rideBoolean(t.Value)}}, out...)
		case *BytesNode:
			out = append([]RNode{&RConst{Value: rideBytes(t.Value)}}, out...)
		case *StringNode:
			out = append([]RNode{&RConst{Value: rideString(t.Value)}}, out...)
		case *FunctionCallNode:
			call := flatCall2(t.Arguments)
			call = append(call, &RCall{
				Name: t.Name,
				Argn: t.ArgumentsCount(),
			})
			out = append(call, out...)
		case *ReferenceNode:
			out = append([]RNode{&RReferenceNode{Name: t.Name}}, out...)
		case *PropertyNode:
			tmp := flatProperty(t)
			out = append(tmp, out...)
		case *ConditionalNode:
			tmp := flatCondNode(t)
			out = append(tmp, out...)
		default:
			panic(fmt.Sprintf("unknown type %T", Arguments[i]))
		}
	}
	return out
}

func flatProperty(p *PropertyNode) []RNode {
	switch v := p.Object.(type) {
	case *ReferenceNode:
		return []RNode{&RReferenceNode{Name: v.Name}, &RConst{Value: rideString(p.Name)}, &RProperty{}}
	case *PropertyNode:
		return append(flatProperty(v), &RConst{Value: rideString(p.Name)}, &RProperty{})
	case *FunctionCallNode:
		call := flatCall2(v.Arguments)
		call = append(call, &RCall{
			Name: v.Name,
			Argn: v.ArgumentsCount(),
		})
		return append(call, &RConst{Value: rideString(p.Name)}, &RProperty{})
	default:
		panic(fmt.Sprintf("unknown type %T", v))
	}
}
