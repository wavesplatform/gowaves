package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

func CallTreeVerifier(env RideEnvironment, tree *Tree) (RideResult, error) {
	e, err := treeVerifierEvaluator(env, tree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call verifier")
	}
	return e.evaluate()
}

func CallVerifier3(txID string, env RideEnvironment, tree *Tree) (RideResult, error) {
	compiled, err := CompileVerifier(txID, tree)
	if err != nil {
		return nil, errors.Wrap(err, "call compile script")
	}
	if env == nil {
		return nil, errors.Errorf("env is nil")
	}
	return compiled.Run(env, []rideType{env.transaction()})
}

func CallVerifier(txID string, env RideEnvironment, tree *Tree) (RideResult, error) {
	r, err := CallVerifier3(txID, env, tree)
	if err != nil {
		return nil, err
	}

	r2, err := CallTreeVerifier(env, tree)
	if err != nil {
		return nil, err
	}
	if !r.Eq(r2) {
		c1 := r.Calls()
		c2 := r2.Calls()
		max := len(c1)
		if len(c2) > len(c1) {
			max = len(c2)
		}
		for i := 0; i < max; i++ {
			//zap.S().Error("R1 != R2: failed to call account script on transaction ")
			if i <= len(c1)-1 {
				zap.S().Error(i, " ", c1[i])
			} else {
				zap.S().Error(i, " ", "<empty>")
			}
			if i <= len(c2)-1 {
				zap.S().Error(i, " ", c2[i])
			} else {
				zap.S().Error(i, " ", "<empty>")
			}
		}

		return nil, errors.New("R1 != R2: failed to call account script on transaction ")
	}
	return r, nil
}

func CallTreeFunction(env RideEnvironment, tree *Tree, name string, args proto.Arguments) (RideResult, error) {
	if name == "" {
		name = "default"
	}
	e, err := treeFunctionEvaluator(env, tree, name, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", name)
	}
	return e.evaluate()
}

func CallFunction(txID string, env RideEnvironment, tree *Tree, name string, args proto.Arguments) (RideResult, error) {
	rs1, err := CallTreeFunction(env, tree, name, args)
	if err != nil {
		return nil, errors.Wrap(err, "call function by tree")
	}
	rs2, err := CallFunction2(txID, env, tree, name, args)
	if err != nil {
		return rs2, errors.Wrap(err, "call function by vm")
	}
	if !rs1.Eq(rs2) {
		zap.S().Errorf("%s, result mismatch", txID)
		zap.S().Errorf("tree: %+q", rs1)
		zap.S().Errorf("vm  : %+q", rs2)
		return nil, errors.New(txID + ": result mismatch")
	}
	return rs2, nil
}

func CallFunction2(txID string, env RideEnvironment, tree *Tree, name string, args proto.Arguments) (RideResult, error) {
	if name == "" {
		name = "default"
	}
	f, numArgs, err := CompileFunction(txID, tree, name, args, tree.IsDApp())
	if err != nil {
		return nil, err
	}
	if l := len(args); l != numArgs {
		return nil, errors.Errorf("invalid arguments count %d for function '%s'", l, name)
	}
	applyArgs := make([]rideType, 0, len(args)+1)
	applyArgs = append(applyArgs, env.invocation())
	for _, arg := range args {
		a, err := convertArgument(arg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to call function '%s'", name)
		}
		//s.pushValue(function.Arguments[i], a)
		applyArgs = append(applyArgs, a)
		//namedArgument{
		//	name: function.Arguments[i],
		//	arg:  a,
		//})
	}
	return f.Run(env, applyArgs)
}
