package ride

import "github.com/pkg/errors"

type callable struct {
	entryPoint    int
	parameterName string
}

type RideScript interface {
	Run(env RideEnvironment) (RideResult, error)
	code() []byte
}

type SimpleScript struct {
	LibVersion int
	EntryPoint int
	Code       []byte
	Constants  []rideType
}

func (s *SimpleScript) Run(env RideEnvironment) (RideResult, error) {
	fs, err := selectFunctions(s.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "simple script execution failed")
	}
	gcs, err := selectConstants(s.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "simple script execution failed")
	}
	np, err := selectFunctionNameProvider(s.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "simple script execution failed")
	}
	m := vm{
		env:          env,
		code:         s.Code,
		ip:           0,
		constants:    s.Constants,
		functions:    fs,
		globals:      gcs,
		stack:        make([]rideType, 0, 2),
		calls:        make([]frame, 0, 2),
		functionName: np,
	}
	r, err := m.run()
	if err != nil {
		return nil, errors.Wrap(err, "simple script execution failed")
	}
	return r, nil
}

func (s *SimpleScript) code() []byte {
	return s.Code
}

type DAppScript struct {
	LibVersion  int
	Code        []byte
	Constants   []rideType
	EntryPoints map[string]callable
}

func (s *DAppScript) Run(env RideEnvironment) (RideResult, error) {
	if _, ok := s.EntryPoints[""]; !ok {
		return nil, errors.Errorf("no verifier")
	}
	fs, err := selectFunctions(s.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "script execution failed")
	}
	gcs, err := selectConstants(s.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "script execution failed")
	}
	np, err := selectFunctionNameProvider(s.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "script execution failed")
	}
	m := vm{
		env:          env,
		code:         s.Code,
		ip:           0,
		constants:    s.Constants,
		functions:    fs,
		globals:      gcs,
		stack:        make([]rideType, 0, 2),
		calls:        make([]frame, 0, 2),
		functionName: np,
	}
	r, err := m.run()
	if err != nil {
		return nil, errors.Wrap(err, "script execution failed")
	}
	return r, nil
}

func (s *DAppScript) code() []byte {
	return s.Code
}
