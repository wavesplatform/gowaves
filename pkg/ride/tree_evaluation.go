package ride

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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

func CallVmVerifier(txID string, env RideEnvironment, compiled *Executable) (RideResult, error) {
	if env == nil {
		return nil, errors.Errorf("env is nil")
	}
	return compiled.Verify(env)
}

func CallVerifier(txID string, env RideEnvironment, tree *Tree, exe *Executable) (RideResult, error) {
	r, err := CallVmVerifier(txID, env, exe)
	if err != nil {
		return nil, errors.Wrap(err, "vm verifier")
	}
	r2, err := CallTreeVerifier(env, tree)
	if err != nil {
		return nil, errors.Wrap(err, "tree verifier")
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
				zap.S().Error(i, txID, " ", c1[i])
			} else {
				zap.S().Error(i, txID, " ", "<empty>")
			}
			if i <= len(c2)-1 {
				zap.S().Error(i, txID, " ", c2[i])
			} else {
				zap.S().Error(i, txID, " ", "<empty>")
			}
		}

		return nil, errors.New("R1 != R2: failed to call account script on transaction ")
	}

	return r, nil
}

func CallTreeFunction(txID string, env RideEnvironment, tree *Tree, name string, args proto.Arguments) (RideResult, error) {
	if name == "" {
		name = "default"
	}
	e, err := treeFunctionEvaluator(env, tree, name, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call function '%s'", name)
	}
	return e.evaluate()
}

func CallFunction(txID string, env RideEnvironment, exe *Executable, tree *Tree, name string, args proto.Arguments) (RideResult, error) {
	rs1, err := CallTreeFunction(txID, env, tree, name, args)
	if err != nil {
		return nil, errors.Wrap(err, "call function by tree")
	}
	rs2, err := CallVmFunction(txID, env, exe, name, args)
	if err != nil {
		return rs2, errors.Wrap(err, "call function by vm")
	}
	if !rs1.Eq(rs2) {
		c1 := rs1.Calls()
		c2 := rs2.Calls()
		max := len(c1)
		if len(c2) > len(c1) {
			max = len(c2)
		}
		for i := 0; i < max; i++ {
			if i <= len(c1)-1 {
				zap.S().Error(i, txID, " ", c1[i])
			} else {
				zap.S().Error(i, txID, " ", "<empty>")
			}
			if i <= len(c2)-1 {
				zap.S().Error(i, txID, " ", c2[i])
			} else {
				zap.S().Error(i, txID, " ", "<empty>")
			}
		}

		ac1 := rs1.ScriptActions()
		ac2 := rs2.ScriptActions()
		for i := range ac1 {
			zap.S().Errorf("%d %s Action %+v", i, txID, ac1[i].(*proto.DataEntryScriptAction).Entry.(*proto.BinaryDataEntry).Value)
			zap.S().Errorf("%d %s Action %+v", i, txID, ac2[i].(*proto.DataEntryScriptAction).Entry.(*proto.BinaryDataEntry).Value)
			zap.S().Errorf("Eq %+v", assert.ObjectsAreEqual(ac1[i].(*proto.DataEntryScriptAction).Entry.(*proto.BinaryDataEntry).Value, ac2[i].(*proto.DataEntryScriptAction).Entry.(*proto.BinaryDataEntry).Value))
			break
			//zap.S().Errorf(i, txID, " Action ", ac2[i])
		}

		return nil, errors.New("R1 != R2: failed to call account script on transaction ")
	}
	return rs2, nil
}

func CallVmFunction(txID string, env RideEnvironment, e *Executable, name string, args proto.Arguments) (RideResult, error) {
	if name == "" {
		name = "default"
	}
	entry, err := e.Entrypoint(name)
	if err != nil {
		return nil, err
	}
	if l := len(args); l != int(entry.argn) {
		return nil, errors.Errorf("invalid arguments count %d for function '%s'", l, name)
	}
	applyArgs := make([]rideType, 0, len(args))
	for _, arg := range args {
		a, err := convertArgument(arg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to call function '%s'", name)
		}
		applyArgs = append(applyArgs, a)
	}
	return e.Invoke(env, name, applyArgs)
}
