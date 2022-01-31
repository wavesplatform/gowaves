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

func selectEvaluationCostsProvider(v, ev int) (map[string]int, error) {
	switch v {
	case 1, 2:
		switch ev {
		case 1:
			return EvaluationCatalogueV2EvaluatorV1, nil
		default:
			return EvaluationCatalogueV2EvaluatorV2, nil
		}
	case 3:
		switch ev {
		case 1:
			return EvaluationCatalogueV3EvaluatorV1, nil
		default:
			return EvaluationCatalogueV3EvaluatorV2, nil
		}
	case 4:
		switch ev {
		case 1:
			return EvaluationCatalogueV4EvaluatorV1, nil
		default:
			return EvaluationCatalogueV4EvaluatorV2, nil
		}
	case 5:
		switch ev {
		case 1:
			return EvaluationCatalogueV5EvaluatorV1, nil
		default:
			return EvaluationCatalogueV5EvaluatorV2, nil
		}
	case 6: // Only new version of evaluator works after activation of RideV6
		return EvaluationCatalogueV6EvaluatorV2, nil
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
