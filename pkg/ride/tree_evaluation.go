package ride

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type InternalInvokeResult struct {
	result               RideResult
	isDAppFromDappInvoke bool
}

func CallVerifier(env RideEnvironment, tree *Tree) (RideResult, error) {
	var invoke InternalInvokeResult
	e, err := treeVerifierEvaluator(env, tree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call verifier")
	}
	return e.evaluate(&invoke)
}

func invokeFunctionFromDApp(env RideEnvironment, recipient proto.Recipient, fnName rideType, args []rideType, invoke *InternalInvokeResult) (RideResult, error) {
	var tree *Tree // TODO get tree from address
	if recipient.Address != nil {
		fmt.Println("dump")
		// tree = getTree(recipient.Address)
	}

	var funcName string
	switch fn := fnName.(type) {
	case rideString:
		funcName = string(fn)
	default:
		return nil, errors.Errorf("wrong function name argument type %T", fnName)
	}

	e, err := treeFunctionEvaluatorForInvokeDAppFromDApp(env, tree, funcName, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", funcName)
	}
	return e.evaluate(invoke)
}

func CallFunction(env RideEnvironment, tree *Tree, name string, args proto.Arguments) (RideResult, error) {
	var invoke InternalInvokeResult
	if name == "" {
		name = "default"
	}
	e, err := treeFunctionEvaluator(env, tree, name, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", name)
	}
	r, err := e.evaluate(&invoke)
	if invoke.isDAppFromDappInvoke {
		return invoke.result, err
	}
	return r, err
}
