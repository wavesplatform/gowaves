package miner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func TestMineBlock(t *testing.T) {
	const scheme = proto.TestNetScheme

	nxt := proto.NxtConsensus{
		BaseTarget:   5767,
		GenSignature: crypto.MustDigestFromBase58("EijBTmUp8j1VRm8542zBii1BdYHvZ26iDk1hLup8kZTP").Bytes(),
	}
	kp, err := proto.NewKeyPair([]byte("abc"))
	require.NoError(t, err)
	parentSig := crypto.MustSignatureFromBase58("4f6Nkihj7j3t2ohNPk69MUZzpdHHwXG9hM2qjgeRmKmDPFiRYeedv6ewc9dhvNo1BxvE5CTgTjTTyAYPfR42eBXP")
	parent := proto.NewBlockIDFromSignature(parentSig)
	b, err := MineBlock(4, nxt, kp, []settings.Feature{13, 14}, 1581610238465, parent, 600000000, scheme)
	require.NoError(t, err)

	bts, err := b.MarshalBinary(scheme)
	require.NoError(t, err)

	require.True(t, crypto.Verify(kp.Public, b.BlockSignature, bts[:len(bts)-64]))
	require.Equal(t, []int16{13, 14}, b.Features)
	require.EqualValues(t, 600000000, b.RewardVote)
}
