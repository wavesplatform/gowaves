package ride

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func CallVerifier(env Environment, tree *Tree) (Result, error) {
	e, err := treeVerifierEvaluator(env, tree)
	if err != nil {
		return nil, RuntimeError.Wrap(err, "failed to call verifier")
	}
	return e.evaluate()
}

func CallFunction(env Environment, tree *Tree, name string, args proto.Arguments) (Result, error) {
	if name == "" {
		name = "default"
	}
	arguments, err := convertProtoArguments(args)
	if err != nil {
		return nil, EvaluationFailure.Wrapf(err, "failed to call function '%s'", name)
	}
	e, err := treeFunctionEvaluator(env, tree, name, arguments)
	if err != nil {
		return nil, EvaluationFailure.Wrapf(err, "failed to call function '%s'", name)
	}
	rideResult, err := e.evaluate()
	if err != nil {
		return nil, err
	}
	dAppResult, ok := rideResult.(DAppResult)
	if !ok {
		return rideResult, EvaluationFailure.Errorf("invalid result of call function '%s'", name)
	}
	if tree.LibVersion < 5 {
		return rideResult, nil
	}
	// Add actions and complexity from wrapped state
	ws, ok := env.state().(*WrappedState)
	if !ok {
		return nil, EvaluationFailure.New("not a wrapped state")
	}
	dAppResult.complexity += ws.totalComplexity
	if ws.act == nil { // No additional actions in wrapped state
		return rideResult, nil
	}
	// Append actions of the original call to the end of actions collected in wrapped state
	dAppResult.actions = append(ws.act, dAppResult.actions...)
	return dAppResult, nil
}
