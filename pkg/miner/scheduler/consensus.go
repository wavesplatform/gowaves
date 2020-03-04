package scheduler

type connectedCount interface {
	ConnectedCount() int
}

type MinerConsensusImpl struct {
	count                 connectedCount
	atLeastConnectedPeers int
}

func NewMinerConsensus(c connectedCount, atLeastConnectedPeers int) MinerConsensusImpl {
	return MinerConsensusImpl{
		count:                 c,
		atLeastConnectedPeers: atLeastConnectedPeers,
	}
}

func (a MinerConsensusImpl) IsMiningAllowed() bool {
	return a.count.ConnectedCount() >= a.atLeastConnectedPeers
}

type StubConsensus struct {
}

func (s StubConsensus) IsMiningAllowed() bool {
	return true
}
