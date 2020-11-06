package ride

import (
	"math"
	//"github.com/pkg/errors"
)

func tx(env RideEnvironment, _ ...rideType) (rideType, error) {
	return env.transaction(), nil
}

func mergeWithPredefined(f func(id int) rideFunction, p predef) func(id int) rideFunction {
	return func(id int) rideFunction {
		if c := p.getn(id); c != nil {
			return c
		}
		return f(id)
	}
}

var predefined predef = map[string]predefFunc{
	"tx":    {id: math.MaxUint16 - 0, f: tx},
	"unit":  {id: math.MaxUint16 - 1, f: unit},
	"NOALG": {id: math.MaxUint16 - 2, f: createNoAlg},
}
