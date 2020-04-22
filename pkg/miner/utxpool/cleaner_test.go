package utxpool

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNewCleaner(t *testing.T) {
	require.NotNil(t, NewCleaner(nil, nil, nil))
}

func TestCleaner_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockstateWrapper(ctrl)
	m.EXPECT().Height().Return(uint64(0), errors.New("some err"))

	c := newCleaner(m, noOnBulkValidator{})
	c.Clean()
}
