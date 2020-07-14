package transpiler

import (
	op2 "github.com/wavesplatform/gowaves/pkg/ride/op"
)

type BaseFsm struct {
	op   *op2.OpCodeBuilderImpl
	lift *lift
	e    Err
}

func (a BaseFsm) If() Fsm {
	return IfFsmTransition(a)
}

func (a BaseFsm) Getter() Fsm {
	return GetterTransition(a)
}

func (a BaseFsm) BlockV1(name []byte) Fsm {
	return BlockV1Transition(a, name)
}

func (a BaseFsm) Long(i int64) Fsm {
	a.op.StackPushL(i)
	return a.lift.Down()
}

func (a BaseFsm) Bool(b bool) Fsm {
	a.op.StackPushB(b)
	return a.lift.Down()
}

func (a BaseFsm) String(s []byte) Fsm {
	a.op.StackPushS(s)
	return a.lift.Down()
}

func (a BaseFsm) Bytes(b []byte) Fsm {
	a.op.StackPushBytes(b)
	return a.lift.Down()
}

func (a BaseFsm) Ref(s []byte) Fsm {
	a.op.JmpRef(s)
	return a.lift.Down()
}

func (a BaseFsm) Call(name []byte, argc int32) Fsm {
	return CallFsmTransition(name, argc, a)
}

//func (a BaseFsm) Finalize() Fsm {
//	return a.lift.Down()
//}
