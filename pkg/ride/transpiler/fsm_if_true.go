package transpiler

type IfTrueFsm struct {
	BaseFsm
}

//func (a IfTrueFsm) BlockV1(name []byte) Fsm {
//	return BlockV1Transition(a.op, a.lift, name)
//}

func (a IfTrueFsm) If() Fsm {
	return IfFsmTransition(a.BaseFsm)
}

func IfTrueTransition(fsm BaseFsm, rewriteAt int32) Fsm {
	fsm.lift.Up(func() Fsm {
		fsm.op.Jmp()
		afterTrueBlockShift := fsm.op.Pos()
		fsm.op.I32(0)
		return IfFalseTransition(fsm, rewriteAt, afterTrueBlockShift)
	})
	return IfTrueFsm{
		BaseFsm: fsm,
	}
}
