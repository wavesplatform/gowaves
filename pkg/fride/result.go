package fride

type RideResult interface {
	_rideResult()
}

type ScriptResult bool

func (r ScriptResult) _rideResult() {}
