package ride

//go:generate go run ./generate

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

func Compile(tree *ast.Tree) (RideScript, error) {
	fCheck, err := selectFunctionChecker(tree.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "compile")
	}
	cCheck, err := selectConstantsChecker(tree.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "compile")
	}
	c := &compiler{
		constants:     newRideConstants(),
		checkFunction: fCheck,
		checkConstant: cCheck,
		values:        make([]rideValue, 0),
		functions:     make([]*localFunction, 0),
		declarations:  make([]rideDeclaration, 0),
		patcher:       newPatcher(),
	}
	if tree.IsDApp() {
		return c.compileDAppScript(tree)
	}
	return c.compileSimpleScript(tree)
}

type compiler struct {
	constants     *rideConstants
	checkFunction func(string) (uint16, bool)
	checkConstant func(string) (uint16, bool)
	values        []rideValue
	functions     []*localFunction
	declarations  []rideDeclaration
	patcher       *patcher
	callable      *rideCallable
}

func (c *compiler) compileSimpleScript(tree *ast.Tree) (*SimpleScript, error) {
	bb := new(bytes.Buffer)
	err := c.compile(bb, tree.Verifier)
	if err != nil {
		return nil, err
	}
	bb.WriteByte(OpHalt)
	for _, d := range c.declarations {
		pos := bb.Len()
		c.patcher.setOrigin(d.buffer(), pos)
		bb.Write(patchedCode(d.buffer(), pos))
		bb.WriteByte(OpReturn)
		for _, ref := range d.references() {
			c.patcher.setPosition(ref, uint16(pos))
		}
	}
	code := bb.Bytes()
	patches, err := c.patcher.get()
	if err != nil {
		return nil, err
	}
	for pos, addr := range patches {
		binary.BigEndian.PutUint16(code[pos:], addr)
	}
	return &SimpleScript{
		LibVersion: tree.LibVersion,
		EntryPoint: 0,
		Code:       bb.Bytes(),
		Constants:  c.constants.items,
	}, nil
}

func (c *compiler) compileDAppScript(tree *ast.Tree) (*DAppScript, error) {
	// Compile global declarations.
	// Each declaration goes to the declaration stack with code compiled in its own buffer.
	functions := make(map[string]callable)
	bb := new(bytes.Buffer)
	for _, node := range tree.Declarations {
		err := c.compile(bb, node)
		if err != nil {
			return nil, err
		}
	}
	for _, n := range tree.Functions {
		fn, ok := n.(*ast.FunctionDeclarationNode)
		if !ok {
			return nil, errors.Errorf("invalid node type %T", n)
		}
		c.callable = &rideCallable{
			name:      fn.Name,
			parameter: fn.InvocationParameter,
		}
		err := c.compile(bb, fn)
		if err != nil {
			return nil, err
		}
	}
	if tree.HasVerifier() {
		v, ok := tree.Verifier.(*ast.FunctionDeclarationNode)
		if !ok {
			return nil, errors.Errorf("invalid node type for DApp's verifier '%T'", tree.Verifier)
		}
		c.callable = &rideCallable{
			name:      "", // Verifier has empty name
			parameter: v.InvocationParameter,
		}
		err := c.compile(bb, v)
		if err != nil {
			return nil, err
		}
		functions[""] = callable{
			entryPoint:    0,
			parameterName: v.InvocationParameter,
		}
	}
	// All declarations go here after verifier and public functions
	for _, d := range c.declarations {
		pos := bb.Len()
		c.patcher.setOrigin(d.buffer(), pos)
		bb.Write(patchedCode(d.buffer(), pos))
		if c := d.callable(); c != nil {
			bb.WriteByte(OpHalt)
			// Add reference to entry point
			functions[c.name] = callable{
				entryPoint:    pos,
				parameterName: c.parameter,
			}
		} else {
			bb.WriteByte(OpReturn)
		}
		for _, ref := range d.references() {
			c.patcher.setPosition(ref, uint16(pos))
		}
	}
	code := bb.Bytes()
	patches, err := c.patcher.get()
	if err != nil {
		return nil, err
	}
	for pos, addr := range patches {
		binary.BigEndian.PutUint16(code[pos:], addr)
	}
	return &DAppScript{
		LibVersion:  tree.LibVersion,
		Code:        bb.Bytes(),
		Constants:   c.constants.items,
		EntryPoints: functions,
	}, nil
}

func (c *compiler) compile(bb *bytes.Buffer, node ast.Node) error {
	switch n := node.(type) {
	case *ast.LongNode:
		return c.longNode(bb, n)
	case *ast.BytesNode:
		return c.bytesNode(bb, n)
	case *ast.StringNode:
		return c.stringNode(bb, n)
	case *ast.BooleanNode:
		return c.booleanNode(bb, n)
	case *ast.ConditionalNode:
		return c.conditionalNode(bb, n)
	case *ast.AssignmentNode:
		return c.assignmentNode(bb, n)
	case *ast.ReferenceNode:
		return c.referenceNode(bb, n)
	case *ast.FunctionDeclarationNode:
		return c.functionDeclarationNode(bb, n)
	case *ast.FunctionCallNode:
		return c.callNode(bb, n)
	case *ast.PropertyNode:
		return c.propertyNode(bb, n)
	default:
		return errors.Errorf("unexpected node type '%T'", node)
	}
}

func (c *compiler) longNode(bb *bytes.Buffer, node *ast.LongNode) error {
	cid, err := c.constants.put(rideInt(node.Value))
	if err != nil {
		return err
	}
	bb.WriteByte(OpPush)
	bb.Write(encode(cid))
	return nil
}

func (c *compiler) bytesNode(bb *bytes.Buffer, node *ast.BytesNode) error {
	cid, err := c.constants.put(rideBytes(node.Value))
	if err != nil {
		return err
	}
	bb.WriteByte(OpPush)
	bb.Write(encode(cid))
	return nil
}

func (c *compiler) stringNode(bb *bytes.Buffer, node *ast.StringNode) error {
	cid, err := c.constants.put(rideString(node.Value))
	if err != nil {
		return err
	}
	bb.WriteByte(OpPush)
	bb.Write(encode(cid))
	return nil
}

func (c *compiler) booleanNode(bb *bytes.Buffer, node *ast.BooleanNode) error {
	if node.Value {
		bb.WriteByte(OpTrue)
	} else {
		bb.WriteByte(OpFalse)
	}
	return nil
}

func (c *compiler) conditionalNode(bb *bytes.Buffer, node *ast.ConditionalNode) error {
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

func (c *compiler) assignmentNode(bb *bytes.Buffer, node *ast.AssignmentNode) error {
	err := c.pushGlobalValue(node.Name, node.Expression, c.callable)
	if err != nil {
		return err
	}
	if node.Block != nil {
		err = c.compile(bb, node.Block)
		if err != nil {
			return err
		}
		err = c.popValue()
		if err != nil {
			return err
		}
	} else {
		err = c.peekValue()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *compiler) referenceNode(bb *bytes.Buffer, node *ast.ReferenceNode) error {
	if id, ok := c.checkConstant(node.Name); ok {
		//Globally declared constant
		bb.WriteByte(OpGlobal)
		bb.Write(encode(id))
		return nil
	} else {
		v, err := c.lookupValue(node.Name)
		if err != nil {
			return err
		}
		switch value := v.(type) {
		case *localValue:
			bb.WriteByte(OpLoadLocal)
			bb.Write(encode(value.position))
			return nil
		case *globalValue:
			bb.WriteByte(OpLoad)
			value.refer(c.patcher.add(bb))
			bb.Write([]byte{0xff, 0xff})
			return nil
		default:
			return errors.Errorf("unexpected value type '%T'", value)
		}
	}
}

func (c *compiler) functionDeclarationNode(bb *bytes.Buffer, node *ast.FunctionDeclarationNode) error {
	var args []string
	if c.callable != nil { // DApp callable function
		args = make([]string, len(node.Arguments)+1)
		args[0] = c.callable.parameter
		copy(args[1:], node.Arguments)
	} else {
		args = node.Arguments
	}
	err := c.pushFunction(node.Name, args, node.Body, c.callable)
	if err != nil {
		return err
	}
	if node.Block != nil {
		err = c.compile(bb, node.Block)
		if err != nil {
			return err
		}
		err = c.popFunction()
		if err != nil {
			return err
		}
	} else {
		err = c.peekFunction()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *compiler) callNode(bb *bytes.Buffer, node *ast.FunctionCallNode) error {
	for _, arg := range node.Arguments {
		err := c.compile(bb, arg)
		if err != nil {
			return err
		}
	}
	cnt := encode(uint16(len(node.Arguments)))

	if id, ok := c.checkFunction(node.Function.Name()); ok {
		//External function
		bb.WriteByte(OpExternalCall)
		bb.Write(encode(id))
		bb.Write(cnt)
	} else {
		//Internal function
		decl, err := c.lookupFunction(node.Function.Name())
		if err != nil {
			return err
		}
		bb.WriteByte(OpCall)
		decl.call(c.patcher.add(bb))
		bb.Write([]byte{0xff, 0xff})
		bb.Write(cnt)
	}
	return nil
}

func (c *compiler) propertyNode(bb *bytes.Buffer, node *ast.PropertyNode) error {
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

func (c *compiler) pushGlobalValue(name string, node ast.Node, annex *rideCallable) error {
	d := newGlobalValue(name, annex)
	c.callable = nil
	c.values = append(c.values, d)
	return c.compile(d.bb, node)
}

func (c *compiler) popValue() error {
	l := len(c.values)
	if l == 0 {
		return errors.New("failed to pop value from empty stack")
	}
	var v rideValue
	v, c.values = c.values[l-1], c.values[:l-1]
	if d, ok := v.(rideDeclaration); ok {
		c.declarations = append(c.declarations, d)
	}
	return nil
}

func (c *compiler) peekValue() error {
	l := len(c.values)
	if l == 0 {
		return errors.New("failed to peek value from empty stack")
	}
	v := c.values[l-1]
	if d, ok := v.(rideDeclaration); ok {
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

func (c *compiler) pushFunction(name string, args []string, node ast.Node, annex *rideCallable) error {
	d := newLocalFunction(name, args, annex)
	c.functions = append(c.functions, d)
	for i, a := range args {
		c.values = append(c.values, newLocalValue(a, i))
	}
	c.callable = nil
	err := c.compile(d.bb, node)
	if err != nil {
		return err
	}
	for range args {
		err := c.popValue()
		if err != nil {
			return err
		}
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
	c.declarations = append(c.declarations, f)
	return nil
}

func (c *compiler) peekFunction() error {
	l := len(c.functions)
	if l == 0 {
		return errors.New("failed to peek function from empty stack")
	}
	f := c.functions[l-1]
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

type rideCallable struct {
	name      string
	parameter string
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
	buffer() *bytes.Buffer
	references() []int
	callable() *rideCallable
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
	annex  *rideCallable
	bb     *bytes.Buffer
	usages []int
	using  []*globalValue
}

func newGlobalValue(name string, annex *rideCallable) *globalValue {
	return &globalValue{
		name:   name,
		annex:  annex,
		bb:     new(bytes.Buffer),
		usages: make([]int, 0),
		using:  make([]*globalValue, 0),
	}
}

func (v *globalValue) buffer() *bytes.Buffer {
	return v.bb
}

func (v *globalValue) references() []int {
	return v.usages
}

func (v *globalValue) id() string {
	return v.name
}

func (v *globalValue) refer(patch int) {
	v.usages = append(v.usages, patch)
}

func (v *globalValue) callable() *rideCallable {
	return v.annex
}

type localFunction struct {
	name  string
	annex *rideCallable
	bb    *bytes.Buffer
	calls []int
	args  []string
}

func newLocalFunction(name string, args []string, annex *rideCallable) *localFunction {
	return &localFunction{
		name:  name,
		annex: annex,
		bb:    new(bytes.Buffer),
		calls: make([]int, 0),
		args:  args,
	}
}

func (f *localFunction) buffer() *bytes.Buffer {
	return f.bb
}

func (f *localFunction) references() []int {
	return f.calls
}

func (f *localFunction) call(pos int) {
	f.calls = append(f.calls, pos)
}

func (f *localFunction) callable() *rideCallable {
	return f.annex
}

type patch struct {
	id     int
	origin int
	pos    int
	addr   uint16
}

type patcher struct {
	count   int
	buffers map[*bytes.Buffer][]patch
}

func newPatcher() *patcher {
	return &patcher{
		count:   0,
		buffers: make(map[*bytes.Buffer][]patch),
	}
}

func (p *patcher) add(bb *bytes.Buffer) int {
	pt := patch{
		id:  p.count,
		pos: bb.Len(),
	}
	if bps, ok := p.buffers[bb]; ok {
		p.buffers[bb] = append(bps, pt)
	} else {
		p.buffers[bb] = []patch{pt}
	}
	p.count++
	return pt.id
}

func (p *patcher) setOrigin(bb *bytes.Buffer, origin int) {
	if ps, ok := p.buffers[bb]; ok {
		for i := range ps {
			ps[i].origin = origin
		}
	}
}

func (p *patcher) setPosition(id int, addr uint16) {
	for _, v := range p.buffers {
		for i := range v {
			if v[i].id == id {
				v[i].addr = addr
			}
		}
	}
}

func (p *patcher) get() (map[int]uint16, error) {
	r := make(map[int]uint16)
	for _, v := range p.buffers {
		for i := range v {
			abs := v[i].origin + v[i].pos
			if _, ok := r[abs]; ok {
				return nil, errors.Errorf("duplicate patch at position %d", abs)
			}
			r[abs] = v[i].addr
		}
	}
	return r, nil
}

func encode(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}

func patchedCode(bb *bytes.Buffer, origin int) []byte {
	bts := bb.Bytes()
	i := 0
	for i < bb.Len() {
		switch bts[i] {
		case OpJumpIfFalse, OpJump:
			addr := bts[i+1 : i+3]
			update(addr, origin)
			i += 3
		case OpPush, OpProperty, OpLoad, OpLoadLocal, OpGlobal:
			i += 3
		case OpCall, OpExternalCall:
			i += 5
		default:
			i++
		}
	}
	return bts
}

func update(b []byte, n int) {
	v := binary.BigEndian.Uint16(b)
	binary.BigEndian.PutUint16(b, uint16(int(v)+n))
}
