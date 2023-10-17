package network

//go:generate stringer -type state -trimprefix state
type state int

const (
	stateDisconnected state = iota
	stateGroup
	stateLeader
	stateHalt
)
