package ride

import (
	"math"
	//"github.com/pkg/errors"
)

func tx(env RideEnvironment, _ ...rideType) (rideType, error) {
	return env.transaction(), nil
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

func lastBlock(env RideEnvironment, _ ...rideType) (rideType, error) {
	return env.block(), nil
}

// Order is important! Only add, avoid changes.
var predefinedFunctions = []predefFunc{
	{"tx", tx},
	{"unit", unit},
	{"NOALG", createNoAlg},
	{"this", this},
	{"height", height},
	{"nil", nilFunc},
	{"lastBlock", lastBlock},
	{"UP", createUp},
	{"DOWN", createDown},
	{"HALFDOWN", createHalfDown},
	{"HALFUP", createHalfUp},
	{"MD5", createMd5},
	{"SHA1", createSha1},
	{"SHA224", createSha224},
	{"SHA256", createSha256},
	{"SHA384", createSha384},
	{"SHA512", createSha512},
	{"SHA3224", createSha3224},
	{"SHA3256", createSha3256},
	{"SHA3384", createSha3384},
	{"SHA3512", createSha3512},
	{"Buy", createBuy},
	{"Sell", createSell},
	{"CEILING", createCeiling},
	{"HALFEVEN", createHalfEven},
}

var predefined *predef

func init() {
	predefined = newPredef(nil)
	for k, v := range predefinedFunctions {
		predefined.set(v.name, uint16(math.MaxUint16-k), v.f)
	}
}
