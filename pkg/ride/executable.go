package ride

import (
	"io"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Executable struct {
	LibVersion  int
	ByteCode    []byte
	EntryPoints map[string]uint16
	References  map[uniqueid]point
	IsDapp      bool
}

func (a *Executable) Run(environment RideEnvironment, arguments []rideType) (RideResult, error) {
	vm, err := a.makeVm(environment, arguments)
	if err != nil {
		return nil, err
	}
	v, err := vm.run()
	if err != nil {
		return nil, err
	}
	switch tv := v.(type) {
	case rideThrow:
		if a.IsDapp {
			return DAppResult{res: false, msg: string(tv), calls: vm.calls}, nil
		}
		return ScriptResult{res: false, msg: string(tv), calls: vm.calls}, nil
	case rideBoolean:
		return ScriptResult{res: bool(tv), operations: vm.numOperations, calls: vm.calls}, nil
	case rideObject:
		actions, err := objectToActions(vm.env, tv)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert evaluation result")
		}
		return DAppResult{res: true, actions: actions, msg: "", calls: vm.calls}, nil
	case rideList:
		actions := make([]proto.ScriptAction, len(tv))
		for i, item := range tv {
			a, err := convertToAction(vm.env, item)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert evaluation result")
			}
			actions[i] = a
		}
		return DAppResult{res: true, actions: actions, calls: vm.calls}, nil
	default:
		return nil, errors.Errorf("unexpected result value '%v' of type '%T'", v, v)
	}
}

func (a *Executable) run(environment RideEnvironment, arguments []rideType) (rideType, error) {
	vm, err := a.makeVm(environment, arguments)
	if err != nil {
		return nil, err
	}
	return vm.run()
}

func (a *Executable) makeVm(environment RideEnvironment, arguments []rideType) (*vm, error) {
	fSelect, err := selectFunctions(a.LibVersion)
	if err != nil {
		return nil, err
	}
	provider, err := selectFunctionNameProvider(a.LibVersion)
	if err != nil {
		return nil, err
	}
	return &vm{
		code:         a.ByteCode,
		ip:           int(a.EntryPoints[""]),
		functions:    fSelect,
		functionName: provider,
		env:          environment,
		ref:          a.References,
		stack:        arguments,
	}, nil
}

func (a *Executable) WriteTo(w io.Writer) (int64, error) {
	panic("Executable WriteTo")
}
