package fride

import (
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
)

func Compile(tree *Tree) (*Program, error) {
	c := &compiler{
		code:      make([]byte, 0),
		constants: make([]rideType, 0),
		functions: make(map[string]int),
		strings:   make(map[string]uint16),
		entry:     0,
	}
	err := c.compile(tree.Verifier)
	if err != nil {
		return nil, err
	}
	return &Program{
		LibVersion: tree.LibVersion,
		Code:       c.code,
		Constants:  c.constants,
		Functions:  c.functions,
		EntryPoint: c.entry,
	}, nil
}

type compiler struct {
	code      []byte
	constants []rideType
	functions map[string]int
	strings   map[string]uint16
	entry     int
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
	pos := len(c.code)
	c.code = append(c.code, op)
	c.code = append(c.code, data...)
	return pos
}

func (c *compiler) longNode(node *LongNode) error {
	v, err := c.makeConstant(rideLong(node.Value))
	if err != nil {
		return err
	}
	c.emit(OpPush, v...)
	return nil
}

func (c *compiler) bytesNode(node *BytesNode) error {
	v, err := c.makeConstant(rideBytes(node.Value))
	if err != nil {
		return err
	}
	c.emit(OpPush, v...)
	return nil
}

func (c *compiler) stringNode(node *StringNode) error {
	v, err := c.makeConstant(rideString(node.Value))
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
	p, err := c.makeConstant(rideString(node.Name))
	if err != nil {
		return err
	}
	c.emit(OpStore, p...)
	//TODO: rewrite resulting pos for laziness
	err = c.compile(node.Block)
	if err != nil {
		return err
	}
	return nil
}

func (c *compiler) referenceNode(node *ReferenceNode) error {
	p, err := c.makeConstant(rideString(node.Name))
	if err != nil {
		return err
	}
	c.emit(OpLoad, p...)
	return nil
}

func (c *compiler) functionDeclarationNode(node *FunctionDeclarationNode) error {
	pos := c.emit(OpBegin)
	for _, arg := range node.Arguments {
		p, err := c.makeConstant(rideString(arg))
		if err != nil {
			return err
		}
		c.emit(OpStore, p...)
	}
	err := c.compile(node.Body)
	if err != nil {
		return err
	}
	c.emit(OpEnd)
	c.functions[node.Name] = pos
	c.entry = len(c.code)
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
	p, err := c.makeConstant(rideString(node.Name))
	if err != nil {
		return err
	}
	c.emit(OpProperty, p...)
	return nil
}

func (c *compiler) makeConstant(v rideType) ([]byte, error) {
	sv, isString := v.(rideString)
	if isString {
		if pos, ok := c.strings[string(sv)]; ok {
			return encode(pos), nil
		}
	}
	c.constants = append(c.constants, v)
	if len(c.constants) > math.MaxUint16 {
		return nil, errors.New("max number of constants exceeded")
	}
	pos := uint16(len(c.constants) - 1)
	if isString {
		c.strings[string(sv)] = pos
	}
	return encode(pos), nil
}

func (c *compiler) makeCall(name string, count int) ([]byte, error) {
	c.constants = append(c.constants, rideCall{name, count})
	if len(c.constants) > math.MaxUint16 {
		return nil, errors.New("max number of constants exceeded")
	}
	pos := uint16(len(c.constants) - 1)
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

func (c *compiler) patchJump(pos int) {
	offset := len(c.code) - pos - 3
	b := encode(uint16(offset))
	c.code[pos+1] = b[0]
	c.code[pos+2] = b[1]
}
