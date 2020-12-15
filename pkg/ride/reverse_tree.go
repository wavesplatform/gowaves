package ride

import "fmt"

func ReverseTree(n []Node) {
	for i := len(n) - 1; i >= 0; i-- {

	}
}

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
