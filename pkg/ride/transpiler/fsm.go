package transpiler

import "github.com/pkg/errors"

type Err interface {
	Put(format string, values ...interface{})
}

type ErrImpl struct {
	err error
}

func (a *ErrImpl) Put(format string, values ...interface{}) {
	a.err = errors.Errorf(format, values...)
}

func (a *ErrImpl) Get() error {
	return a.err
}

type Fsm interface {
	Long(int64) Fsm
	Bool(bool) Fsm
	Ref([]byte) Fsm

	BlockV1(name []byte) Fsm

	Call(name []byte, argc int32) Fsm
	String([]byte) Fsm
	Bytes([]byte) Fsm
	If() Fsm
	Getter() Fsm
}
