package state_fsm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// check it has no action
func TestIdleFsm_MicroBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	def := NewMockDefault(ctrl)
	fakeCh := make(chan proto.PeerMessageIDs, 1)
	defer close(fakeCh)
	idle := NewIdleFsm(BaseInfo{d: def, excludeListCh: fakeCh})
	def.EXPECT().Noop(gomock.Any())
	_, _, _ = idle.MicroBlock(nil, nil)
}

// check it just call default
func TestIdleFsm_MicroBlockInv(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	def := NewMockDefault(ctrl)
	fakeCh := make(chan proto.PeerMessageIDs, 1)
	defer close(fakeCh)
	idle := NewIdleFsm(BaseInfo{d: def, excludeListCh: fakeCh})

	def.EXPECT().Noop(gomock.Any())
	_, _, _ = idle.MicroBlockInv(nil, nil)
}

// check it just call default
func TestIdleFsm_Signatures(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	def := NewMockDefault(ctrl)
	fakeCh := make(chan proto.PeerMessageIDs, 1)
	defer close(fakeCh)
	idle := NewIdleFsm(BaseInfo{d: def, excludeListCh: fakeCh})

	def.EXPECT().Noop(gomock.Any())
	_, _, _ = idle.BlockIDs(nil, nil)
}

// check it just call default
func TestIdleFsm_PeerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	def := NewMockDefault(ctrl)
	fakeCh := make(chan proto.PeerMessageIDs, 1)
	defer close(fakeCh)
	idle := NewIdleFsm(BaseInfo{d: def, excludeListCh: fakeCh})

	def.EXPECT().Noop(gomock.Any())
	_, _, _ = idle.BlockIDs(nil, nil)
}
