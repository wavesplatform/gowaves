package fride

import "github.com/wavesplatform/gowaves/pkg/proto"

type RideResult interface {
	_rideResult()
}

type ScriptResult bool

func (r ScriptResult) _rideResult() {}

type DAppResult []proto.ScriptAction

func (r DAppResult) _rideResult() {}
