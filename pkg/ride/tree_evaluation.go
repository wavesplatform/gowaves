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

func invokeFunctionFromDApp(env RideEnvironment, recipient proto.Recipient, fnName rideType, args []rideType) (RideResult, error) {
	var funcName string
	switch fn := fnName.(type) {
	case rideString:
		funcName = string(fn)
	default:
		return nil, errors.Errorf("wrong function name argument type %T", fnName)
	}

	newScript, err := env.state().GetByteTree(recipient)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get script by recipient")
	}

	tree, err := Parse(newScript)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get tree by script")
	}

	e, err := treeFunctionEvaluatorForInvokeDAppFromDApp(env, tree, funcName, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", funcName)
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
