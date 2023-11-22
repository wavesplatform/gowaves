package network

//go:generate stringer -type event -trimprefix event
type event int

const (
	eventPeerConnected event = iota
	eventPeerDisconnected
	eventScore
	eventGetPeers
	eventPeers
	eventFollowGroup
	eventFollowLeader
	eventBlacklistPeer
	eventSuspendPeer
	eventBroadcastTransaction
	eventQuorumChanged
	eventCheckQuorum
	eventFollowingModeChanged
	eventScoreUpdated
	eventAskPeers
	eventAnnounceScore
	eventBroadcastMicroBlockInv
	eventBroadcastBlock
	eventHalt
)
