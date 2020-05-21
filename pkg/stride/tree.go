package stride

const (
	LongLiteral = iota + 1
	BytesLiteral
	StringLiteral
	BooleanLiteral
	IfExpression
	LetExpression
	ReferenceExpression
	FunctionDeclarationExpression
	FunctionCallExpression
	GetterExpression
)

type ScriptMeta struct {
	Version int
	Bytes   []byte
}

type SourceTree struct {
	Digest       [32]byte
	AppVersion   int
	LibVersion   int
	HasBlockV2   bool
	Meta         ScriptMeta
	Declarations []Node
	Functions    []Node
	Verifier     Node
}

func (t *SourceTree) HasVerifier() bool {
	return t.Verifier.Kind != 0
}

func (t *SourceTree) IsDApp() bool {
	return t.AppVersion != scriptApplicationVersion
}

type Value struct {
	Long       int64
	Bytes      []byte
	String     string
	Boolean    bool
	Parameters []string
}

type Node struct {
	Kind                int
	V                   Value
	Siblings            []Node
	invocationParameter string
}

func (n Node) append(node Node) Node {
	n.Siblings = append(n.Siblings, node)
	return n
}

func NewLongLiteral(v int64) Node {
	return Node{
		Kind: LongLiteral,
		V:    Value{Long: v},
	}
}

func NewBytesLiteral(v []byte) Node {
	return Node{
		Kind: BytesLiteral,
		V:    Value{Bytes: v},
	}
}

func NewStringLiteral(v string) Node {
	return Node{
		Kind: StringLiteral,
		V:    Value{String: v},
	}
}

func NewBooleanLiteral(v bool) Node {
	return Node{
		Kind: BooleanLiteral,
		V:    Value{Boolean: v},
	}
}

func NewIfExpression(condition, consequence, alternative Node) Node {
	return Node{
		Kind:     IfExpression,
		Siblings: []Node{condition, consequence, alternative},
	}
}

func NewLetExpression(id string, expression Node) Node {
	return Node{
		Kind:     LetExpression,
		V:        Value{String: id},
		Siblings: []Node{expression},
	}
}

func NewReferenceExpression(id string) Node {
	return Node{
		Kind: ReferenceExpression,
		V:    Value{String: id},
	}
}

func NewFunctionDeclarationExpression(id string, arguments []string, body Node) Node {
	return Node{
		Kind:     FunctionDeclarationExpression,
		V:        Value{String: id, Parameters: arguments},
		Siblings: []Node{body},
	}
}

func NewFunctionCallExpression(id string, arguments []Node) Node {
	return Node{
		Kind:     FunctionCallExpression,
		V:        Value{String: id},
		Siblings: arguments,
	}
}

func NewGetterExpression(object Node, field string) Node {
	return Node{
		Kind:     GetterExpression,
		V:        Value{String: field},
		Siblings: []Node{object},
	}
}
