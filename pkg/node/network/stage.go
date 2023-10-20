package network

//go:generate stringer -type stage -trimprefix stage
type stage int

const (
	stageDisconnected stage = iota
	stageGroup
	stageLeader
	stageHalt
)
