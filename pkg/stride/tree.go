package stride

type Node interface {
	node()
}

type DeclarationNode interface {
	Node
	SetBlock(node Node)
}

type LongNode struct {
	Value int64
}

func (*LongNode) node() {}

func NewLongNode(v int64) *LongNode {
	return &LongNode{Value: v}
}

type BytesNode struct {
	Value []byte
}

func (*BytesNode) node() {}

func NewBytesNode(v []byte) *BytesNode {
	return &BytesNode{Value: v}
}

type StringNode struct {
	Value string
}

func (*StringNode) node() {}

func NewStringNode(v string) *StringNode {
	return &StringNode{Value: v}
}

type BooleanNode struct {
	Value bool
}

func (*BooleanNode) node() {}

func NewBooleanNode(v bool) *BooleanNode {
	return &BooleanNode{Value: v}
}

type ConditionalNode struct {
	Condition       Node
	TrueExpression  Node
	FalseExpression Node
}

func (*ConditionalNode) node() {}

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

func (n *AssignmentNode) SetBlock(node Node) {
	n.Block = node
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

func (ReferenceNode) node() {}

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

func NewFunctionDeclarationNode(name string, arguments []string, body, block Node) *FunctionDeclarationNode {
	return &FunctionDeclarationNode{
		Name:      name,
		Arguments: arguments,
		Body:      body,
		Block:     block,
	}
}

type FunctionCallNode struct {
	Name      string
	Arguments []Node
}

func (*FunctionCallNode) node() {}

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
}

func (t *Tree) HasVerifier() bool {
	return t.Verifier != nil
}

func (t *Tree) IsDApp() bool {
	return t.AppVersion != scriptApplicationVersion
}
