package transpiler

type CallFsm struct {
	BaseFsm
}

func (a CallFsm) BlockV1(name []byte) Fsm {
	a.e.Put("CallFsm BlockV1: illegal call")
	return HaltFsm{}
	//panic("CallFsm BlockV1: illegal call")
}

func (a CallFsm) Call(name []byte, argc int32) Fsm {
	return CallFsmTransition(name, argc, a.BaseFsm)
}

func (a CallFsm) If() Fsm {
	return IfFsmTransition(a.BaseFsm)
}

func CallFsmTransition(name []byte, argc int32, fsm BaseFsm) Fsm {
	a := CallFsm{
		BaseFsm: fsm,
	}
	if argc > 0 {
		for i := int32(0); i < argc; i++ {
			first := i == 0
			if first {
				//fsm.op.Call(name)
				fsm.lift.Up(func() Fsm {
					fsm.op.Call(name)
					return fsm.lift.Down()
				})
			} else {
				fsm.lift.Up(func() Fsm {
					return a
				})
			}
		}
	}
	return a
}
