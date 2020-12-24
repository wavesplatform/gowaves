package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func CallVerifier(env RideEnvironment, tree *Tree) (RideResult, error) {
	e, err := treeVerifierEvaluator(env, tree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call verifier")
	}
	return e.evaluate()
}

func invokeFunctionFromDApp(env RideEnvironment, recipient proto.Recipient, fnName rideString, listArgs rideList) (RideResult, error) {

	address, err := env.state().NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, errors.Errorf("cannot get address from dApp, invokeFunctionFromDApp")
	}
	env.setNewDAppAddress(*address)

	newScript, err := env.state().GetByteTree(recipient)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get script by recipient")
	}

	tree, err := Parse(newScript)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get tree by script")
	}

	e, err := treeFunctionEvaluatorForInvokeDAppFromDApp(env, tree, string(fnName), listArgs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", fnName)
	}
	return e.evaluate()
}

func CallFunction(env RideEnvironment, tree *Tree, name string, args proto.Arguments) (RideResult, error) {
	if name == "" {
		name = "default"
	}
	e, err := treeFunctionEvaluator(env, tree, name, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", name)
	}
	return e.evaluate()
}
