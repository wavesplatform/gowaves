package ride

import (
	"fmt"
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

func invokeFunctionFromDApp(rideEnv RideEnvironment, tree *Tree, args proto.Arguments) (bool, []proto.ScriptAction, error) {
	binaryAddr, err := args[0].MarshalBinary()
	if err != nil {
		return false, nil, errors.Errorf("Failed to marshal binaryAddr")
	}

	binaryFunctionNameFromDApp, err := args[1].MarshalBinary()
	if err != nil {
		return false, nil, errors.Errorf("Failed to marshal fn name")
	}

	scriptAddress, err := proto.NewAddressFromBytes(binaryAddr)
	if err != nil {
		return false, nil, errors.Errorf("Failed to get dApp adress from bytes")
	}

	newFn := proto.FunctionCall{Name: string(binaryFunctionNameFromDApp), Arguments: args[2:]}

	//invoke := Invoke{dAppAddress: scriptAddress, function: , , args: argsForFnFromDApp}

	return rideEnv.state().InvokeFunctionFromDApp(scriptAddress, newFn)
}

func CallFunction(env RideEnvironment, tree *Tree, name string, args proto.Arguments) (RideResult, error) {

	if name == "callDApp" {
		ok, res, err := invokeFunctionFromDApp(env, tree, args)
		if err != nil {
			fmt.Println(ok, res)
		}
		// TODO
	}
	if name == "" {
		name = "default"
	}
	e, err := treeFunctionEvaluator(env, tree, name, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", name)
	}
	return e.evaluate()
}
