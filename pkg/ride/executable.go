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

	v := vm{
		code:      a.ByteCode,
		ip:        int(a.EntryPoints[""]),
		constants: a.Constants,
		functions: fSelect,
	}

	return v.run()
}
