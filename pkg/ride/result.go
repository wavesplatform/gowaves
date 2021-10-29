package ride

import "github.com/wavesplatform/gowaves/pkg/proto"

type Result interface {
	Result() bool
	UserError() string
	userResult() rideType
	ScriptActions() []proto.ScriptAction
	Complexity() int
}

type ScriptResult struct {
	res        bool
	msg        string
	param      rideType
	complexity int
}

func (r ScriptResult) Result() bool {
	return r.res
}

func (r ScriptResult) userResult() rideType {
	return r.param
}

func (r ScriptResult) UserError() string {
	return r.msg
}

func (r ScriptResult) ScriptActions() []proto.ScriptAction {
	return nil
}

func (r ScriptResult) Complexity() int {
	return r.complexity
}

type DAppResult struct {
	res        bool // true - success, false - call failed, read msg
	actions    []proto.ScriptAction
	msg        string
	param      rideType
	complexity int
}

func (r DAppResult) Result() bool {
	return r.res
}

func (r DAppResult) userResult() rideType {
	return r.param
}

func (r DAppResult) UserError() string {
	return r.msg
}

func (r DAppResult) ScriptActions() []proto.ScriptAction {
	return r.actions
}

func (r DAppResult) Complexity() int {
	return r.complexity
}
