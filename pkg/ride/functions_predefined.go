package ride

import (
	"math"
	//"github.com/pkg/errors"
)

func tx(env RideEnvironment, _ ...rideType) (rideType, error) {
	return env.transaction(), nil
}

func mergeWithPredefined(f func(id int) rideFunction, p *predef) func(id int) rideFunction {
	return func(id int) rideFunction {
		if c := p.getn(id); c != nil {
			return c
		}
		return f(id)
	}
}

func this(env RideEnvironment, _ ...rideType) (rideType, error) {
	return env.this(), nil
}

func height(env RideEnvironment, _ ...rideType) (rideType, error) {
	return env.height(), nil
}

func nilFunc(env RideEnvironment, _ ...rideType) (rideType, error) {
	var out rideList = nil
	return out, nil
}

func retRideType() rideList {
	return nil
}

var predefinedFunctions = map[string]predefFunc{
	"tx":     {id: math.MaxUint16 - 0, f: tx},
	"unit":   {id: math.MaxUint16 - 1, f: unit},
	"NOALG":  {id: math.MaxUint16 - 2, f: createNoAlg},
	"this":   {id: math.MaxUint16 - 3, f: this},
	"height": {id: math.MaxUint16 - 4, f: height},
	"nil":    {id: math.MaxUint16 - 5, f: nilFunc},
}

var predefined *predef

func init() {
	predefined = newPredef(nil)
	for k, v := range predefinedFunctions {
		predefined.set(k, v.id, v.f)
	}
}
