package transpiler

type GetterFsm struct {
	BaseFsm
}

func GetterTransition(b BaseFsm) Fsm {
	//b.lift.Up(func() Fsm {
	//	return GetterFieldTransition(b)
	//})
	return GetterFsm{
		BaseFsm: b,
	}
}
