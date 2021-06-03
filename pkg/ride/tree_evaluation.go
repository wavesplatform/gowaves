package ride

import (
	"fmt"
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

	if rideResult != nil {
		for _, action := range DAppResult.actions {
			switch res := action.(type) {

			case *proto.DataEntryScriptAction:
				switch dataEntry := res.Entry.(type) {

				case *proto.IntegerDataEntry:
					fmt.Printf("it's integer data entry with value : %d and key: %s\n", dataEntry.Value, dataEntry.Key)
				}

			case *proto.TransferScriptAction:
				fmt.Printf("it's transfer action  with value : %d and asset ID: %s, and recipient address is %s\n", res.Amount, res.Asset.ID.String(), res.Recipient.Address.String())
				if res.Sender != nil {
					fmt.Printf("sender of transfer is %s\n", res.Sender.String())
				}

			}
		}
	}

	fmt.Printf("RideResult is %t\n", DAppResult.res)

	return DAppResult, err
}
