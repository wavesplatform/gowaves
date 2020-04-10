package api

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestApp_BlocksFirst(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	g := &proto.Block{
		BlockHeader: proto.BlockHeader{
			BlockSignature: crypto.MustSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa"),
		},
	}

	s := mock.NewMockState(ctrl)
	s.EXPECT().BlockByHeight(proto.Height(1)).Return(g, nil)

	app, err := NewApp("api-key", nil, services.Services{State: s})
	require.NoError(t, err)
	first, err := app.BlocksFirst()
	require.NoError(t, err)
	require.EqualValues(t, 1, first.Height)
}
func TestApp_BlocksLast(t *testing.T) {
	g := &proto.Block{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := mock.NewMockState(ctrl)
	s.EXPECT().Height().Return(proto.Height(1), nil)
	s.EXPECT().BlockByHeight(proto.Height(1)).Return(g, nil)

	app, err := NewApp("api-key", nil, services.Services{State: s})
	require.NoError(t, err)
	first, err := app.BlocksLast()
	require.NoError(t, err)
	require.EqualValues(t, 1, first.Height)
}
