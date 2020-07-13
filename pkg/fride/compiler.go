package fride

import (
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
)

func Compile(tree *Tree) (*Program, error) {
	c := &compiler{
		code:      make([]byte, 0, 256),
		longs:     make([]int64, 0),
		bytes:     make([][]byte, 0),
		strings:   make([]string, 0),
		calls:     make([]call, 0),
		functions: make(map[string]function),
	}
	err := c.compile(tree.Verifier)
	if err != nil {
		return nil, err
	}
	return &Program{
		Code:            c.code,
		LongConstants:   c.longs,
		ByteConstants:   c.bytes,
		StringConstants: c.strings,
	}, nil
}

type compiler struct {
	code      []byte
	longs     []int64
	bytes     [][]byte
	strings   []string
	calls     []call
	functions map[string]function
}

func (c *compiler) compile(node Node) error {
	switch n := node.(type) {
	case *LongNode:
		return c.longNode(n)
	case *BytesNode:
		return c.bytesNode(n)
	case *StringNode:
		return c.stringNode(n)
	case *BooleanNode:
		return c.booleanNode(n)
	case *ConditionalNode:
		return c.conditionalNode(n)
	case *AssignmentNode:
		return c.assignmentNode(n)
	case *ReferenceNode:
		return c.referenceNode(n)
	case *FunctionDeclarationNode:
		return c.functionDeclarationNode(n)
	case *FunctionCallNode:
		return c.callNode(n)
	case *PropertyNode:
		return c.propertyNode(n)
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

func (c *compiler) longNode(node *LongNode) error {
	v, err := c.makeLongConstant(node.Value)
	if err != nil {
		return err
	}
	c.emit(OpPush, v...)
	return nil
}

func (c *compiler) bytesNode(node *BytesNode) error {
	v, err := c.makeBytesConstant(node.Value)
	if err != nil {
		return err
	}
	c.emit(OpPush, v...)
	return nil
}

func (c *compiler) stringNode(node *StringNode) error {
	v, err := c.makeStringConstant(node.Value)
	if err != nil {
		return err
	}
	c.emit(OpPush, v...)
	return nil
}

func (c *compiler) booleanNode(node *BooleanNode) error {
	if node.Value {
		c.emit(OpTrue)
	} else {
		c.emit(OpFalse)
	}
	return nil
}

func (c *compiler) conditionalNode(node *ConditionalNode) error {
	err := c.compile(node.Condition)
	if err != nil {
		return err
	}
	otherwise := c.emit(OpJumpIfFalse, c.placeholder()...)

	c.emit(OpPop)
	err = c.compile(node.TrueExpression)
	if err != nil {
		return err
	}
	end := c.emit(OpJump, c.placeholder()...)

	c.patchJump(otherwise)
	c.emit(OpPop)
	err = c.compile(node.FalseExpression)
	if err != nil {
		return err
	}

	c.patchJump(end)
	return nil
}

func (c *compiler) assignmentNode(node *AssignmentNode) error {
	err := c.compile(node.Expression)
	if err != nil {
		return err
	}
	p, err := c.makeStringConstant(node.Name)
	if err != nil {
		return err
	}
	c.emit(OpStore, p...)

	return c.compile(node.Block)
}

func (c *compiler) referenceNode(node *ReferenceNode) error {
	p, err := c.makeStringConstant(node.Name)
	if err != nil {
		return err
	}
	c.emit(OpLoad, p...)
	return nil
}

func (c *compiler) functionDeclarationNode(node *FunctionDeclarationNode) error {
	err := c.compile(node.Body)
	if err != nil {
		return err
	}
	c.emit(OpReturn)
	return c.compile(node.Block)
}

func (c *compiler) callNode(node *FunctionCallNode) error {
	for _, arg := range node.Arguments {
		err := c.compile(arg)
		if err != nil {
			return err
		}
	}
	call, err := c.makeCall(node.Name, len(node.Arguments))
	if err != nil {
		return err
	}

	c.emit(OpCall, call...)
	return nil
}

func (c *compiler) propertyNode(node *PropertyNode) error {
	err := c.compile(node.Object)
	if err != nil {
		return err
	}
	p, err := c.makeStringConstant(node.Name)
	if err != nil {
		return err
	}
	c.emit(OpProperty, p...)
	return nil
}

func (c *compiler) makeLongConstant(v int64) ([]byte, error) {
	c.longs = append(c.longs, v)
	if len(c.longs) > math.MaxUint16 {
		return nil, errors.New("max number of longNode constants exceeded")
	}
	pos := uint16(len(c.longs) - 1)
	return encode(pos), nil
}

func (c *compiler) makeBytesConstant(v []byte) ([]byte, error) {
	c.bytes = append(c.bytes, v)
	if len(c.bytes) > math.MaxUint16 {
		return nil, errors.New("max number of bytes constants exceeded")
	}
	pos := uint16(len(c.bytes) - 1)
	return encode(pos), nil
}

func (c *compiler) makeStringConstant(v string) ([]byte, error) {
	c.strings = append(c.strings, v)
	if len(c.strings) > math.MaxUint16 {
		return nil, errors.New("max number of string constants exceeded")
	}
	pos := uint16(len(c.strings) - 1)
	return encode(pos), nil
}

func (c *compiler) makeCall(name string, count int) ([]byte, error) {
	c.calls = append(c.calls, call{name, count})
	if len(c.calls) > math.MaxUint16 {
		return nil, errors.New("max number of function calls exceeded")
	}
	pos := uint16(len(c.calls) - 1)
	return encode(pos), nil
}

func encode(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}

func (c *compiler) placeholder() []byte {
	return []byte{0xFF, 0xFF}
}

func (c *compiler) patchJump(placeholder int) {
	offset := len(c.code) - 2 - placeholder
	b := encode(uint16(offset))
	c.code[placeholder] = b[0]
	c.code[placeholder+1] = b[1]
}
