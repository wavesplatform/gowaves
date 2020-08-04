package fride

//go:generate go run ./generate

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
)

func Compile(tree *Tree) (*Program, error) {
	check, err := selectFunctionChecker(tree.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "compile")
	}
	c := &compiler{
		constants:     newRideConstants(),
		checkFunction: check,
		values:        make([]rideValue, 0),
		functions:     make([]*localFunction, 0),
		declarations:  make([]rideDeclaration, 0),
	}
	bb := new(bytes.Buffer)
	err = c.compile(bb, tree.Verifier)
	if err != nil {
		return nil, err
	}
	bb.WriteByte(OpReturn)
	patches := make(map[int]uint16)
	for _, d := range c.declarations {
		pos := bb.Len()
		bb.Write(d.code())
		bb.WriteByte(OpReturn)
		for _, ref := range d.references() {
			patches[ref] = uint16(pos)
		}
	}
	code := bb.Bytes()
	for pos, addr := range patches {
		binary.BigEndian.PutUint16(code[pos:], addr)
	}
	return &Program{
		LibVersion: tree.LibVersion,
		EntryPoint: 0,
		Code:       bb.Bytes(),
		Constants:  c.constants.items,
	}, nil
}

type compiler struct {
	constants     *rideConstants
	checkFunction func(string) (byte, bool)
	values        []rideValue
	functions     []*localFunction
	declarations  []rideDeclaration
}

func (c *compiler) compile(bb *bytes.Buffer, node Node) error {
	switch n := node.(type) {
	case *LongNode:
		return c.longNode(bb, n)
	case *BytesNode:
		return c.bytesNode(bb, n)
	case *StringNode:
		return c.stringNode(bb, n)
	case *BooleanNode:
		return c.booleanNode(bb, n)
	case *ConditionalNode:
		return c.conditionalNode(bb, n)
	case *AssignmentNode:
		return c.assignmentNode(bb, n)
	case *ReferenceNode:
		return c.referenceNode(bb, n)
	case *FunctionDeclarationNode:
		return c.functionDeclarationNode(bb, n)
	case *FunctionCallNode:
		return c.callNode(bb, n)
	case *PropertyNode:
		return c.propertyNode(bb, n)
	default:
		return errors.Errorf("unexpected node type '%T'", node)
	}
}

func (c *compiler) longNode(bb *bytes.Buffer, node *LongNode) error {
	cid, err := c.constants.put(rideInt(node.Value))
	if err != nil {
		return err
	}
	bb.WriteByte(OpPush)
	bb.Write(encode(cid))
	return nil
}

func (c *compiler) bytesNode(bb *bytes.Buffer, node *BytesNode) error {
	cid, err := c.constants.put(rideBytes(node.Value))
	if err != nil {
		return err
	}
	bb.WriteByte(OpPush)
	bb.Write(encode(cid))
	return nil
}

func (c *compiler) stringNode(bb *bytes.Buffer, node *StringNode) error {
	cid, err := c.constants.put(rideString(node.Value))
	if err != nil {
		return err
	}
	bb.WriteByte(OpPush)
	bb.Write(encode(cid))
	return nil
}

func (c *compiler) booleanNode(bb *bytes.Buffer, node *BooleanNode) error {
	if node.Value {
		bb.WriteByte(OpTrue)
	} else {
		bb.WriteByte(OpFalse)
	}
	return nil
}

func (c *compiler) conditionalNode(bb *bytes.Buffer, node *ConditionalNode) error {
	err := c.compile(bb, node.Condition)
	if err != nil {
		return err
	}
	bb.WriteByte(OpJumpIfFalse)
	otherwise := bb.Len()
	bb.Write([]byte{0xff, 0xff})

	// Truthy branch
	bb.WriteByte(OpPop) // Remove condition result from stack

	err = c.compile(bb, node.TrueExpression)
	if err != nil {
		return err
	}

	bb.WriteByte(OpJump)
	end := bb.Len()
	bb.Write([]byte{0xff, 0xff})

	// Patch jump to alternative branch
	code := bb.Bytes()
	binary.BigEndian.PutUint16(code[otherwise:], uint16(bb.Len()))

	// Alternative branch
	bb.WriteByte(OpPop) // Remove condition result from stack
	err = c.compile(bb, node.FalseExpression)
	if err != nil {
		return err
	}

	// Patch jump to the end of alternative branch
	code = bb.Bytes()
	binary.BigEndian.PutUint16(code[end:], uint16(bb.Len()))

	return nil
}

func (c *compiler) assignmentNode(bb *bytes.Buffer, node *AssignmentNode) error {
	err := c.pushGlobalValue(node.Name, node.Expression)
	if err != nil {
		return err
	}

	err = c.compile(bb, node.Block)
	if err != nil {
		return err
	}
	err = c.popValue()
	if err != nil {
		return err
	}

	return nil
}

func (c *compiler) referenceNode(bb *bytes.Buffer, node *ReferenceNode) error {
	v, err := c.lookupValue(node.Name)
	if err != nil {
		return err
	}
	switch tv := v.(type) {
	case *localValue:
		bb.WriteByte(OpLoadLocal)
		bb.Write(encode(tv.position))
		return nil
	case *globalValue:
		bb.WriteByte(OpLoad)
		tv.refer(bb.Len())
		bb.Write([]byte{0xff, 0xff})
		return nil
	default:
		return errors.Errorf("unexpected value type '%T'", tv)
	}
}

func (c *compiler) functionDeclarationNode(bb *bytes.Buffer, node *FunctionDeclarationNode) error {
	err := c.pushFunction(node.Name, node.Arguments, node.Body)
	if err != nil {
		return err
	}
	err = c.compile(bb, node.Block)
	if err != nil {
		return err
	}
	err = c.popFunction()
	if err != nil {
		return err
	}
	return nil
}

func (c *compiler) callNode(bb *bytes.Buffer, node *FunctionCallNode) error {
	for _, arg := range node.Arguments {
		err := c.compile(bb, arg)
		if err != nil {
			return err
		}
	}
	id, ok := c.checkFunction(node.Name)
	if ok {
		//External function
		cnt := byte(len(node.Arguments))
		bb.WriteByte(OpExternalCall)
		bb.WriteByte(id)
		bb.WriteByte(cnt)
	} else {
		//Internal function
		decl, err := c.lookupFunction(node.Name)
		if err != nil {
			return err
		}
		bb.WriteByte(OpCall)
		decl.call(bb.Len())
		bb.Write([]byte{0xff, 0xff})
	}
	return nil
}

func (c *compiler) propertyNode(bb *bytes.Buffer, node *PropertyNode) error {
	err := c.compile(bb, node.Object)
	if err != nil {
		return err
	}
	id, err := c.constants.put(rideString(node.Name))
	if err != nil {
		return err
	}
	bb.WriteByte(OpProperty)
	bb.Write(encode(id))
	return nil
}

func (c *compiler) pushGlobalValue(name string, node Node) error {
	d := newGlobalValue(name)
	c.values = append(c.values, d)
	err := c.compile(d.bb, node)
	if err != nil {
		return err
	}
	return nil
}

func (c *compiler) popValue() error {
	l := len(c.values)
	if l == 0 {
		return errors.New("failed to pop value from empty stack")
	}
	var v rideValue
	v, c.values = c.values[l-1], c.values[:l-1]
	if d, ok := v.(rideDeclaration); ok {
		//TODO: originally was `append([]rideDeclaration{d}, c.declarations...)` to store the order. Argh!
		c.declarations = append(c.declarations, d)
	}
	return nil
}

func (c *compiler) lookupValue(name string) (rideValue, error) {
	for i := len(c.values) - 1; i >= 0; i-- {
		if c.values[i].id() == name {
			return c.values[i], nil
		}
	}
	return nil, errors.Errorf("value '%s' is not declared", name)
}

func (c *compiler) pushFunction(name string, args []string, node Node) error {
	d := newLocalFunction(name, args)
	c.functions = append(c.functions, d)
	for i, a := range args {
		c.values = append(c.values, newLocalValue(a, i))
	}
	err := c.compile(d.bb, node)
	if err != nil {
		return err
	}
	return nil
}

func (c *compiler) popFunction() error {
	l := len(c.functions)
	if l == 0 {
		return errors.New("failed to pop function from empty stack")
	}
	var f *localFunction
	f, c.functions = c.functions[l-1], c.functions[:l-1]
	for range f.args {
		err := c.popValue()
		if err != nil {
			return err
		}
	}
	c.declarations = append(c.declarations, f)
	return nil
}

func (c *compiler) lookupFunction(name string) (*localFunction, error) {
	for i := len(c.functions) - 1; i >= 0; i-- {
		if c.functions[i].name == name {
			return c.functions[i], nil
		}
	}
	return nil, errors.Errorf("function '%s' is not declared", name)
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

type rideDeclaration interface {
	code() []byte
	references() []int
}

type rideValue interface {
	id() string
}

type localValue struct {
	name     string
	position uint16
}

func newLocalValue(name string, pos int) *localValue {
	return &localValue{
		name:     name,
		position: uint16(pos),
	}
}

func (v *localValue) id() string {
	return v.name
}

type globalValue struct {
	name   string
	bb     *bytes.Buffer
	usages []int
	using  []*globalValue
}

func newGlobalValue(name string) *globalValue {
	return &globalValue{
		name:   name,
		bb:     new(bytes.Buffer),
		usages: make([]int, 0),
		using:  make([]*globalValue, 0),
	}
}

func (v *globalValue) code() []byte {
	return v.bb.Bytes()
}

func (v *globalValue) references() []int {
	return v.usages
}

func (v *globalValue) id() string {
	return v.name
}

func (v *globalValue) refer(pos int) {
	v.usages = append(v.usages, pos)
}

type localFunction struct {
	name  string
	bb    *bytes.Buffer
	calls []int
	args  []string
}

func newLocalFunction(name string, args []string) *localFunction {
	return &localFunction{
		name:  name,
		bb:    new(bytes.Buffer),
		calls: make([]int, 0),
		args:  args,
	}
}

func (f *localFunction) code() []byte {
	return f.bb.Bytes()
}

func (f *localFunction) references() []int {
	return f.calls
}

func (f *localFunction) call(pos int) {
	f.calls = append(f.calls, pos)
}

func encode(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}
