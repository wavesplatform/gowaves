package blocks_applier

import (
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/state/stateerr"
)

func notFound() stateerr.StateError {
	return stateerr.NewStateError(stateerr.NotFoundError, proto.ErrNotFound)
}

type MockStateManager struct {
	state           []*proto.Block
	snapshots       []*proto.BlockSnapshot
	id2Block        map[proto.BlockID]*proto.Block
	Peers_          []proto.TCPAddr
	blockIDToHeight map[proto.BlockID]proto.Height
}

func (a *MockStateManager) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	if _, ok := a.blockIDToHeight[block.BlockID()]; ok {
		panic("duplicate block")
	}
	a.state = append(a.state, block)
	a.blockIDToHeight[block.BlockID()] = proto.Height(len(a.state))
	return block, nil
}

func NewMockStateManager(blocks ...*proto.Block) (*MockStateManager, error) {
	m := &MockStateManager{
		blockIDToHeight: make(map[proto.BlockID]proto.Height),
	}
	for _, b := range blocks {
		if _, err := m.AddDeserializedBlock(b); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (a *MockStateManager) Block(blockID proto.BlockID) (*proto.Block, error) {
	if block, ok := a.id2Block[blockID]; ok {
		return block, nil
	}
	return nil, notFound()
}

func (a *MockStateManager) BlockByHeight(height proto.Height) (*proto.Block, error) {
	if height > proto.Height(len(a.state)) {
		return nil, notFound()
	}
	return a.state[height-1], nil
}

func (a *MockStateManager) Height() (proto.Height, error) {
	return proto.Height(len(a.state)), nil
}

func (a *MockStateManager) BlockIDToHeight(blockID proto.BlockID) (uint64, error) {
	if height, ok := a.blockIDToHeight[blockID]; ok {
		return height, nil
	}
	return 0, notFound()
}

func (a *MockStateManager) RollbackToHeight(height uint64) error {
	if height > proto.Height(len(a.state)) {
		return notFound()
	}

	for i := proto.Height(len(a.state)); i > height; i-- {
		block := a.state[len(a.state)-1]
		a.state = a.state[:len(a.state)-1]
		delete(a.blockIDToHeight, block.BlockID())
		if len(a.snapshots) != 0 {
			a.snapshots = a.snapshots[:len(a.snapshots)-1]
		}
	}
	return nil
}

func (a *MockStateManager) CheckRollbackHeightAuto(height proto.Height) error {
	currentHeight, _ := a.Height()
	if height == 0 || height > currentHeight {
		return stateerr.NewStateError(stateerr.NotFoundError, proto.ErrNotFound)
	}
	return nil
}

func (a *MockStateManager) ScoreAtHeight(height uint64) (*big.Int, error) {
	if height > uint64(len(a.state)) {
		return nil, notFound()
	}
	score := big.NewInt(0)
	for _, b := range a.state[:height] {
		n, err := state.CalculateScore(b.BaseTarget)
		if err != nil {
			panic(err)
		}
		score.Add(score, n)
	}
	return score, nil
}

func (a *MockStateManager) CurrentScore() (*big.Int, error) {
	return a.ScoreAtHeight(proto.Height(len(a.state)))
}

func (a *MockStateManager) Close() error {
	panic("implement me")
}

func (a *MockStateManager) AddDeserializedBlocks(
	blocks []*proto.Block,
) (*proto.Block, error) {
	var out *proto.Block
	var err error
	for _, b := range blocks {
		if out, err = a.AddDeserializedBlock(b); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (a *MockStateManager) AddDeserializedBlocksWithSnapshots(
	blocks []*proto.Block,
	snapshots []*proto.BlockSnapshot,
) (*proto.Block, error) {
	var out *proto.Block
	var err error
	if len(blocks) != len(snapshots) {
		panic("the numbers of snapshots doesn't match the number of blocks")
	}
	for i, b := range blocks {
		if out, err = a.AddDeserializedBlock(b); err != nil {
			return nil, err
		}
		a.snapshots = append(a.snapshots, snapshots[i])
	}
	return out, nil
}

func (a *MockStateManager) IsActivated(_ int16) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) IsApproved(_ int16) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) StartProvidingExtendedApi() error {
	panic("implement me")
}

func (a *MockStateManager) SnapshotsAtHeight(h proto.Height) (proto.BlockSnapshot, error) {
	if h > proto.Height(len(a.snapshots)) {
		return proto.BlockSnapshot{}, notFound()
	}
	return *a.snapshots[h-1], nil
}
