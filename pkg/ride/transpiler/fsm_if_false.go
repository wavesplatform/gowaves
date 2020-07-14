package transpiler

type IfFalseFsm struct {
	BaseFsm
}

func (a IfFalseFsm) If() Fsm {
	return IfFsmTransition(a.BaseFsm)
}

func IfFalseTransition(fsm BaseFsm, rewriteAt int32, rewriteAtTrueJmp int32) Fsm {
	fsm.op.ShiftAt(rewriteAt, fsm.op.Pos())
	fsm.lift.Up(func() Fsm {
		fsm.op.ShiftAt(rewriteAtTrueJmp, fsm.op.Pos())
		return fsm.lift.Down()
	})

	a := IfFalseFsm{
		BaseFsm: fsm,
	}
	return a
}
