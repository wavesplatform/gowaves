package ride

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func CallVerifier(env environment, tree *ast.Tree) (Result, error) {
	e, err := treeVerifierEvaluator(env, tree)
	if err != nil {
		return nil, RuntimeError.Wrap(err, "failed to call verifier")
	}
	return e.evaluate()
}

func CallFunction(env environment, tree *ast.Tree, name string, args proto.Arguments) (Result, error) {
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
		et := GetEvaluationErrorType(err)
		if et == Undefined {
			return nil, EvaluationErrorSetComplexity(
				et.Wrap(err, "unhandled error"),
				// Error was not handled in wrapped state properly,
				// so we need to add both complexity from current evaluation and from internal invokes
				e.complexity(),
			)
		}
		return nil, EvaluationErrorSetComplexity(err, e.complexity())
	}
	dAppResult, ok := rideResult.(DAppResult)
	if !ok { // Unexpected result type
		return nil, EvaluationErrorSetComplexity(
			EvaluationFailure.Errorf("invalid result of call function '%s'", name),
			// New error, both complexities should be added
			e.complexity(),
		)
	}
	if tree.LibVersion < ast.LibV5 { // Shortcut because no wrapped state before version 5
		return rideResult, nil
	}
	// Add actions from wrapped state
	// Append actions of the original call to the end of actions collected in wrapped state
	dAppResult.actions = append(wrappedStateActions(env.state()), dAppResult.actions...)
	return dAppResult, nil
}

func wrappedStateActions(state types.SmartState) []proto.ScriptAction {
	ws, ok := state.(*WrappedState)
	if !ok {
		return nil
	}
	return ws.act
}
