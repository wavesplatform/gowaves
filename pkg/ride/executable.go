package ride

type Executable struct {
	LibVersion  int
	ByteCode    []byte
	EntryPoints map[string]uint16
	References  map[uniqueid]point
}

func (a *Executable) Run(environment RideEnvironment, arguments []rideType) (RideResult, error) {
	fSelect, err := selectFunctions(a.LibVersion)
	if err != nil {
		return nil, err
	}

	provider, err := selectFunctionNameProvider(a.LibVersion)
	if err != nil {
		return nil, err
	}

	v := vm{
		code:         a.ByteCode,
		ip:           int(a.EntryPoints[""]),
		functions:    mergeWithPredefined(fSelect, predefined),
		functionName: provider,
		env:          environment,
		ref:          a.References,
		stack:        arguments,
	}

	return v.run()
}
