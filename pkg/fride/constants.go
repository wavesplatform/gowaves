// Code generated by fride/generate/main.go. DO NOT EDIT.

package fride

const _constants_V1 = "heighttxunit"

var _constructors_V1 = [...]rideConstructor{newHeight, newTx, newUnit}
var _c_index_V1 = [...]int{0, 6, 8, 12}

func constantV1(id int) rideConstructor {
	if id < 0 || id > 2 {
		return nil
	}
	return _constructors_V1[id]
}
func checkConstantV1(name string) (uint16, bool) {
	for i := 0; i <= 2; i++ {
		if _constants_V1[_c_index_V1[i]:_c_index_V1[i+1]] == name {
			return uint16(i), true
		}
	}
	return 0, false
}

const _constants_V2 = "BuyCEILINGDOWNFLOORHALFDOWNHALFEVENHALFUPSellUPheightniltxunit"

var _constructors_V2 = [...]rideConstructor{newBuy, newCeiling, newDown, newFloor, newHalfDown, newHalfEven, newHalfUp, newSell, newUp, newHeight, newNil, newTx, newUnit}
var _c_index_V2 = [...]int{0, 3, 10, 14, 19, 27, 35, 41, 45, 47, 53, 56, 58, 62}

func constantV2(id int) rideConstructor {
	if id < 0 || id > 12 {
		return nil
	}
	return _constructors_V2[id]
}
func checkConstantV2(name string) (uint16, bool) {
	for i := 0; i <= 12; i++ {
		if _constants_V2[_c_index_V2[i]:_c_index_V2[i+1]] == name {
			return uint16(i), true
		}
	}
	return 0, false
}

const _constants_V3 = "BuyCEILINGDOWNFLOORHALFDOWNHALFEVENHALFUPMD5NOALGSHA1SHA224SHA256SHA3224SHA3256SHA3384SHA3512SHA384SHA512SellUPheightlastBlocknilthistxunit"

var _constructors_V3 = [...]rideConstructor{newBuy, newCeiling, newDown, newFloor, newHalfDown, newHalfEven, newHalfUp, newMd5, newNoAlg, newSha1, newSha224, newSha256, newSha3224, newSha3256, newSha3384, newSha3512, newSha384, newSha512, newSell, newUp, newHeight, newLastBlock, newNil, newThis, newTx, newUnit}
var _c_index_V3 = [...]int{0, 3, 10, 14, 19, 27, 35, 41, 44, 49, 53, 59, 65, 72, 79, 86, 93, 99, 105, 109, 111, 117, 126, 129, 133, 135, 139}

func constantV3(id int) rideConstructor {
	if id < 0 || id > 25 {
		return nil
	}
	return _constructors_V3[id]
}
func checkConstantV3(name string) (uint16, bool) {
	for i := 0; i <= 25; i++ {
		if _constants_V3[_c_index_V3[i]:_c_index_V3[i+1]] == name {
			return uint16(i), true
		}
	}
	return 0, false
}

const _constants_V4 = "BuyCEILINGDOWNFLOORHALFDOWNHALFEVENHALFUPMD5NOALGSHA1SHA224SHA256SHA3224SHA3256SHA3384SHA3512SHA384SHA512SellUPheightlastBlocknilthistxunit"

var _constructors_V4 = [...]rideConstructor{newBuy, newCeiling, newDown, newFloor, newHalfDown, newHalfEven, newHalfUp, newMd5, newNoAlg, newSha1, newSha224, newSha256, newSha3224, newSha3256, newSha3384, newSha3512, newSha384, newSha512, newSell, newUp, newHeight, newLastBlock, newNil, newThis, newTx, newUnit}
var _c_index_V4 = [...]int{0, 3, 10, 14, 19, 27, 35, 41, 44, 49, 53, 59, 65, 72, 79, 86, 93, 99, 105, 109, 111, 117, 126, 129, 133, 135, 139}

func constantV4(id int) rideConstructor {
	if id < 0 || id > 25 {
		return nil
	}
	return _constructors_V4[id]
}
func checkConstantV4(name string) (uint16, bool) {
	for i := 0; i <= 25; i++ {
		if _constants_V4[_c_index_V4[i]:_c_index_V4[i+1]] == name {
			return uint16(i), true
		}
	}
	return 0, false
}

func newHalfEven(RideEnvironment) rideType {
	return rideNamedType{name: "HalfEven"}
}

func createHalfEven(...rideType) (rideType, error) {
	return rideNamedType{name: "HalfEven"}, nil
}

func newHalfUp(RideEnvironment) rideType {
	return rideNamedType{name: "HalfUp"}
}

func createHalfUp(...rideType) (rideType, error) {
	return rideNamedType{name: "HalfUp"}, nil
}

func newNoAlg(RideEnvironment) rideType {
	return rideNamedType{name: "NoAlg"}
}

func createNoAlg(...rideType) (rideType, error) {
	return rideNamedType{name: "NoAlg"}, nil
}

func newSha256(RideEnvironment) rideType {
	return rideNamedType{name: "Sha256"}
}

func createSha256(...rideType) (rideType, error) {
	return rideNamedType{name: "Sha256"}, nil
}

func newCeiling(RideEnvironment) rideType {
	return rideNamedType{name: "Ceiling"}
}

func createCeiling(...rideType) (rideType, error) {
	return rideNamedType{name: "Ceiling"}, nil
}

func newDown(RideEnvironment) rideType {
	return rideNamedType{name: "Down"}
}

func createDown(...rideType) (rideType, error) {
	return rideNamedType{name: "Down"}, nil
}

func newSha224(RideEnvironment) rideType {
	return rideNamedType{name: "Sha224"}
}

func createSha224(...rideType) (rideType, error) {
	return rideNamedType{name: "Sha224"}, nil
}

func newSha512(RideEnvironment) rideType {
	return rideNamedType{name: "Sha512"}
}

func createSha512(...rideType) (rideType, error) {
	return rideNamedType{name: "Sha512"}, nil
}

func newSha3512(RideEnvironment) rideType {
	return rideNamedType{name: "Sha3512"}
}

func createSha3512(...rideType) (rideType, error) {
	return rideNamedType{name: "Sha3512"}, nil
}

func newSell(RideEnvironment) rideType {
	return rideNamedType{name: "Sell"}
}

func createSell(...rideType) (rideType, error) {
	return rideNamedType{name: "Sell"}, nil
}

func newUp(RideEnvironment) rideType {
	return rideNamedType{name: "Up"}
}

func createUp(...rideType) (rideType, error) {
	return rideNamedType{name: "Up"}, nil
}

func newSha1(RideEnvironment) rideType {
	return rideNamedType{name: "Sha1"}
}

func createSha1(...rideType) (rideType, error) {
	return rideNamedType{name: "Sha1"}, nil
}

func newSha3256(RideEnvironment) rideType {
	return rideNamedType{name: "Sha3256"}
}

func createSha3256(...rideType) (rideType, error) {
	return rideNamedType{name: "Sha3256"}, nil
}

func newBuy(RideEnvironment) rideType {
	return rideNamedType{name: "Buy"}
}

func createBuy(...rideType) (rideType, error) {
	return rideNamedType{name: "Buy"}, nil
}

func newFloor(RideEnvironment) rideType {
	return rideNamedType{name: "Floor"}
}

func createFloor(...rideType) (rideType, error) {
	return rideNamedType{name: "Floor"}, nil
}

func newMd5(RideEnvironment) rideType {
	return rideNamedType{name: "Md5"}
}

func createMd5(...rideType) (rideType, error) {
	return rideNamedType{name: "Md5"}, nil
}

func newSha384(RideEnvironment) rideType {
	return rideNamedType{name: "Sha384"}
}

func createSha384(...rideType) (rideType, error) {
	return rideNamedType{name: "Sha384"}, nil
}

func newSha3224(RideEnvironment) rideType {
	return rideNamedType{name: "Sha3224"}
}

func createSha3224(...rideType) (rideType, error) {
	return rideNamedType{name: "Sha3224"}, nil
}

func newSha3384(RideEnvironment) rideType {
	return rideNamedType{name: "Sha3384"}
}

func createSha3384(...rideType) (rideType, error) {
	return rideNamedType{name: "Sha3384"}, nil
}

func newHalfDown(RideEnvironment) rideType {
	return rideNamedType{name: "HalfDown"}
}

func createHalfDown(...rideType) (rideType, error) {
	return rideNamedType{name: "HalfDown"}, nil
}
