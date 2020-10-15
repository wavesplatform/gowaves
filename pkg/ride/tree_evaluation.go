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
