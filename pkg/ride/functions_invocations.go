package ride

import (
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

func invokeFunctionFromDApp(env environment, tree *ast.Tree, fnName rideString, listArgs rideList) (Result, error) {
	args, err := convertListArguments(listArgs, env.rideV6Activated())
	if err != nil {
		return nil, EvaluationFailure.Wrapf(err, "failed to invoke function '%s'", fnName)
	}
	e, err := treeFunctionEvaluator(env, tree, string(fnName), args)
	if err != nil {
		return nil, EvaluationFailure.Wrapf(err, "failed to call function '%s'", fnName)
	}
	res, err := e.evaluate()
	if err != nil {
		// Evaluation failed we have to add spent execution complexity to an error
		return nil, EvaluationErrorSetComplexity(err, e.complexity())
	}
	return res, nil
}
