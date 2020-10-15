package ride

import "github.com/pkg/errors"

func selectFunctions(v int) (func(id int) rideFunction, error) {
	switch v {
	case 1, 2:
		return functionV2, nil
	case 3:
		return functionV3, nil
	case 4:
		return functionV4, nil
	default:
		return nil, errors.Errorf("unsupported library version '%d'", v)
	}
}

func selectFunctionChecker(v int) (func(name string) (uint16, bool), error) {
	switch v {
	case 1, 2:
		return checkFunctionV2, nil
	case 3:
		return checkFunctionV3, nil
	case 4:
		return checkFunctionV4, nil
	default:
		return nil, errors.Errorf("unsupported library version '%d'", v)
	}
}

func selectFunctionNameProvider(v int) (func(int) string, error) {
	switch v {
	case 1, 2:
		return functionNameV2, nil
	case 3:
		return functionNameV3, nil
	case 4:
		return functionNameV4, nil
	default:
		return nil, errors.Errorf("unsupported library version '%d'", v)
	}
}

//func selectCostProvider(v int) (func(int) int, error) {
//	switch v {
//	case 1, 2:
//		return costV2, nil
//	case 3:
//		return costV3, nil
//	case 4:
//		return costV4, nil
//	default:
//		return nil, errors.Errorf("unsupported library version '%d'", v)
//	}
//}
//
func selectConstants(v int) (func(int) rideConstructor, error) {
	switch v {
	case 1:
		return constantV1, nil
	case 2:
		return constantV2, nil
	case 3:
		return constantV3, nil
	case 4:
		return constantV4, nil
	default:
		return nil, errors.Errorf("unsupported library version '%d'", v)
	}
}

func selectConstantsChecker(v int) (func(name string) (uint16, bool), error) {
	switch v {
	case 1:
		return checkConstantV1, nil
	case 2:
		return checkConstantV2, nil
	case 3:
		return checkConstantV3, nil
	case 4:
		return checkConstantV4, nil
	default:
		return nil, errors.Errorf("unsupported library version '%d'", v)
	}
}
