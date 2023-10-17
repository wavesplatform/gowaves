package node

//go:generate stringer -type state -trimprefix state
type state int

const (
	stateIdle state = iota
	stateOperation
	stateOperationNG
	stateSync
	statePersistence
	stateHalt
)
