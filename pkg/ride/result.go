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
}

type ScriptResult struct {
	res        bool
	msg        string
	operations int
}

func (r ScriptResult) Result() bool {
	return r.res
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
}

func (r DAppResult) Result() bool {
	return r.res
}

func (r DAppResult) UserError() string {
	return r.msg
}

func (r DAppResult) ScriptActions() proto.ScriptActions {
	return r.actions
}

func (r DAppResult) Eq(other RideResult) bool {
	switch other.(type) {
	case DAppResult:
		return assert.ObjectsAreEqual(r, other)
	default:
		return false
	}
}
