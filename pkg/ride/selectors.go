package ride

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
	case 6:
		return functionV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
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
	case 6:
		return checkFunctionV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}

func selectEvaluationCostsProvider(v int, separateEvaluationCosts bool) (map[string]int, error) {
	switch v {
	case 1, 2:
		if separateEvaluationCosts {
			return EvaluationCatalogueV2, nil
		}
		return CatalogueV2, nil
	case 3:
		if separateEvaluationCosts {
			return EvaluationCatalogueV3, nil
		}
		return CatalogueV3, nil
	case 4:
		if separateEvaluationCosts {
			return EvaluationCatalogueV4, nil
		}
		return CatalogueV4, nil
	case 5:
		if separateEvaluationCosts {
			return EvaluationCatalogueV5, nil
		}
		return CatalogueV5, nil
	case 6:
		return CatalogueV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
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
	case 6:
		return functionNameV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
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
	case 6:
		return constantV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
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
	case 6:
		return checkConstantV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}
