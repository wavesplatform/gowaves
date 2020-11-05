package ride

type Executable struct {
	LibVersion  int
	ByteCode    []byte
	Constants   []rideType
	EntryPoints map[string]uint16
}

func (a *Executable) Run(environment RideEnvironment) (RideResult, error) {
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
		constants:    a.Constants,
		functions:    mergeWithPredefined(fSelect, predefined),
		functionName: provider,
		env:          environment,
	}

	//v.push(environment.transaction())

	return v.run()
}
