package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Executable struct {
	LibVersion  int
	IsDapp      bool
	hasVerifier bool
	position    int // Non-default value assumes interrupted evaluation.
	stack       []rideType
	ByteCode    []byte
	EntryPoints map[string]Entrypoint
	References  map[uniqueid]point
}

func (a *Executable) HasVerifier() bool {
	return a.hasVerifier
}

func (a *Executable) Interrupted() bool {
	return a.position != 0
}

func (a *Executable) Continue() (RideResult, error) {
	panic("unimplemented!")
}

func (a *Executable) Verify(environment RideEnvironment) (RideResult, error) {
	if a.Interrupted() {
		return nil, errors.New("illegal `Verify` call on interrupted state")
	}
	if a.IsDapp {
		if !a.HasVerifier() {
			return nil, errors.Errorf("no verifier attached to script")
		}
		if environment == nil {
			return nil, errors.Errorf("expect to get env.transaction(), but env is nil")
		}
		return a.runWithoutChecks(environment, "", []rideType{environment.transaction()})
	}
	return a.runWithoutChecks(environment, "", nil)
}

func (a *Executable) Entrypoint(name string) (Entrypoint, error) {
	if a.Interrupted() {
		return Entrypoint{}, errors.New("illegal `Entrypoint` call on interrupted state")
	}
	v, ok := a.EntryPoints[name]
	if !ok {
		return Entrypoint{}, errors.Errorf("entrypoint %s not found", name)
	}
	return v, nil
}

func (a *Executable) runWithoutChecks(environment RideEnvironment, name string, arguments []rideType) (RideResult, error) {
	fcall, ok := a.EntryPoints[name]
	if !ok {
		return nil, errors.Errorf("function %s not found", name)
	}
	vm, err := a.makeVm(environment, int(fcall.at), arguments)
	if err != nil {
		return nil, err
	}
	v, err := vm.run()
	if err != nil {
		return ScriptResult{res: false, msg: "", calls: vm.calls, refs: vm.ref, operations: vm.numOperations}, err
	}
	switch tv := v.(type) {
	case rideThrow:
		if a.IsDapp {
			return DAppResult{res: false, msg: string(tv), calls: vm.calls, refs: vm.ref}, nil
		}
		return ScriptResult{res: false, msg: string(tv), calls: vm.calls, refs: vm.ref}, nil
	case rideBoolean:
		return ScriptResult{res: bool(tv), operations: vm.numOperations, calls: vm.calls, refs: vm.ref}, nil
	case rideObject:
		actions, err := objectToActions(vm.env, tv)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert evaluation result")
		}
		return DAppResult{res: true, actions: actions, msg: "", calls: vm.calls, refs: vm.ref}, nil
	case rideList:
		actions := make([]proto.ScriptAction, len(tv))
		for i, item := range tv {
			a, err := convertToAction(vm.env, item)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert evaluation result")
			}
			actions[i] = a
		}
		return DAppResult{res: true, actions: actions, calls: vm.calls, refs: vm.ref}, nil
	default:
		return ScriptResult{calls: vm.calls}, errors.Errorf("unexpected result value '%v' of type '%T'", v, v)
	}
}

func (a *Executable) Invoke(env RideEnvironment, name string, arguments []rideType) (RideResult, error) {
	if a.Interrupted() {
		return nil, errors.New("illegal `Invoke` call on interrupted state")
	}
	if name == "" {
		return nil, errors.Errorf("expected func name, found \"\"")
	}
	fcall, ok := a.EntryPoints[name]
	if !ok {
		return nil, errors.Errorf("function %s not found", name)
	}
	arguments = append([]rideType{env.invocation()}, arguments...)
	if len(arguments) != int(fcall.argn)+1 {
		return nil, errors.Errorf("func `%s` requires %d arguments(1 invoke + %d args), but provided %d", name, fcall.argn+1, fcall.argn, len(arguments))
	}
	return a.runWithoutChecks(env, name, arguments)
}

func (a *Executable) run(environment RideEnvironment, arguments []rideType) (rideType, error) {
	vm, err := a.makeVm(environment, int(a.EntryPoints[""].at), arguments)
	if err != nil {
		return nil, err
	}
	return vm.run()
}

func (a *Executable) makeVm(environment RideEnvironment, entrypoint int, arguments []rideType) (*vm, error) {
	refs := copyReferences(a.References)
	return &vm{
		code:       a.ByteCode,
		ip:         entrypoint,
		env:        environment,
		ref:        refs,
		stack:      arguments,
		jmps:       []int{1},
		libVersion: a.LibVersion,
	}, nil
}

func (a *Executable) Serialize(s Serializer) error {
	var magicNumber byte = 202
	s.Byte(magicNumber)
	s.Uint16(uint16(a.LibVersion))
	s.Bool(a.IsDapp)
	s.Bool(a.hasVerifier)
	err := s.Bytes(a.ByteCode)
	if err != nil {
		return err
	}
	// entrypoints
	err = s.Map(len(a.EntryPoints), func(m Map) error {
		for k, v := range a.EntryPoints {
			err := m.RideString(rideString(k))
			if err != nil {
				return err
			}
			err = v.Serialize(s)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// references
	err = s.Map(len(a.References), func(m Map) error {
		for k, v := range a.References {
			m.Uint16(k)
			err := v.Serialize(s)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func DeserializeExecutable(source []byte) (*Executable, error) {
	d := NewDeserializer(source)
	magic, err := d.Byte()
	if err != nil {
		return nil, err
	}
	if magic != 202 {
		return nil, errors.New("invalid magic number")
	}
	libVersion, err := d.Uint16()
	if err != nil {
		return nil, err
	}
	isDapp, err := d.Bool()
	if err != nil {
		return nil, err
	}
	hasVerifier, err := d.Bool()
	if err != nil {
		return nil, err
	}
	byteCode, err := d.Bytes()
	if err != nil {
		return nil, err
	}
	size, err := d.Map()
	if err != nil {
		return nil, err
	}
	entrypoints := make(map[string]Entrypoint, size)
	for i := uint16(0); i < size; i++ {
		name, err := d.RideString()
		if err != nil {
			return nil, err
		}
		entrypoint, err := deserializeEntrypoint(d)
		if err != nil {
			return nil, err
		}
		entrypoints[name] = entrypoint
	}

	size, err = d.Map()
	if err != nil {
		return nil, err
	}
	references := make(map[uniqueid]point, size)
	for i := uint16(0); i < size; i++ {
		id, err := d.Uint16()
		if err != nil {
			return nil, err
		}
		p, err := deserializePoint(d)
		if err != nil {
			return nil, err
		}
		references[id] = p
	}

	return &Executable{
		LibVersion:  int(libVersion),
		IsDapp:      isDapp,
		ByteCode:    byteCode,
		hasVerifier: hasVerifier,
		EntryPoints: entrypoints,
		References:  references,
	}, nil
}

func copyReferences(refs Refs) Refs {
	out := make(Refs, len(refs))
	for k, v := range refs {
		out[k] = v
	}
	return out
}
