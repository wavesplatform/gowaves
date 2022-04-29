package ride

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func invokeFunctionFromDApp(env environment, recipient proto.Recipient, fnName rideString, listArgs rideList) (Result, error) {
	newScript, err := env.state().GetByteTree(recipient)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to get script by recipient")
	}
	tree, err := Parse(newScript)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to parse script")
	}
	if tree.LibVersion < 5 {
		return nil, RuntimeError.Errorf("failed to call 'invoke' for script with version %d. Scripts with version 5 are only allowed to be used in 'invoke'", tree.LibVersion)
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
