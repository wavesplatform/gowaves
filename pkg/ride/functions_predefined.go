package ride

import (
	"math"
	//"github.com/pkg/errors"
)

func tx(env RideEnvironment, _ ...rideType) (rideType, error) {
	return env.transaction(), nil
}

//func property(env RideEnvironment, args ...rideType) (rideType, error) {
//	if len(args) != 2 {
//		return nil, errors.Errorf("property: expected pass 2 arguments, got %d", len(args))
//	}
//	name, ok := args[1].(rideString)
//	if !ok {
//		return nil, errors.Errorf("property: expected second argument to be string, got %T", args[1])
//	}
//
//	v, err := args[0].get(string(name))
//	if err != nil {
//		return nil, errors.Wrap(err, "property")
//	}
//	return v, nil
//}

func mergeWithPredefined(f func(id int) rideFunction, p predef) func(id int) rideFunction {
	return func(id int) rideFunction {
		if c := p.getn(id); c != nil {
			return c
		}
		return f(id)
	}
}

var predefined predef = map[string]predefFunc{
	"tx": {id: math.MaxUint16, f: tx},
	//"$property": {id: math.MaxUint16 - 1, f: property},
}
