package ride

import (
	"math"
	//"github.com/pkg/errors"
)

func tx(env RideEnvironment, _ ...rideType) (rideType, error) {
	return env.transaction(), nil
}

//func mergeWithPredefined(f func(id int) rideFunction, p *predef) func(id int) rideFunction {
//	return func(id int) rideFunction {
//		if c := p.getn(id); c != nil {
//			return c
//		}
//		return f(id)
//	}
//}

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

func lastBlock(env RideEnvironment, _ ...rideType) (rideType, error) {
	return env.block(), nil
}

var predefinedFunctions = map[string]predefFunc{
	"tx":        {id: math.MaxUint16 - 0, f: tx},
	"unit":      {id: math.MaxUint16 - 1, f: unit},
	"NOALG":     {id: math.MaxUint16 - 2, f: createNoAlg},
	"this":      {id: math.MaxUint16 - 3, f: this},
	"height":    {id: math.MaxUint16 - 4, f: height},
	"nil":       {id: math.MaxUint16 - 5, f: nilFunc},
	"lastBlock": {id: math.MaxUint16 - 6, f: lastBlock},
	"UP":        {id: math.MaxUint16 - 7, f: createUp},
	"DOWN":      {id: math.MaxUint16 - 8, f: createDown},
	"HALFDOWN":  {id: math.MaxUint16 - 9, f: createHalfDown},
}

var predefined *predef

func init() {
	predefined = newPredef(nil)
	for k, v := range predefinedFunctions {
		predefined.set(k, v.id, v.f)
	}
}
