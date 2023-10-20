package node

//go:generate stringer -type stage -trimprefix stage
type stage int

const (
	stageIdle stage = iota
	stageOperation
	stageOperationNG
	stageSync
	stagePersistence
	stageHalt
)
