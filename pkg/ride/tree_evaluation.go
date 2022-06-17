package ride

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// invokeCallComplexityV5 is invoke() or reentrantInvoke() functions cost for RideV5
const invokeCallComplexityV5 = 75

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
		return nil, handleComplexityInCaseOfEvaluationError(err, e, env)
	}
	dAppResult, ok := rideResult.(DAppResult)
	if !ok { // Unexpected result type
		return nil, EvaluationErrorAddComplexity(
			EvaluationFailure.Errorf("invalid result of call function '%s'", name),
			// New error, both complexities should be added (also see comment in complexityInCaseOfEvaluationError)
			e.complexity()+wrappedStateComplexity(env.state()),
		)
	}
	if tree.LibVersion < ast.LibV5 { // Shortcut because no wrapped state before version 5
		return rideResult, nil
	}
	maxChainInvokeComplexity, err := maxChainInvokeComplexityByVersion(tree.LibVersion)
	if err != nil {
		return nil, EvaluationFailure.Errorf("failed to get max chain invoke complexity: %v", err)
	}
	// Add actions and complexity from wrapped state
	// Append actions of the original call to the end of actions collected in wrapped state
	dAppResult.complexity += wrappedStateComplexity(env.state())
	if dAppResult.complexity > maxChainInvokeComplexity {
		return nil, EvaluationErrorAddComplexity(
			RuntimeError.Errorf("evaluation complexity %d exceeds %d limit for library version %d",
				dAppResult.complexity, maxChainInvokeComplexity, tree.LibVersion,
			),
			maxChainInvokeComplexity,
		)
	}
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

func evaluationErrorSetComplexity(err error, complexity int) error {
	if ee, ok := err.(evaluationError); ok {
		ee.spentComplexity = complexity
		return ee
	}
	return err
}

func handleComplexityInCaseOfEvaluationError(err error, e *treeEvaluator, env environment) error {
	// Error was not handled in wrapped state properly,
	// so we need to add complexities from current evaluation, from internal invokes and from internal failed invoke
	totalComplexity := e.complexity() + wrappedStateComplexity(env.state()) + EvaluationErrorSpentComplexity(err)
	switch et := GetEvaluationErrorType(err); et {
	case Undefined:
		return evaluationErrorSetComplexity(et.Wrap(err, "unhandled error"), totalComplexity)
	case InternalInvocationError:
		// reproduce scala's node buggy behaviour
		if ws, ok := env.state().(*WrappedState); ok && env.rideV5Activated() && !env.rideV6Activated() {
			// if invoke script tx depth level is 2 or less ==> complexity should be set to 0
			// invCount() is calls count of invoke() or reentrantInvoke() functions ==> txDepthLevel = 1 + invCount()
			if txDepthLevel := 1 + ws.invCount(); txDepthLevel <= 2 {
				totalComplexity = 0
			} else {
				// if depth level is 3 or greater, then we should sub last two invoke complexities plus
				// cost of the last invoke() or reentrantInvoke() function call
				totalComplexity -= ws.lastTwoInvokeComplexities.sum() + invokeCallComplexityV5
			}
		}
		return evaluationErrorSetComplexity(err, totalComplexity)
	default:
		return evaluationErrorSetComplexity(err, totalComplexity)
	}
}
