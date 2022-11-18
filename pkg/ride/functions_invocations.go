package ride

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func invokeFunctionFromDApp(env environment, recipient proto.Recipient, fnName rideString, listArgs rideList) (Result, error) {
	tree, err := env.state().NewestScriptByAccount(recipient)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to get script by recipient")
	}
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
		return nil, EvaluationErrorAddComplexity(err, e.complexity())
	}
	return res, nil
}
