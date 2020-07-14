package transpiler

type IfFsm struct {
	BaseFsm
}

func (a IfFsm) If() Fsm {
	return IfFsmTransition(a.BaseFsm)
}

func (a IfFsm) BlockV1(name []byte) Fsm {
	a.BaseFsm.e.Put("IfFsm condition BlockV1: illegal call, it look like (if (let x = ))")
	return HaltFsm{}
}

func IfFsmTransition(fsm BaseFsm) Fsm {
	fsm.lift.Up(func() Fsm {
		fsm.op.JumpIfNot()
		rewriteAt := fsm.op.Pos()
		fsm.op.I32(0)
		return IfTrueTransition(fsm, rewriteAt)
	})
	return IfFsm{
		BaseFsm: fsm,
	}
}
