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
	Meta       *scriptMeta
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
	c, err := selectCostProvider(s.LibVersion)
	if err != nil {
		return 0, errors.Wrap(err, "estimate")
	}
	n, err := selectFunctionNameProvider(s.LibVersion)
	if err != nil {
		return 0, errors.Wrap(err, "estimate")
	}
	switch v {
	case 1:
		e := estimatorV1{
			code:  s.Code,
			meta:  s.Meta,
			ip:    0,
			costs: c,
			names: n,
			scope: newEstimatorScopeV1(),
		}
		e.scope.submerge() // Add initial scope to stack
		err = e.estimate()
		if err != nil {
			return 0, errors.Wrap(err, "estimate")
		}
		return e.scope.current(), nil
	case 2:
		return 0, errors.New("not implemented")
	case 3:
		//e := estimatorV3{
		//	code:  s.Code,
		//	ip:    0,
		//	costs: c,
		//	names: n,
		//	scope: newEstimatorScopeV3(),
		//}
		//e.scope.submerge() // Add initial scope to stack
		//err = e.estimate()
		//if err != nil {
		//	return 0, errors.Wrap(err, "estimate")
		//}
		//return e.scope.current(), nil
		return 0, errors.New("not implemented")
	default:
		return 0, errors.Errorf("unsupported estimator version '%d'", v)
	}
}

func (s *SimpleScript) code() []byte {
	return s.Code
}

type DAppScript struct {
	LibVersion  int
	Code        []byte
	Constants   []rideType
	EntryPoints map[string]callable
	Meta        *scriptMeta
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
