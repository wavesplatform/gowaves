package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func CallVerifier(env Environment, tree *Tree) (Result, error) {
	e, err := treeVerifierEvaluator(env, tree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call verifier")
	}
	return e.evaluate()
}

func invokeFunctionFromDApp(env Environment, recipient proto.Recipient, fnName rideString, listArgs rideList) (Result, error) {
	newScript, err := env.state().GetByteTree(recipient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get script by recipient")
	}

	tree, err := Parse(newScript)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tree by script")
	}
	if tree.LibVersion < 5 {
		return nil, errors.Errorf("failed to call 'invoke' for script with version %d. Scripts with version 5 are only allowed to be used in 'invoke'", tree.LibVersion)
	}

	e, err := treeFunctionEvaluatorForInvokeDAppFromDApp(env, tree, string(fnName), listArgs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", fnName)
	}
	return e.evaluate()
}

func CallFunction(env Environment, tree *Tree, name string, args proto.Arguments) (Result, error) {
	if name == "" {
		name = "default"
	}
	e, err := treeFunctionEvaluator(env, tree, name, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", name)
	}
	rideResult, err := e.evaluate()

	DAppResult, ok := rideResult.(DAppResult)
	if !ok {
		return rideResult, err
	}
	if tree.LibVersion < 5 {
		return rideResult, err
	}

	ws, ok := env.state().(*WrappedState)
	if !ok {
		return nil, errors.New("wrong state")
	}

	complexity, ok := ws.checkTotalComplexity()
	if !ok {
		return nil, errors.Errorf("complexity of invocation chain %d exceeds maximum allowed complexity of %d", complexity, MaxChainInvokeComplexity)
	}

	if ws.act == nil {
		return rideResult, err
	}

	fullActions := ws.act
	fullActions = append(fullActions, DAppResult.actions...)
	DAppResult.actions = fullActions
	return DAppResult, err
}
