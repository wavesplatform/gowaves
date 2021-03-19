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

type nameReplacements struct {
	im *im.Map
}

func newNameReplacements() nameReplacements {
	return nameReplacements{
		im: im.New(),
	}
}

func (a nameReplacements) add(original string, replacement string) nameReplacements {
	return nameReplacements{
		im: a.im.Insert([]byte(original), replacement),
	}
}

func (a nameReplacements) addAll(postfix string, args []string) nameReplacements {
	tmp := a
	for _, arg := range args {
		tmp = tmp.add(arg, arg+postfix)
	}
	return tmp
}

func (a nameReplacements) get(name string) string {
	inf, ok := a.im.Get([]byte(name))
	if ok {
		return inf.(string)
	}
	return name
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
			declarations = append(declarations, expand(scope, f, newNameReplacements()))
			continue
		}
		v2 := cloneFuncDecl(v, expand(scope, v.Body, newNameReplacements().addAll("$"+v.Name, v.Arguments)), nil)
		scope = scope.add(v.Name, v2)
	}
	functions := make([]Node, 0, len(t.Functions))
	for _, f := range t.Functions {
		v, ok := f.(*FunctionDeclarationNode)
		if !ok {
			return nil, errors.Errorf("can't expand tree. Expected function to be `*FunctionDeclarationNode`, found %T", f)
		}
		v2 := v.Clone().(*FunctionDeclarationNode)
		v2.Body = expand(scope, v.Body, newNameReplacements())
		scope = scope.add(v.Name, v2)
		functions = append(functions, v2)
	}
	verifier := t.Verifier
	if t.IsDApp() && t.HasVerifier() {
		verifier = cloneFuncDecl(t.Verifier.(*FunctionDeclarationNode), expand(scope, t.Verifier.(*FunctionDeclarationNode).Body, newNameReplacements()), nil)
	} else {
		verifier = expand(scope, t.Verifier, newNameReplacements())
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

func expand(scope expandScope, node Node, replacements nameReplacements) Node {
	switch v := node.(type) {
	case *FunctionCallNode:
		f, ok := scope.get(v.Name)
		if ok {
			root := f.Body
			for i := len(v.Arguments) - 1; i >= 0; i-- {
				root = &AssignmentNode{
					Name:       fmt.Sprintf("%s$%s", f.Arguments[i], f.Name),
					Expression: expand(scope, v.Arguments[i], replacements),
					Block:      root,
				}
			}
			return root
		} else {
			return &FunctionCallNode{
				Name: v.Name,
				Arguments: v.Arguments.Map(func(node Node) Node {
					return expand(scope, node, replacements)
				}),
			}
		}

	case *FunctionDeclarationNode:

		body := expand(scope, v.Body, replacements.addAll("$"+v.Name, v.Arguments))
		v2 := cloneFuncDecl(v, body, nil)
		block := expand(scope.add(v.Name, v2), v.Block, replacements)
		return block

	case *AssignmentNode:
		return &AssignmentNode{
			Name:       v.Name,
			Block:      expand(scope, v.Block, replacements),
			Expression: expand(scope, v.Expression, replacements),
		}
	case nil:
		return node
	case *ConditionalNode:
		return &ConditionalNode{
			Condition:       expand(scope, v.Condition, replacements),
			TrueExpression:  expand(scope, v.TrueExpression, replacements),
			FalseExpression: expand(scope, v.FalseExpression, replacements),
		}
	case *ReferenceNode:
		return &ReferenceNode{Name: replacements.get(v.Name)}
	case *StringNode, *LongNode, *BytesNode, *BooleanNode:
		return v
	case *PropertyNode:
		return &PropertyNode{
			Name:   v.Name,
			Object: expand(scope, v.Object, replacements),
		}
	default:
		panic(fmt.Sprintf("unknown %T", node))
		return v.Clone()
	}
}
