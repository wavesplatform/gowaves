package ride

import "github.com/wavesplatform/gowaves/pkg/ride/ast"

func selectFunctionsByName(v ast.LibraryVersion, enableInvocation bool) (func(string) (rideFunction, bool), error) {
	switch v {
	case ast.LibV1, ast.LibV2:
		return functionsV2, nil
	case ast.LibV3:
		return functionsV3, nil
	case ast.LibV4:
		return functionsV4, nil
	case ast.LibV5:
		if enableInvocation {
			return functionsV5, nil
		}
		return expressionFunctionsV5, nil
	case ast.LibV6:
		if enableInvocation {
			return functionsV6, nil
		}
		return expressionFunctionsV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}

func selectFunctions(v ast.LibraryVersion) (func(id int) rideFunction, error) {
	switch v {
	case ast.LibV1, ast.LibV2:
		return functionV2, nil
	case ast.LibV3:
		return functionV3, nil
	case ast.LibV4:
		return functionV4, nil
	case ast.LibV5:
		return functionV5, nil
	case ast.LibV6:
		return functionV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}

func selectFunctionChecker(v ast.LibraryVersion) (func(name string) (uint16, bool), error) {
	switch v {
	case ast.LibV1, ast.LibV2:
		return checkFunctionV2, nil
	case ast.LibV3:
		return checkFunctionV3, nil
	case ast.LibV4:
		return checkFunctionV4, nil
	case ast.LibV5:
		return checkFunctionV5, nil
	case ast.LibV6:
		return checkFunctionV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}

func selectEvaluationCostsProvider(v ast.LibraryVersion, ev int) (map[string]int, error) {
	switch v {
	case ast.LibV1, ast.LibV2:
		switch ev {
		case 1:
			return EvaluationCatalogueV2EvaluatorV1, nil
		default:
			return EvaluationCatalogueV2EvaluatorV2, nil
		}
	case ast.LibV3:
		switch ev {
		case 1:
			return EvaluationCatalogueV3EvaluatorV1, nil
		default:
			return EvaluationCatalogueV3EvaluatorV2, nil
		}
	case ast.LibV4:
		switch ev {
		case 1:
			return EvaluationCatalogueV4EvaluatorV1, nil
		default:
			return EvaluationCatalogueV4EvaluatorV2, nil
		}
	case ast.LibV5:
		switch ev {
		case 1:
			return EvaluationCatalogueV5EvaluatorV1, nil
		default:
			return EvaluationCatalogueV5EvaluatorV2, nil
		}
	case ast.LibV6: // Only new version of evaluator works after activation of RideV6
		return EvaluationCatalogueV6EvaluatorV2, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}

func selectFunctionNameProvider(v ast.LibraryVersion) (func(int) string, error) {
	switch v {
	case ast.LibV1, ast.LibV2:
		return functionNameV2, nil
	case ast.LibV3:
		return functionNameV3, nil
	case ast.LibV4:
		return functionNameV4, nil
	case ast.LibV5:
		return functionNameV5, nil
	case ast.LibV6:
		return functionNameV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}

func selectConstants(v ast.LibraryVersion) (func(int) rideConstructor, error) {
	switch v {
	case ast.LibV1:
		return constantV1, nil
	case ast.LibV2:
		return constantV2, nil
	case ast.LibV3:
		return constantV3, nil
	case ast.LibV4:
		return constantV4, nil
	case ast.LibV5:
		return constantV5, nil
	case ast.LibV6:
		return constantV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}

func selectConstantsChecker(v ast.LibraryVersion) (func(name string) (uint16, bool), error) {
	switch v {
	case ast.LibV1:
		return checkConstantV1, nil
	case ast.LibV2:
		return checkConstantV2, nil
	case ast.LibV3:
		return checkConstantV3, nil
	case ast.LibV4:
		return checkConstantV4, nil
	case ast.LibV5:
		return checkConstantV5, nil
	case ast.LibV6:
		return checkConstantV6, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}
