package transpiler

type HaltFsm struct {
}

func (a HaltFsm) Long(i int64) Fsm {
	return a
}

func (a HaltFsm) Bool(b bool) Fsm {
	return a
}

func (a HaltFsm) Ref(bytes []byte) Fsm {
	return a
}

func (a HaltFsm) BlockV1(name []byte) Fsm {
	return a
}

func (a HaltFsm) Call(name []byte, argc int32) Fsm {
	return a
}

func (a HaltFsm) String(bytes []byte) Fsm {
	return a
}

func (a HaltFsm) Bytes(bytes []byte) Fsm {
	return a
}

func (a HaltFsm) If() Fsm {
	return a
}

func (a HaltFsm) Getter() Fsm {
	return a
}
