package server

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	stateConnecting = "connecting"
	stateConnected  = "connected"
)

type NodeState struct {
	State          string           `json:"state"`
	Addr           string           `json:"addr"`
	LastKnownBlock crypto.Signature `json:"last_known_block"`
	KnownVersion   proto.Version    `json:"known_versoin"`
}
