package ride

import (
	"testing"

	"github.com/stretchr/testify/require"
)

/*
func abc(key: String) = {
	let x = 1
	let y = 2
	x + y
}
*/
func TestReverseFunc(t *testing.T) {
	n := &FunctionDeclarationNode{
		Name:      "abc",
		Arguments: []string{"key"},
		Body: &AssignmentNode{
			Name:       "x",
			Expression: &LongNode{Value: 1},
			Block: &AssignmentNode{
				Name:       "y",
				Expression: &LongNode{Value: 2},
				Block: &FunctionCallNode{
					Name: "+",
					Arguments: []Node{
						&ReferenceNode{Name: "x"},
						&ReferenceNode{Name: "y"},
					},
				},
			},
		},
	}

	rs := reverseTree(n, nil)

	require.Equal(t, &RFunc{
		Invocation: "",
		Name:       "abc",
		Arguments:  []string{"key"},
		Body: &RCall{
			Name: "+",
			Arguments: []RNode{
				&RRef{Name: "x"},
				&RRef{Name: "y"},
			},
			Assigments: []*RLet{
				{Name: "x", Body: &RLong{Value: 1}},
				{Name: "y", Body: &RLong{Value: 2}},
			},
		},
	}, rs)

}

/*
func abc(key: String) = {
	match getInteger(this, key) {
		case a: Int =>
			a
		case _ =>
			0
}
*/
func TestReverseFunc2(t *testing.T) {
	n := &FunctionDeclarationNode{
		Name:      "abc",
		Arguments: []string{"key"},
		Body: &AssignmentNode{
			Name: "$match0",
			Expression: &FunctionCallNode{
				Name: "1050",
				Arguments: []Node{
					&ReferenceNode{Name: "this"},
					&ReferenceNode{Name: "key"},
				},
			},
			Block: &ConditionalNode{
				Condition: &FunctionCallNode{
					Name: "1",
					Arguments: []Node{
						&ReferenceNode{Name: "$match0"},
						&StringNode{Value: "Int"},
					},
				},
				TrueExpression: &AssignmentNode{
					Name:       "a",
					Expression: &ReferenceNode{Name: "$match0"},
					Block:      &ReferenceNode{Name: "a"},
				},
				FalseExpression: &LongNode{Value: 0},
			},
		},
	}

	rs := reverseTree(n, nil)

	require.Equal(t, &RFunc{
		Invocation: "",
		Name:       "abc",
		Arguments:  []string{"key"},
		Body: &RCond{
			Cond: &RCall{
				Name: "1",
				Arguments: []RNode{
					&RRef{Name: "$match0"},
					&RString{Value: "Int"},
				},
			},
			True: &RRef{
				Name: "a",
				Assigments: []*RLet{
					{
						Name: "a",
						Body: &RRef{Name: "$match0"},
					},
				},
			},
			False: &RLong{Value: 0},
			Assigments: []*RLet{
				{Name: "$match0", Body: &RCall{
					Name: "1050",
					Arguments: []RNode{
						&RRef{Name: "this"},
						&RRef{Name: "key"},
					},
				}},
			},
		},
		//Assigments: []*RLet{
		//	{
		//		Name: "$match0",
		//		N:    0,
		//		Body: nil,
		//	},
		//},
	}, rs)

}
