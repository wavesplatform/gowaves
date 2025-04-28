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
	case ast.LibV7:
		if enableInvocation {
			return functionsV7, nil
		}
		return expressionFunctionsV7, nil
	case ast.LibV8:
		if enableInvocation {
			return functionsV8, nil
		}
		return expressionFunctionsV8, nil
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
	case ast.LibV7:
		return functionV7, nil
	case ast.LibV8:
		return functionV8, nil
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
	case ast.LibV7:
		return checkFunctionV7, nil
	case ast.LibV8:
		return checkFunctionV8, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}

type evaluatorVersion uint8

const (
	evaluatorV1 evaluatorVersion = 1
	evaluatorV2 evaluatorVersion = 2
)

func selectByEvaluatorVersion(ev evaluatorVersion, catalogueEV1, catalogueEV2 map[string]int) (map[string]int, error) {
	switch ev {
	case evaluatorV1:
		return catalogueEV1, nil
	case evaluatorV2:
		return catalogueEV2, nil
	default:
		return nil, EvaluationFailure.Errorf("catalogue not found for evaluator version '%d'", ev)
	}
}

func selectEvaluationCostsProvider(v ast.LibraryVersion, ev evaluatorVersion) (map[string]int, error) {
	switch v {
	case ast.LibV1, ast.LibV2:
		return selectByEvaluatorVersion(ev, EvaluationCatalogueV2EvaluatorV1, EvaluationCatalogueV2EvaluatorV2)
	case ast.LibV3:
		return selectByEvaluatorVersion(ev, EvaluationCatalogueV3EvaluatorV1, EvaluationCatalogueV3EvaluatorV2)
	case ast.LibV4:
		return selectByEvaluatorVersion(ev, EvaluationCatalogueV4EvaluatorV1, EvaluationCatalogueV4EvaluatorV2)
	case ast.LibV5:
		return selectByEvaluatorVersion(ev, EvaluationCatalogueV5EvaluatorV1, EvaluationCatalogueV5EvaluatorV2)
	case ast.LibV6: // Only second version of evaluator actually works after activation of RideV6, but support both
		return selectByEvaluatorVersion(ev, EvaluationCatalogueV6EvaluatorV1, EvaluationCatalogueV6EvaluatorV2)
	case ast.LibV7:
		return selectByEvaluatorVersion(ev, EvaluationCatalogueV7EvaluatorV1, EvaluationCatalogueV7EvaluatorV2)
	case ast.LibV8:
		return selectByEvaluatorVersion(ev, EvaluationCatalogueV8EvaluatorV1, EvaluationCatalogueV8EvaluatorV2)
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
	case ast.LibV7:
		return functionNameV7, nil
	case ast.LibV8:
		return functionNameV8, nil
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
	case ast.LibV7:
		return constantV7, nil
	case ast.LibV8:
		return constantV8, nil
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
	case ast.LibV7:
		return checkConstantV7, nil
	case ast.LibV8:
		return checkConstantV8, nil
	default:
		return nil, EvaluationFailure.Errorf("unsupported library version '%d'", v)
	}
}
