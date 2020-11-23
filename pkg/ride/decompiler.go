package ride

import (
	"fmt"
	"strings"
)

type detreeType = func(s *strings.Builder, tree Node)

func prefix(s *strings.Builder, name string, nodes []Node, f detreeType) {
	s.WriteString(name)
	s.WriteString("(")
	for i, a := range nodes {
		f(s, a)
		if i+1 != len(nodes) {
			s.WriteString(",")
		}
	}
	s.WriteString(")")
}

func infix(s *strings.Builder, name string, nodes []Node, f detreeType) {
	s.WriteString("(")
	f(s, nodes[0])
	s.WriteString(fmt.Sprintf(" %s ", name))
	f(s, nodes[1])
	s.WriteString(")")
}

var defuncs = map[string]func(s *strings.Builder, name string, nodes []Node, f detreeType){
	"0": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		s.WriteString("(")
		f(s, nodes[0])
		s.WriteString(" == ")
		f(s, nodes[1])
		s.WriteString(")")
	},
	"100": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		infix(s, "+", nodes, f)
	},
	"101": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		infix(s, "-", nodes, f)
	},
	"1": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "instanceOf", nodes, f)
	},
	"1052": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "getBinary", nodes, f)
	},
}

func defunc(s *strings.Builder, name string, nodes []Node, f detreeType) {
	if v, ok := defuncs[name]; ok {
		v(s, name, nodes, f)
	} else {
		s.WriteString(name)
		s.WriteString("(")
		for _, a := range nodes {
			detree(s, a)
			s.WriteString(",")
		}
		s.WriteString(")")
	}
}

func Decompiler(tree Node) string {
	s := &strings.Builder{}
	detree(s, tree)
	return s.String()
}

func detree(s *strings.Builder, tree Node) {
	switch n := tree.(type) {
	case *FunctionDeclarationNode:
		s.WriteString(fmt.Sprintf("func %s(", n.Name))
		for i, a := range n.Arguments {
			s.WriteString(a)
			if i+1 != len(n.Arguments) {
				s.WriteString(",")
			}
		}
		s.WriteString(") { ")
		detree(s, n.Body)
		s.WriteString(" } ")
		detree(s, n.Block)
	case *AssignmentNode:
		s.WriteString(fmt.Sprintf("let %s = ", n.Name))
		detree(s, n.Expression)
		s.WriteString("; ")
		detree(s, n.Block)
	case *ConditionalNode:
		s.WriteString(fmt.Sprintf("if ("))
		detree(s, n.Condition)
		s.WriteString(") { ")
		detree(s, n.TrueExpression)
		s.WriteString(" } else { ")
		detree(s, n.FalseExpression)
		s.WriteString(" }")
	case *FunctionCallNode:
		defunc(s, n.Name, n.Arguments, detree)
	case *ReferenceNode:
		s.WriteString(n.Name)
	case *StringNode:
		s.WriteString(`"`)
		s.WriteString(n.Value)
		s.WriteString(`"`)
	case *PropertyNode:
		detree(s, n.Object)
		s.WriteString(".")
		s.WriteString(n.Name)
	case *BooleanNode:
		s.WriteString(fmt.Sprintf("%t", n.Value))
	case *LongNode:
		s.WriteString(fmt.Sprintf("%d", n.Value))
	default:
		panic(fmt.Sprintf("unknown type %T", n))
	}
}
