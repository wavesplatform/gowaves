package transpiler

import (
	"github.com/wavesplatform/gowaves/pkg/ride/op"
)

type InitialFsm struct {
	BaseFsm
}

func (a InitialFsm) If() Fsm {
	return IfFsmTransition(a.BaseFsm)
}

func NewInitial(opcodeBuilder *op.OpCodeBuilderImpl, e ...Err) Fsm {
	if len(e) == 0 {
		e = []Err{&ErrImpl{}}
	}
	lift := NewLift()
	a := &InitialFsm{
		BaseFsm: BaseFsm{
			op:   opcodeBuilder,
			lift: lift,
			e:    e[0],
		},
	}
	lift.Up(func() Fsm {
		opcodeBuilder.Ret()
		return a
	})
	return a
}
