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
	case 5:
		return functionV5, nil
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
	case 5:
		return checkFunctionV5, nil
	default:
		return nil, errors.Errorf("unsupported library version '%d'", v)
	}
}

func selectEvaluationCostsProvider(v int) (map[string]int, map[string]struct{}, error) {
	switch v {
	case 1, 2:
		return CatalogueV2, FreeFunctionsV2, nil
	case 3:
		return CatalogueV3, FreeFunctionsV3, nil
	case 4:
		return CatalogueV4, FreeFunctionsV4, nil
	case 5:
		return CatalogueV5, FreeFunctionsV5, nil
	default:
		return nil, nil, errors.Errorf("unsupported library version '%d'", v)
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
	case 5:
		return functionNameV5, nil
	default:
		return nil, errors.Errorf("unsupported library version '%d'", v)
	}
}

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
	case 5:
		return constantV5, nil
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
	case 5:
		return checkConstantV5, nil
	default:
		return nil, errors.Errorf("unsupported library version '%d'", v)
	}
}
