package ride

import "bytes"

type constid = uint16

type Refs map[uint16]point

type Entrypoint struct {
	name string
	at   uint16
	argn uint16
}

func (a Entrypoint) Serialize(s Serializer) error {
	err := s.String(rideString(a.name))
	if err != nil {
		return err
	}
	s.Uint16(a.at)
	s.Uint16(a.argn)
	return nil
}

func deserializeEntrypoint(d *Deserializer) (Entrypoint, error) {
	var err error
	a := Entrypoint{}
	a.name, err = d.String()
	if err != nil {
		return a, err
	}
	a.at, err = d.Uint16()
	if err != nil {
		return a, err
	}
	a.argn, err = d.Uint16()
	if err != nil {
		return a, err
	}
	return a, nil
}

type builder struct {
	w           *bytes.Buffer
	entrypoints map[string]Entrypoint
}

func newBuilder() *builder {
	return &builder{
		w:           new(bytes.Buffer),
		entrypoints: make(map[string]Entrypoint),
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

func (b *builder) setStart(name string, argn int) {
	b.entrypoints[name] = Entrypoint{
		name: name,
		at:   b.len(),
		argn: uint16(argn),
	}
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

func patchBuffer(b *bytes.Buffer, at uint16, val []byte) {
	bts := b.Bytes()[at:]
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

func (b *builder) build() (map[string]Entrypoint, []byte) {
	return b.entrypoints, b.w.Bytes()
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
	position  uint16
	value     rideType
	fn        uint16
	debugInfo string
}

func (a point) Serialize(s Serializer) error {
	s.Uint16(a.position)
	if a.value != nil {
		err := a.value.Serialize(s)
		if err != nil {
			return err
		}
	} else {
		err := s.RideNoValue()
		if err != nil {
			return err
		}
	}

	s.Uint16(a.fn)
	return nil
}

func (a point) constant() bool {
	return a.position == 0 && a.fn == 0
}

func deserializePoint(d *Deserializer) (point, error) {
	var a point
	var err error
	a.position, err = d.Uint16()
	if err != nil {
		return a, err
	}

	a.value, err = d.RideValue()
	if err != nil {
		return a, err
	}
	a.fn, err = d.Uint16()
	if err != nil {
		return a, err
	}
	return a, nil
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

func (a *references) pop() *references {
	if a.prev != nil {
		return a.prev
	}
	panic("no previous refs")
}

type predefFunc struct {
	name string
	f    rideFunction
}

type pfunc struct {
	name string
	f    rideFunction
	id   uint16
}

type predef struct {
	prev *predef
	m    map[string]pfunc
}

func newPredef(prev *predef) *predef {
	return &predef{
		prev: prev,
		m:    make(map[string]pfunc),
	}
}

func (a *predef) set(name string, id uint16, f rideFunction) {
	a.m[name] = pfunc{
		name: name,
		id:   id,
		f:    f,
	}
}

func (a *predef) get(name string) (pfunc, bool) {
	if a == nil {
		return pfunc{}, false
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
