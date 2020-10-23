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
	//ctx := newContext()
	//ctx = ctx.add("tx", environment.transaction())

	v := vm{
		code:      a.ByteCode,
		ip:        int(a.EntryPoints[""]),
		constants: a.Constants,
		functions: fSelect,
		//context:   ctx,
		functionName: provider,
	}

	//v.push(environment.transaction())

	return v.run()
}
