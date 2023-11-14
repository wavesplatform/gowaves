package node

//go:generate stringer -type event -trimprefix event
type event int

const (
	eventTransaction event = iota
	eventBlock
	eventBlockIDs
	eventMicroBlockInv
	eventGetMicroBlock
	eventMicroBlock
	eventChangeSyncPeer
	eventResume
	eventSuspend
	eventBlockGenerated
	eventPersistenceRequired
	eventPersistenceComplete
	eventAbortSync
	eventBlockSequenceComplete
	eventBroadcastTransaction
	eventHalt
)
