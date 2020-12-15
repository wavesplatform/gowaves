package ride

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type RideResult interface {
	Result() bool
	UserError() string
	ScriptActions() proto.ScriptActions
	Eq(RideResult) bool
	Calls() []callLog
}

type ScriptResult struct {
	res        bool
	msg        string
	operations int
	calls      []callLog
}

func (r ScriptResult) Result() bool {
	return r.res
}

func (r ScriptResult) Calls() []callLog {
	return r.calls
}

func (r ScriptResult) UserError() string {
	return r.msg
}

func (r ScriptResult) ScriptActions() proto.ScriptActions {
	return nil
}

func (r ScriptResult) Eq(other RideResult) bool {
	switch a := other.(type) {
	case ScriptResult:
		return a.res == r.res && a.msg == r.msg
	default:
		return false
	}
}

type DAppResult struct {
	res        bool // true - success, false - call failed, read msg
	actions    proto.ScriptActions
	msg        string
	operations int
	calls      []callLog
}

func (r DAppResult) Result() bool {
	return r.res
}

func (r DAppResult) Calls() []callLog {
	return r.calls
}

func (r DAppResult) UserError() string {
	return r.msg
}

func (r DAppResult) ScriptActions() proto.ScriptActions {
	return r.actions
}

func (r DAppResult) Eq(other RideResult) bool {
	switch v := other.(type) {
	case DAppResult:
		return r.res == v.res && assert.ObjectsAreEqual(r.actions, v.actions)
	default:
		return false
	}
}
