package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func CallVerifier2(env RideEnvironment, tree *Tree) (RideResult, error) {
	e, err := treeVerifierEvaluator(env, tree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call verifier")
	}
	return e.evaluate()
}

func CallVerifier3(env RideEnvironment, tree *Tree) (RideResult, error) {
	compiled, err := CompileSimpleScript(tree)
	if err != nil {
		return nil, errors.Wrap(err, "call compile script")
	}
	return compiled.Run(env)
}

func CallVerifier(env RideEnvironment, tree *Tree) (RideResult, error) {
	//r, err := CallVerifier3(env, tree)
	//if err != nil {
	//	return nil, err
	//}

	r, err := CallVerifier2(env, tree)
	if err != nil {
		return nil, err
	}
	//if !r.Eq(r2) {
	//	return nil, errors.New("R1 != R2: failed to call account script on transaction ")
	//}
	return r, nil
}

func CallFunction(env RideEnvironment, tree *Tree, name string, args proto.Arguments) (RideResult, error) {
	panic("CallFunction called")
	if name == "" {
		name = "default"
	}
	e, err := treeFunctionEvaluator(env, tree, name, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", name)
	}
	return e.evaluate()
}
