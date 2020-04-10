package api

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestApp_PeersAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := mock.NewMockState(ctrl)
	s.EXPECT().Peers().Return([]proto.TCPAddr{proto.NewTCPAddrFromString("127.0.0.1:6868")}, nil)

	app, err := NewApp("key", nil, services.Services{State: s})
	require.NoError(t, err)

	rs2, err := app.PeersAll()
	require.NoError(t, err)
	require.Len(t, rs2.Peers, 1)
}
