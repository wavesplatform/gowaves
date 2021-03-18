package ride

import (
	"fmt"

	im "github.com/frozen/immutable_map"
	"github.com/pkg/errors"
)

type expandScope struct {
	im *im.Map
}

func newExpandScope() expandScope {
	return expandScope{
		im: im.New(),
	}
}

func (a expandScope) add(name string, n *FunctionDeclarationNode) expandScope {
	return expandScope{
		im: a.im.Insert([]byte(name), n),
	}
}

func (a expandScope) get(name string) (*FunctionDeclarationNode, bool) {
	v, ok := a.im.Get([]byte(name))
	if ok {
		return v.(*FunctionDeclarationNode), true
	}
	return nil, false
}

func (a expandScope) get1(name string) *FunctionDeclarationNode {
	v, ok := a.im.Get([]byte(name))
	if ok {
		return v.(*FunctionDeclarationNode)
	}
	return nil
}

func Expand(t *Tree) (*Tree, error) {
	if t.Expanded {
		return t, nil
	}
	scope := newExpandScope()
	declarations := make([]Node, 0, len(t.Declarations))
	for _, f := range t.Declarations {
		v, ok := f.(*FunctionDeclarationNode)
		if !ok {
			declarations = append(declarations, expand(scope, f))
			continue
		}
		v2 := cloneFuncDecl(v, expand(scope, v.Body), nil)
		scope = scope.add(v.Name, v2)
	}
	functions := make([]Node, 0, len(t.Functions))
	for _, f := range t.Functions {
		v, ok := f.(*FunctionDeclarationNode)
		if !ok {
			return nil, errors.Errorf("can't expand tree. Expected function to be `*FunctionDeclarationNode`, found %T", f)
		}
		v2 := v.Clone().(*FunctionDeclarationNode)
		v2.Body = expand(scope, v.Body)
		scope = scope.add(v.Name, v2)
		functions = append(functions, v2)
	}
	verifier := t.Verifier
	if t.IsDApp() && t.HasVerifier() {
		verifier = cloneFuncDecl(t.Verifier.(*FunctionDeclarationNode), expand(scope, t.Verifier.(*FunctionDeclarationNode).Body), nil)
	} else {
		verifier = expand(scope, t.Verifier)
	}
	return &Tree{
		Digest:       t.Digest,
		AppVersion:   t.AppVersion,
		LibVersion:   t.LibVersion,
		HasBlockV2:   t.HasBlockV2,
		Meta:         t.Meta,
		Declarations: declarations,
		Functions:    functions,
		Verifier:     verifier,
		Expanded:     true,
	}, nil
}

func MustExpand(t *Tree) *Tree {
	rs, err := Expand(t)
	if err != nil {
		panic(err)
	}
	return rs
}

func expand(scope expandScope, node Node) Node {
	switch v := node.(type) {
	case *FunctionCallNode:
		f, ok := scope.get(v.Name)
		if ok {
			root := f.Body
			for i := len(v.Arguments) - 1; i >= 0; i-- {
				root = &AssignmentNode{
					Name:       f.Arguments[i],
					Expression: expand(scope, v.Arguments[i]),
					Block:      root,
				}
			}
			return root
		} else {
			return &FunctionCallNode{
				Name: v.Name,
				Arguments: v.Arguments.Map(func(node Node) Node {
					return expand(scope, node)
				}),
			}
		}

	case *FunctionDeclarationNode:
		body := expand(scope, v.Body)
		v2 := cloneFuncDecl(v, body, nil)
		block := expand(scope.add(v.Name, v2), v.Block)
		return block

	case *AssignmentNode:
		return &AssignmentNode{
			Name:       v.Name,
			Block:      expand(scope, v.Block),
			Expression: expand(scope, v.Expression),
		}
	case nil:
		return node
	case *ConditionalNode:
		return &ConditionalNode{
			Condition:       expand(scope, v.Condition),
			TrueExpression:  expand(scope, v.TrueExpression),
			FalseExpression: expand(scope, v.FalseExpression),
		}
	case *ReferenceNode:
		return v
	case *StringNode, *LongNode, *BytesNode, *BooleanNode, *PropertyNode:
		return v
	default:
		panic(fmt.Sprintf("unknown %T", node))
	}
}
