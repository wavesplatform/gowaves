package ride

import (
	"fmt"
	"strings"

	"github.com/mr-tron/base58"
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
	"!=": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		s.WriteString("(")
		f(s, nodes[0])
		s.WriteString(" != ")
		f(s, nodes[1])
		s.WriteString(")")
	},
	"100": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		infix(s, "+", nodes, f)
	},
	"101": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		infix(s, "-", nodes, f)
	},
	"103": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		infix(s, ">=", nodes, f)
	},
	"104": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		infix(s, "*", nodes, f)
	},
	"105": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		infix(s, "/", nodes, f)
	},
	"200": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "size", nodes, f)
	},
	"201": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "take", nodes, f)
	},
	"202": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "drop", nodes, f)
	},
	"203": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		infix(s, "+", nodes, f)
	},
	"300": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		infix(s, "+", nodes, f)
	},
	"1": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "instanceOf", nodes, f)
	},
	"401": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "getList", nodes, f)
	},
	"410": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "toBytes", nodes, f)
	},
	"411": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "toBytes", nodes, f)
	},
	"420": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "toString", nodes, f)
	},
	"500": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "sigVerify", nodes, f)
	},
	"504": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "rsaVerify", nodes, f)
	},
	"600": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "toBase58String", nodes, f)
	},
	"604": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "fromBase64String", nodes, f)
	},
	"2": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "throw", nodes, f)
	},
	"1050": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "getInteger", nodes, f)
	},
	"1052": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "getBinary", nodes, f)
	},
	"1201": func(s *strings.Builder, name string, nodes []Node, f detreeType) {
		prefix(s, "toInt", nodes, f)
	},
}

func defunc(s *strings.Builder, name string, nodes []Node, f detreeType) {
	if v, ok := defuncs[name]; ok {
		v(s, name, nodes, f)
	} else {
		s.WriteString(name)
		s.WriteString("(")
		for i, a := range nodes {
			detree(s, a)
			if len(nodes)-1 != i {
				s.WriteString(",")
			}
		}
		s.WriteString(")")
	}
}

func DecompileTree(t *Tree) string {
	var s strings.Builder
	for _, v := range t.Declarations {
		s.WriteString(Decompiler(v))
	}
	for _, v := range t.Functions {
		s.WriteString(Decompiler(v))
	}
	s.WriteString(Decompiler(t.Verifier))
	return strings.TrimSpace(s.String())
}

func Decompiler(tree Node) string {
	s := &strings.Builder{}
	detree(s, tree)
	return s.String()
}

func detree(s *strings.Builder, tree Node) {
	switch n := tree.(type) {
	case *FunctionDeclarationNode:
		if n.invocationParameter != "" {
			s.WriteString("@" + n.invocationParameter + "\\n")
		}
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
		s.WriteString(fmt.Sprintf("let %s = { ", n.Name))
		detree(s, n.Expression)
		s.WriteString(" }; ")
		detree(s, n.Block)
	case *ConditionalNode:
		s.WriteString("if (")
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
	case *BytesNode:
		s.WriteString("b58:")
		s.WriteString(base58.Encode(n.Value))
	case nil:
		// nothing
	default:
		panic(fmt.Sprintf("unknown type %T", n))
	}
}
