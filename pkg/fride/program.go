package fride

import "github.com/pkg/errors"

type callable struct {
	entryPoint    int
	parameterName string
}

type RideScript interface {
	Run(env RideEnvironment) (RideResult, error)
	Estimate(v int) (int, error)
	code() []byte
}

type SimpleScript struct {
	LibVersion int
	EntryPoint int
	Code       []byte
	Constants  []rideType
	Meta       programMeta
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
		code:         s.Code,
		constants:    s.Constants,
		functions:    fs,
		globals:      gcs,
		stack:        make([]rideType, 0, 2),
		calls:        make([]frame, 0, 2),
		ip:           0,
		functionName: np,
	}
	r, err := m.run()
	if err != nil {
		return nil, errors.Wrap(err, "simple script execution failed")
	}
	return r, nil
}

func (s *SimpleScript) Estimate(v int) (int, error) {
	return 0, errors.New("not implemented")
}

func (s *SimpleScript) code() []byte {
	return s.Code
}

type DAppScript struct {
	LibVersion  int
	Code        []byte
	Constants   []rideType
	Meta        programMeta
	EntryPoints map[string]callable
}

func (s *DAppScript) Run(env RideEnvironment) (RideResult, error) {
	return nil, errors.New("not implemented")
}

func (s *DAppScript) Estimate(v int) (int, error) {
	return 0, errors.New("not implemented")
}

func (s *DAppScript) code() []byte {
	return s.Code
}
