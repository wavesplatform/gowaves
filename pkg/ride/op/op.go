package op

import (
	"encoding/binary"
)

const (
	NoOp uint8 = iota
	Label
	Call           // function call 2
	JmpRef         // 3
	Ret            // 4
	StackPushL     // 5
	StackPushS     // 6
	StackPushTrue  // 7
	StackPushFalse // 8
	StackPushBytes // 9
	JumpIfNot      // 10 (if else)
	Jmp            // 11

)

type OpCodeBuilderImpl struct {
	body      []byte
	lastShift int32
}

func NewOpCodeBuilder() *OpCodeBuilderImpl {
	return &OpCodeBuilderImpl{
		lastShift: -1,
	}
}

func (a *OpCodeBuilderImpl) Pos() int32 {
	return int32(len(a.body))
}

func (a *OpCodeBuilderImpl) Code() []byte {
	return a.body
}

func (a *OpCodeBuilderImpl) Label(name []byte) *OpCodeBuilderImpl {
	a.add(Label)
	l := uint16(len(name))
	size := make([]byte, 2)
	binary.BigEndian.PutUint16(size, l)
	a.add(size...)
	a.add(name...)
	return a
}

func (a *OpCodeBuilderImpl) LabelS(name string) *OpCodeBuilderImpl {
	return a.Label([]byte(name))
}

func (a *OpCodeBuilderImpl) str(bts []byte) {
	l := uint16(len(bts))
	size := make([]byte, 2)
	binary.BigEndian.PutUint16(size, l)
	a.add(size...)
	a.add(bts...)
}

func (a *OpCodeBuilderImpl) Add(bts []byte) {
	a.add(bts...)
}

func (a *OpCodeBuilderImpl) JmpRef(name []byte) *OpCodeBuilderImpl {
	a.add(JmpRef)
	a.str(name)
	return a
}

func (a *OpCodeBuilderImpl) JmpRefS(name string) *OpCodeBuilderImpl {
	return a.JmpRef([]byte(name))
}

func (a *OpCodeBuilderImpl) StackPushL(value int64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(value))
	a.add(StackPushL)
	a.add(buf...)
}

func (a *OpCodeBuilderImpl) JumpIfNot() *OpCodeBuilderImpl {
	return a.add(JumpIfNot)
}

func (a *OpCodeBuilderImpl) StackPushB(b bool) {
	if b {
		a.StackPushTrue()
	} else {
		a.StackPushFalse()
	}
}

func (a *OpCodeBuilderImpl) StackPushTrue() {
	a.add(StackPushTrue)
}

func (a *OpCodeBuilderImpl) StackPushFalse() {
	a.add(StackPushFalse)
}

func (a *OpCodeBuilderImpl) StackPushS(value /*string*/ []byte) {
	a.add(StackPushS)
	a.str(value)
}

func (a *OpCodeBuilderImpl) StackPushBytes(value []byte) *OpCodeBuilderImpl {
	a.add(StackPushBytes)
	a.str(value)
	return a
}

func (a *OpCodeBuilderImpl) add(b ...byte) *OpCodeBuilderImpl {
	a.body = append(a.body, b...)
	return a
}

func (a *OpCodeBuilderImpl) Ret() {
	a.add(Ret)
}

func (a *OpCodeBuilderImpl) Call(name []byte) *OpCodeBuilderImpl {
	a.add(Call)
	a.str(name)
	return a
}

func (a *OpCodeBuilderImpl) CallS(name string) *OpCodeBuilderImpl {
	return a.Call([]byte(name))
}

func (a *OpCodeBuilderImpl) I32(i int32) {
	bts := make([]byte, 4)
	binary.BigEndian.PutUint32(bts, uint32(i))
	a.add(bts...)
}

func (a *OpCodeBuilderImpl) ShiftAt(at int32, value int32) {
	bts := make([]byte, 4)
	binary.BigEndian.PutUint32(bts, uint32(value))
	for i, v := range bts {
		a.body[at+int32(i)] = v
	}
}

func (a *OpCodeBuilderImpl) RememberShift() {
	a.lastShift = int32(len(a.body))
}

func (a *OpCodeBuilderImpl) ApplyShift() {
	//bts := make([]byte, 4)
	if a.lastShift == -1 {
		panic("no previous shift")
	}
	//binary.BigEndian.PutUint32(bts, uint32(a.lastShift))
	a.ShiftAt(a.lastShift, int32(len(a.body)))
}

func (a *OpCodeBuilderImpl) Jmp() *OpCodeBuilderImpl {
	return a.add(Jmp)
}
