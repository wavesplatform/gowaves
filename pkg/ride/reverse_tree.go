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

//func compileArguments(call *RCall, out *bytes.Buffer) {
//	d := []RNode{
//		call,
//	}
//	for i := len(call.Arguments) - 1; i >= 0; i-- {
//		switch call.Arguments[i].(type) {
//		case *RLong:
//			d = append(d, RLong{})
//		}
//	}
//}

func reverseTree2(n Node, out []RNode, deferreds []RNode) []RNode {
	switch t := n.(type) {
	case *FunctionDeclarationNode:
		out = append(out, &RFunc{Name: t.Name, Arguments: t.Arguments, Invocation: t.invocationParameter})
		return reverseTree2(t.Body, out, nil)
	case *AssignmentNode:
		return reverseTree2(t.Block, out, append(deferreds, &RLet{Name: t.Name, Body: reverseTree2(t.Expression, nil, nil)}))
	case *LongNode:
		return append(out, &RLong{Value: t.Value})
	case *FunctionCallNode:
		out = append(out, reverseRnodes(flatCall(t))...)
		return append(out, deferreds...)
	default:
		panic(fmt.Sprintf("unknown type %T", n))
	}
}

func flatCall(call *FunctionCallNode) []RNode {
	out := []RNode{
		&RCall{
			Name: call.Name,
		},
	}
	for i := len(call.Arguments) - 1; i >= 0; i-- {
		switch t := call.Arguments[i].(type) {
		case *LongNode:
			out = append(out, &RLong{Value: t.Value})
		case *FunctionCallNode:
			out = append(out, flatCall(t)...)
		case *ReferenceNode:
			out = append(out, &RRef{Name: t.Name})
		default:
			panic(fmt.Sprintf("unknown type %T", call.Arguments[i]))
		}
	}
	return out
}
