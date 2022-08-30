package ride

import "github.com/wavesplatform/gowaves/pkg/proto"

type Result interface {
	Result() bool
	ScriptActions() []proto.ScriptAction
	Complexity() int
	userResult() rideType
}

type scriptExecutionResult struct {
	Res             bool
	SpentComplexity int
	param           rideType
}

func (r scriptExecutionResult) Result() bool {
	return r.Res
}

func (r scriptExecutionResult) userResult() rideType {
	return r.param
}

func (r scriptExecutionResult) ScriptActions() []proto.ScriptAction {
	return nil
}

func (r scriptExecutionResult) Complexity() int {
	return r.SpentComplexity
}

type dAppResult struct {
	Actions         []proto.ScriptAction
	SpentComplexity int
	param           rideType
}

func (r dAppResult) Result() bool {
	return true
}

func (r dAppResult) userResult() rideType {
	return r.param
}

func (r dAppResult) ScriptActions() []proto.ScriptAction {
	return r.Actions
}

func (r dAppResult) Complexity() int {
	return r.SpentComplexity
}
