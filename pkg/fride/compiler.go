package fride

//go:generate go run ./generate

import (
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
)

func Compile(tree *Tree) (*Program, error) {
	ec, err := externalChecker(tree.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "compile")
	}
	c := &compiler{
		code:          make([]byte, 0),
		constants:     newRideConstants(),
		checkExternal: ec,
	}
	err = c.compile(tree.Verifier)
	if err != nil {
		return nil, err
	}
	c.emitNone(OpReturn)
	return &Program{
		LibVersion: tree.LibVersion,
		Code:       c.code,
		Constants:  c.constants.items,
	}, nil
}

type compiler struct {
	code          []byte
	constants     *rideConstants
	checkExternal func(string) bool
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

func (c *compiler) emitNone(op byte) int {
	pos := len(c.code)
	c.code = append(c.code, op)
	return pos
}

func (c *compiler) emitByte(op, data byte) int {
	pos := len(c.code)
	c.code = append(c.code, []byte{op, data}...)
	return pos
}

func (c *compiler) emitByteByte(op, data1, data2 byte) int {
	pos := len(c.code)
	c.code = append(c.code, []byte{op, data1, data2}...)
	return pos
}

func (c *compiler) emitUint16Placeholder(op byte) int {
	pos := len(c.code)
	c.code = append(c.code, []byte{op, 0xff, 0xff}...)
	return pos + 1
}

func (c *compiler) emitUint16(op byte, n uint16) int {
	pos := len(c.code)
	bts := []byte{op, 0, 0}
	binary.BigEndian.PutUint16(bts[1:], n)
	c.code = append(c.code, bts...)
	return pos
}

func (c *compiler) emit5Placeholder(op byte, n uint16) int {
	l := len(c.code)
	bts := []byte{op, 0, 0, 0xff, 0xff}
	binary.BigEndian.PutUint16(bts[1:], n)
	c.code = append(c.code, bts...)
	return l + 3
}

func (c *compiler) longNode(node *LongNode) error {
	n, err := c.constants.put(rideInt(node.Value))
	if err != nil {
		return err
	}
	c.emitUint16(OpPush, n)
	return nil
}

func (c *compiler) bytesNode(node *BytesNode) error {
	n, err := c.constants.put(rideBytes(node.Value))
	if err != nil {
		return err
	}
	c.emitUint16(OpPush, n)
	return nil
}

func (c *compiler) stringNode(node *StringNode) error {
	n, err := c.constants.put(rideString(node.Value))
	if err != nil {
		return err
	}
	c.emitUint16(OpPush, n)
	return nil
}

func (c *compiler) booleanNode(node *BooleanNode) error {
	if node.Value {
		c.emitNone(OpTrue)
	} else {
		c.emitNone(OpFalse)
	}
	return nil
}

func (c *compiler) conditionalNode(node *ConditionalNode) error {
	err := c.compile(node.Condition)
	if err != nil {
		return err
	}
	otherwise := c.emitUint16Placeholder(OpJumpIfFalse)

	c.emitNone(OpPop)
	err = c.compile(node.TrueExpression)
	if err != nil {
		return err
	}
	end := c.emitUint16Placeholder(OpJump)

	c.patchJump(otherwise)
	c.emitNone(OpPop)
	err = c.compile(node.FalseExpression)
	if err != nil {
		return err
	}

	c.patchJump(end)
	return nil
}

func (c *compiler) assignmentNode(node *AssignmentNode) error {
	n, err := c.constants.put(rideString(node.Name))
	_ = c.emit5Placeholder(OpRecord, n)
	err = c.compile(node.Expression)
	if err != nil {
		return err
	}
	c.emitNone(OpReturn)
	//TODO: patch jump with block position
	return c.compile(node.Block)
}

func (c *compiler) referenceNode(node *ReferenceNode) error {
	n, err := c.constants.put(rideString(node.Name))
	if err != nil {
		return err
	}
	c.emitUint16(OpLoad, n)
	return nil
}

func (c *compiler) functionDeclarationNode(node *FunctionDeclarationNode) error {
	//pos := c.emit(OpBegin)
	for _, arg := range node.Arguments {
		n, err := c.constants.put(rideString(arg))
		if err != nil {
			return err
		}
		c.emitUint16(OpStore, n)
	}
	err := c.compile(node.Body)
	if err != nil {
		return err
	}
	//c.emit(OpEnd)
	return c.compile(node.Block)
}

func (c *compiler) callNode(node *FunctionCallNode) error {
	// Put call parameters on stack
	for _, arg := range node.Arguments {
		err := c.compile(arg)
		if err != nil {
			return err
		}
	}
	id, ok := functionByName(node.Name)
	if ok {
		//External function
		cnt := byte(len(node.Arguments))
		c.emitByteByte(OpExternalCall, id, cnt)
	} else {
		//Internal function
		n, err := c.constants.put(rideString(node.Name))
		if err != nil {
			return err
		}
		c.emitUint16(OpCall, n)
	}
	return nil
}

func (c *compiler) propertyNode(node *PropertyNode) error {
	err := c.compile(node.Object)
	if err != nil {
		return err
	}
	n, err := c.constants.put(rideString(node.Name))
	if err != nil {
		return err
	}
	c.emitUint16(OpProperty, n)
	return nil
}

func encode(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}

func (c *compiler) patchJump(pos int) {
	offset := len(c.code) - pos - 3
	b := encode(uint16(offset))
	c.code[pos+1] = b[0]
	c.code[pos+2] = b[1]
}

type rideConstants struct {
	items   []rideType
	strings map[string]uint16
}

func newRideConstants() *rideConstants {
	return &rideConstants{
		items:   make([]rideType, 0, 4),
		strings: make(map[string]uint16, 4),
	}
}

func (c *rideConstants) put(value rideType) (uint16, error) {
	switch v := value.(type) {
	case rideString:
		s := string(v)
		if pos, ok := c.strings[s]; ok {
			return pos, nil
		}
		pos, err := c.append(value)
		if err != nil {
			return 0, err
		}
		c.strings[s] = pos
		return pos, nil
	default:
		pos, err := c.append(value)
		if err != nil {
			return 0, err
		}
		return pos, nil
	}
}

func (c *rideConstants) append(value rideType) (uint16, error) {
	if len(c.items) >= math.MaxUint16 {
		return 0, errors.New("max number of constants reached")
	}
	c.items = append(c.items, value)
	return uint16(len(c.items) - 1), nil
}
