package ride

import "github.com/wavesplatform/gowaves/pkg/proto"

type Result interface {
	Result() bool
	ScriptActions() []proto.ScriptAction
	Complexity() int
	userResult() rideType
}

type ScriptResult struct {
	res        bool
	param      rideType
	complexity int
}

func (r ScriptResult) Result() bool {
	return r.res
}

func (r ScriptResult) userResult() rideType {
	return r.param
}

func (r ScriptResult) ScriptActions() []proto.ScriptAction {
	return nil
}

func (r ScriptResult) Complexity() int {
	return r.complexity
}

type DAppResult struct {
	actions    []proto.ScriptAction
	param      rideType
	complexity int
}

func (r DAppResult) Result() bool {
	return true
}

func (r DAppResult) userResult() rideType {
	return r.param
}

func (r DAppResult) ScriptActions() []proto.ScriptAction {
	return r.actions
}

func (r DAppResult) Complexity() int {
	return r.complexity
}
