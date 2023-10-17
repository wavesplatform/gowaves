package node

//go:generate stringer -type event -trimprefix event
type event int

const (
	eventTransaction event = iota
	eventGetBlock
	eventBlock
	eventGetBlockIDs
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
	eventSyncTimeout
	eventBroadcastTransaction
	eventHalt
)
