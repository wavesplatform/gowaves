package ride

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
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
	// After that instruction script/function is executed,
	// so result of the execution and spent complexity should be considered outside.
	rideResult, err := e.evaluate()
	if err != nil {
		// Evaluation failed we have to return a DAppResult that contains spent execution complexity
		// Produced actions are not stored for failed transactions, no need to return them here
		switch et := err.(type) {
		case evaluationError:
			switch et.errorType {
			case UserError, RuntimeError, InternalInvocationError:
				return DAppResult{complexity: e.complexity + wrappedStateComplexity(env.state()), err: err}, nil
			default:
				return DAppResult{complexity: e.complexity + wrappedStateComplexity(env.state())}, err
			}
		default:
			return DAppResult{complexity: e.complexity + wrappedStateComplexity(env.state())}, err
		}
	}
	dAppResult, ok := rideResult.(DAppResult)
	if !ok { // Unexpected result type
		return DAppResult{complexity: e.complexity + wrappedStateComplexity(env.state())}, EvaluationFailure.Errorf("invalid result of call function '%s'", name)
	}
	if tree.LibVersion < 5 { // Shortcut because no wrapped state before version 5
		return rideResult, nil
	}
	// Add actions and complexity from wrapped state
	// Append actions of the original call to the end of actions collected in wrapped state
	dAppResult.complexity += wrappedStateComplexity(env.state())
	dAppResult.actions = append(wrappedStateActions(env.state()), dAppResult.actions...)
	return dAppResult, nil
}

func wrappedStateComplexity(state types.SmartState) int {
	ws, ok := state.(*WrappedState)
	if !ok {
		return 0
	}
	return ws.totalComplexity
}

func wrappedStateActions(state types.SmartState) []proto.ScriptAction {
	ws, ok := state.(*WrappedState)
	if !ok {
		return nil
	}
	return ws.act
}
