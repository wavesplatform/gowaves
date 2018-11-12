package server

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	stateSyncing = iota
	stateUpdating
)

type NodeState struct {
	State          uint8         `json:"state"`
	Addr           string        `json:"addr"`
	LastKnownBlock proto.BlockID `json:"last_known_block"`
	KnownVersion   proto.Version `json:"known_versoin"`

	pendingBlocksHave map[proto.BlockID]bool
	pendingSignatures []proto.BlockID
}
