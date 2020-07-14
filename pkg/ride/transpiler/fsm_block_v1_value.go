package transpiler

type BlockV1FsmValue struct {
	BaseFsm
	blockShiftPos int32
}

func (a BlockV1FsmValue) If() Fsm {
	return IfFsmTransition(a.BaseFsm)
}

func BlockV1Transition(fsm BaseFsm, name []byte) Fsm {
	fsm.op.Label(name)
	pos := fsm.op.Pos()
	fsm.op.I32(0)
	fsm.lift.Up(func() Fsm {
		fsm.op.Ret()
		fsm.op.ShiftAt(pos, fsm.op.Pos())
		return BlockV1BodyTransition(fsm)
	})

	return BlockV1FsmValue{
		BaseFsm: fsm,
	}
}
