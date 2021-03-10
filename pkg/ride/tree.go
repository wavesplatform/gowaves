package ride

type Node interface {
	node()
	SetBlock(node Node)
	Clone() Node
}

type LongNode struct {
	Value int64
}

func (*LongNode) node() {}

func (*LongNode) SetBlock(Node) {}

func (a *LongNode) Clone() Node {
	return &*a
}

func NewLongNode(v int64) *LongNode {
	return &LongNode{Value: v}
}

type BytesNode struct {
	Value []byte
}

func (*BytesNode) node() {}

func (*BytesNode) SetBlock(Node) {}

func (a *BytesNode) Clone() Node {
	// Bytes references to the same location, but it makes no sense to modify them.
	return &*a
}

func NewBytesNode(v []byte) *BytesNode {
	return &BytesNode{Value: v}
}

type StringNode struct {
	Value string
}

func (*StringNode) node() {}

func (*StringNode) SetBlock(Node) {}

func (a *StringNode) Clone() Node {
	return &*a
}

func NewStringNode(v string) *StringNode {
	return &StringNode{Value: v}
}

type BooleanNode struct {
	Value bool
}

func (*BooleanNode) node() {}

func (*BooleanNode) SetBlock(Node) {}

func (a *BooleanNode) Clone() Node {
	return &*a
}

func NewBooleanNode(v bool) *BooleanNode {
	return &BooleanNode{Value: v}
}

type ConditionalNode struct {
	Condition       Node
	TrueExpression  Node
	FalseExpression Node
}

func (*ConditionalNode) node() {}

func (*ConditionalNode) SetBlock(Node) {}

func (a *ConditionalNode) Clone() Node {
	return &ConditionalNode{
		Condition:       a.Condition.Clone(),
		TrueExpression:  a.TrueExpression.Clone(),
		FalseExpression: a.FalseExpression.Clone(),
	}
}

func NewConditionalNode(condition, trueExpression, falseExpression Node) *ConditionalNode {
	return &ConditionalNode{
		Condition:       condition,
		TrueExpression:  trueExpression,
		FalseExpression: falseExpression,
	}
}

type AssignmentNode struct {
	Name       string
	Expression Node
	Block      Node
}

func (*AssignmentNode) node() {}

func (a *AssignmentNode) SetBlock(node Node) {
	a.Block = node
}

func (a *AssignmentNode) Clone() Node {
	return &AssignmentNode{
		Name:       a.Name,
		Expression: a.Expression.Clone(),
		Block:      a.Block.Clone(),
	}
}

func NewAssignmentNode(name string, expression, block Node) *AssignmentNode {
	return &AssignmentNode{
		Name:       name,
		Expression: expression,
		Block:      block,
	}
}

type ReferenceNode struct {
	Name string
}

func (*ReferenceNode) node() {}

func (*ReferenceNode) SetBlock(Node) {}

func (a *ReferenceNode) Clone() Node {
	return &ReferenceNode{Name: a.Name}
}

func NewReferenceNode(name string) *ReferenceNode {
	return &ReferenceNode{Name: name}
}

type FunctionDeclarationNode struct {
	Name                string
	Arguments           []string
	Body                Node
	Block               Node
	invocationParameter string
}

func (*FunctionDeclarationNode) node() {}

func (n *FunctionDeclarationNode) SetBlock(node Node) {
	n.Block = node
}

func clone(n Node) Node {
	if n == nil {
		return n
	}
	return n.Clone()
}

func cloneFuncDecl(n *FunctionDeclarationNode, body Node, block Node) *FunctionDeclarationNode {
	return &FunctionDeclarationNode{
		Name:                n.Name,
		Arguments:           n.Arguments,
		Body:                body,
		Block:               block,
		invocationParameter: n.invocationParameter,
	}
}

func (n *FunctionDeclarationNode) Clone() Node {
	args := make([]string, len(n.Arguments))
	copy(args, n.Arguments)

	return &FunctionDeclarationNode{
		Name:                n.Name,
		Arguments:           args,
		Body:                n.Body.Clone(),
		Block:               clone(n.Block),
		invocationParameter: n.invocationParameter,
	}
}

func NewFunctionDeclarationNode(name string, arguments []string, body, block Node) *FunctionDeclarationNode {
	return &FunctionDeclarationNode{
		Name:      name,
		Arguments: arguments,
		Body:      body,
		Block:     block,
	}
}

type Nodes []Node

func (a Nodes) Clone() Nodes {
	out := make(Nodes, 0, len(a))
	for _, v := range a {
		out = append(out, v.Clone())
	}
	return out
}

func (a Nodes) Map(f func(Node) Node) Nodes {
	args := make([]Node, 0, len(a))
	for _, v := range a {
		args = append(args, f(v))
	}
	return args
}

type FunctionCallNode struct {
	Name      string
	Arguments Nodes
}

func (a *FunctionCallNode) ArgumentsCount() uint16 {
	return uint16(len(a.Arguments))
}

func (*FunctionCallNode) node() {}

func (*FunctionCallNode) SetBlock(Node) {}

func (a *FunctionCallNode) Clone() Node {
	return &FunctionCallNode{
		Name:      a.Name,
		Arguments: a.Arguments.Clone(),
	}
}

func NewFunctionCallNode(name string, arguments []Node) *FunctionCallNode {
	return &FunctionCallNode{
		Name:      name,
		Arguments: arguments,
	}
}

type PropertyNode struct {
	Name   string
	Object Node
}

func (*PropertyNode) node() {}

func (*PropertyNode) SetBlock(Node) {}

func (a *PropertyNode) Clone() Node {
	return &PropertyNode{
		Name:   a.Name,
		Object: a.Object.Clone(),
	}
}

func NewPropertyNode(name string, object Node) *PropertyNode {
	return &PropertyNode{
		Name:   name,
		Object: object,
	}
}

type ScriptMeta struct {
	Version int
	Bytes   []byte
}

type Tree struct {
	Digest       [32]byte
	AppVersion   int
	LibVersion   int
	HasBlockV2   bool
	Meta         ScriptMeta
	Declarations []Node
	Functions    []Node
	Verifier     Node
	Expanded     bool
}

func (t *Tree) HasVerifier() bool {
	return t.Verifier != nil
}

func (t *Tree) IsDApp() bool {
	return t.AppVersion != scriptApplicationVersion
}
