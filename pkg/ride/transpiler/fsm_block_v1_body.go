package transpiler

type BlockV1FsmBody struct {
	BaseFsm
	blockShiftPos int32
}

func (a BlockV1FsmBody) Call(name []byte, argc int32) Fsm {
	return CallFsmTransition(name, argc, a.BaseFsm)
}

func (a BlockV1FsmBody) BlockV1(name []byte) Fsm {
	return BlockV1Transition(a.BaseFsm, name)
}

func (a BlockV1FsmBody) If() Fsm {
	return IfFsmTransition(a.BaseFsm)
}

func BlockV1BodyTransition(fsm BaseFsm) Fsm {
	return &BlockV1FsmBody{
		BaseFsm: fsm,
	}
}
