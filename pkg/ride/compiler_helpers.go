package ride

import "bytes"

type constid = uint16

type builder struct {
	w       *bytes.Buffer
	startAt uint16
}

func newBuilder() *builder {
	return &builder{
		w: new(bytes.Buffer),
	}
}

func (b *builder) writeStub(len int) (position uint16) {
	position = uint16(b.w.Len())
	for i := 0; i < len; i++ {
		b.w.WriteByte(0)
	}
	return position
}

func (b *builder) push(uint162 uint16) {
	b.w.WriteByte(OpPush)
	b.w.Write(encode(uint162))
}

func (b *builder) bool(v bool) {
	if v {
		b.w.WriteByte(OpTrue)
	} else {
		b.w.WriteByte(OpFalse)
	}
}

func (b *builder) bytes() []byte {
	return b.w.Bytes()
}

func (b *builder) ret() {
	b.w.WriteByte(OpReturn)
}

func (b *builder) jump(uint162 uint16) {
	b.w.WriteByte(OpJump)
	b.w.Write(encode(uint162))
}

func (b *builder) patch(at uint16, val []byte) {
	bts := b.w.Bytes()[at:]
	for i := range val {
		bts[i] = val[i]
	}
}

func (b *builder) len() uint16 {
	return uint16(b.w.Len())
}

func (b *builder) externalCall(id uint16, argc uint16) {
	b.w.WriteByte(OpExternalCall)
	b.w.Write(encode(id))
	b.w.Write(encode(argc))
}

// Call user defined function.
func (b *builder) call(id uint16, argc uint16) {
	b.w.WriteByte(OpCall)
	b.w.Write(encode(id))
}

func (b *builder) startPos() {
	b.startAt = uint16(b.w.Len())
}

func (b *builder) build() (uint16, []byte) {
	return b.startAt, b.w.Bytes()
}

func (b *builder) jpmIfFalse() {
	b.w.WriteByte(OpJumpIfFalse)
}

func (b *builder) writeByte(p byte) {
	b.w.WriteByte(p)
}

func (b *builder) write(i []byte) {
	b.w.Write(i)
}

//func (b *builder) fillContext(id constid) {
//	b.w.WriteByte(OpFillContext)
//	b.w.Write(encode(id))
//}

type constants struct {
	values []rideType
}

func newConstants() *constants {
	return &constants{}
}

func (a *constants) put(value rideType) uint16 {
	a.values = append(a.values, value)
	return uint16(len(a.values) - 1)
}

func (a *constants) constants() []rideType {
	return a.values
}

type references struct {
	prev *references
	refs map[string]uint16
}

func newReferences(prev *references) *references {
	return &references{
		prev: prev,
		refs: make(map[string]uint16),
	}
}

func (a *references) get(name string) (uint16, bool) {
	if a == nil {
		return 0, false
	}
	if offset, ok := a.refs[name]; ok {
		return offset, ok
	}
	return a.prev.get(name)
}

func (a *references) set(name string, offset uint16) {
	a.refs[name] = offset
}
