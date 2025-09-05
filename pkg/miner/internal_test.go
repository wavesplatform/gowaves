package miner

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func TestMineBlock(t *testing.T) {
	doTest := func(t *testing.T, m state.State) {
		const scheme = proto.TestNetScheme

		nxt := proto.NxtConsensus{
			BaseTarget:   5767,
			GenSignature: crypto.MustDigestFromBase58("EijBTmUp8j1VRm8542zBii1BdYHvZ26iDk1hLup8kZTP").Bytes(),
		}
		kp, err := proto.NewKeyPair([]byte("abc"))
		require.NoError(t, err)
		const parentSigB58 = "4f6Nkihj7j3t2ohNPk69MUZzpdHHwXG9hM2qjgeRmKmDPFiRYeedv6ewc9dhvNo1BxvE5CTgTjTTyAYPfR42eBXP"
		parentSig := crypto.MustSignatureFromBase58(parentSigB58)
		parent := proto.NewBlockIDFromSignature(parentSig)
		b, err := mineKeyBlock(m, 4, nxt, kp, []settings.Feature{13, 14}, 1581610238465, parent, 600000000, scheme)
		require.NoError(t, err)

		bts, err := b.MarshalBinary(scheme)
		require.NoError(t, err)

		require.True(t, crypto.Verify(kp.Public, b.BlockSignature, bts[:len(bts)-64]))
		require.Equal(t, []int16{13, 14}, b.Features)
		require.EqualValues(t, 600000000, b.RewardVote)
	}
	t.Run("BeforeLightNode", func(t *testing.T) {
		m := mock.NewMockState(gomock.NewController(t))

		m.EXPECT().Height().Return(proto.Height(42), nil).Times(1)
		m.EXPECT().IsActiveLightNodeNewBlocksFields(proto.Height(43)).Return(false, nil).Times(1)

		doTest(t, m)
	})
	t.Run("AfterLightNode", func(t *testing.T) {
		m := mock.NewMockState(gomock.NewController(t))

		m.EXPECT().Height().Return(proto.Height(42), nil).Times(1)
		m.EXPECT().IsActiveLightNodeNewBlocksFields(proto.Height(43)).Return(true, nil).Times(1)
		m.EXPECT().CreateNextSnapshotHash(gomock.AssignableToTypeOf(&proto.Block{})).
			Return(crypto.MustDigestFromBase58("EijBTmUp8j1VRm8542zBii1BdYHvZ26iDk1hLup8kZTP"), nil).Times(1)

		doTest(t, m)
	})
}
