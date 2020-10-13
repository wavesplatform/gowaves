package ride

import "github.com/wavesplatform/gowaves/pkg/proto"

type RideResult interface {
	Result() bool
	UserError() string
	ScriptActions() []proto.ScriptAction
}

type ScriptResult struct {
	res bool
	msg string
}

func (r ScriptResult) Result() bool {
	return r.res
}

func (r ScriptResult) UserError() string {
	return r.msg
}

func (r ScriptResult) ScriptActions() []proto.ScriptAction {
	return nil
}

type DAppResult struct {
	res     bool // true - success, false - call failed, read msg
	actions []proto.ScriptAction
	msg     string
}

func (r DAppResult) Result() bool {
	return r.res
}

func (r DAppResult) UserError() string {
	return r.msg
}

func (r DAppResult) ScriptActions() []proto.ScriptAction {
	return r.actions
}
