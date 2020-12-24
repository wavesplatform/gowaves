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
	b.w.WriteByte(OpRef)
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

func (b *builder) ref(uint162 uint16) {
	b.w.WriteByte(OpRef)
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

//func (b *builder) startPos() {
//	b.startAt = uint16(b.w.Len())
//}

func (b *builder) build() (uint16, []byte) {
	return 1, b.w.Bytes()
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

type point struct {
	position  uint16   `cbor:"0,keyasint"`
	value     rideType `cbor:"1,keyasint"`
	fn        uint16   `cbor:"2,keyasint"`
	constant  bool     `cbor:"3,keyasint"`
	debugInfo string   `cbor:"4,keyasint"`
}

type cell struct {
	values map[uniqueid]point
}

func newCell() *cell {
	return &cell{
		values: make(map[uniqueid]point),
	}
}

func (a *cell) set(u uniqueid, result rideType, fn uint16, position uint16, constant bool, debug string) {
	a.values[u] = point{
		position:  position,
		value:     result,
		fn:        fn,
		constant:  constant,
		debugInfo: debug,
	}
}

func (a *cell) get(u uniqueid) (point, bool) {
	rs, ok := a.values[u]
	return rs, ok
}

type uniqueid = uint16

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

func (a *references) get(name string) (uniqueid, bool) {
	if a == nil {
		return 0, false
	}
	if offset, ok := a.refs[name]; ok {
		return offset, ok
	}
	return a.prev.get(name)
}

func (a *references) set(name string, uniq uniqueid) {
	a.refs[name] = uniq
}

type predefFunc struct {
	id uint16
	f  rideFunction
}

type predef struct {
	prev *predef
	m    map[string]predefFunc
}

func newPredef(prev *predef) *predef {
	return &predef{
		prev: prev,
		m:    make(map[string]predefFunc),
	}
}

func newPredefWithValue(prev *predef, name string, id uint16, f rideFunction) *predef {
	p := newPredef(prev)
	p.set(name, id, f)
	return p
}

func (a *predef) set(name string, id uint16, f rideFunction) {
	a.m[name] = predefFunc{
		id: id,
		f:  f,
	}
}

func (a *predef) get(name string) (predefFunc, bool) {
	if a == nil {
		return predefFunc{}, false
	}
	rs, ok := a.m[name]
	if ok {
		return rs, ok
	}
	return a.prev.get(name)
}

func (a *predef) getn(id int) rideFunction {
	if a == nil {
		return nil
	}
	for _, v := range a.m {
		if v.id == uint16(id) {
			return v.f
		}
	}
	return a.prev.getn(id)
}

func reverse(f []Deferred) []Deferred {
	out := make([]Deferred, 0, len(f))
	for i := len(f) - 1; i >= 0; i-- {
		out = append(out, f[i])
	}
	return out
}

type Deferred interface {
	Write
	Clean
}

//type deferred struct {
//	write func()
//	clean func()
//}

//func (a deferred) Write(_ params, _ []byte) {
//	if a.write != nil {
//		a.write()
//	}
//}
//
//func (a deferred) Clean() {
//	if a.clean != nil {
//		a.clean()
//	}
//}

type constantDeferred struct {
	n uniqueid
}

func (a constantDeferred) Write(p params, _ []byte) {
	p.b.writeByte(OpRef)
	p.b.write(encode(a.n))
}

func (a constantDeferred) Clean() {
}

//func NewDeferred(writeFunc func(), cleanFunc func()) Deferred {
//	return deferred{
//		write: writeFunc,
//		clean: cleanFunc,
//	}
//}

func NewConstantDeferred(n uniqueid) constantDeferred {
	return constantDeferred{n: n}
}

//func writeDeferred(params params, d []Deferred) {
//	panic("writeDeferred 1")
//	if len(d) != 1 {
//		panic("writeDeferred len != 1")
//	}
//	d2 := reverse(d)
//
//	d2[0].Write(params)
//
//	for _, v := range d2 {
//		v.Clean()
//	}
//
//	params.b.ret()
//	for _, v := range d2[1:] {
//		v.Write(params)
//	}
//}

func isConstant(deferred Deferred) (uniqueid, bool) {
	v, ok := deferred.(constantDeferred)
	return v.n, ok
}
