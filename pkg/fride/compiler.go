package fride

import "github.com/pkg/errors"

func Compile(tree *Tree) (*Program, error) {
	c := &compiler{
		code: make([]byte, 0, 256),
	}
	err := c.compile(tree)
	if err != nil {
		return nil, err
	}
}

type compiler struct {
	code []byte
}

func (c *compiler) compile(node Node) error {
	switch n := node.(type) {
	case *LongNode:
		return c.long(n)
	case *BytesNode:
		return c.bytes(n)
	case *StringNode:
		return c.string(n)
	case *BooleanNode:
		return c.boolean(n)
	case *ConditionalNode:
		return c.conditional(n)
	case *AssignmentNode:
		return c.assignment(n)
	case *ReferenceNode:
		return c.reference(n)
	case *FunctionDeclarationNode:
		return c.declaration(n)
	case *FunctionCallNode:
		return c.call(n)
	case *PropertyNode:
		return c.property(n)
	default:
		return errors.Errorf("unexpected node type '%T'", node)
	}
}

func (c *compiler) emit(op byte, data ...byte) int {
	c.code = append(c.code, op)
	current := len(c.code)
	c.code = append(c.code, data...)
	return current
}

func (c *compiler) long(node *LongNode) error {

}

func (c *compiler) bytes(node *BytesNode) error {

}

func (c *compiler) string(node *StringNode) error {

}

func (c *compiler) boolean(node *BooleanNode) error {

}

func (c *compiler) conditional(node *ConditionalNode) error {

}

func (c *compiler) assignment(node *AssignmentNode) error {

}

func (c *compiler) reference(node *ReferenceNode) error {

}

func (c *compiler) declaration(node *FunctionDeclarationNode) error {

}

func (c *compiler) call(node *FunctionCallNode) error {

}

func (c *compiler) property(node *PropertyNode) error {

}
