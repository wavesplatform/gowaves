package fride

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

type DAppResult []proto.ScriptAction

func (r DAppResult) Result() bool {
	return true
}

func (r DAppResult) UserError() string {
	return ""
}

func (r DAppResult) ScriptActions() []proto.ScriptAction {
	return r
}
