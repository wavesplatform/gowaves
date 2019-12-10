package node

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func TestExpectedBlocks(t *testing.T) {

	s1, _ := crypto.NewSignatureFromBase58("22gRwjusnFYDoS31hRFEpFq21FjPCca2bUYtwicUH41GwzVkEAv7G22pAbRisu5s3bbhpzRRUpwF5png6ooKkb1n")
	s2, _ := crypto.NewSignatureFromBase58("3YzRwee4k7ddfXK9FtMtZs9V4r8sxThVLUAF6ATfz1Efrxv29CjoHnw2oCz8uvjFhgPMgrsKMmgSyVZ3nw5Hswme")

	ch := make(chan blockBytes, 2)
	e := newExpectedBlocks([]crypto.Signature{s1, s2}, ch)

	require.True(t, e.hasNext())

	// first we add second bytes
	require.NoError(t, e.add(s2.Bytes()))
	require.True(t, e.hasNext())

	select {
	case <-ch:
		t.Fatal("received unexpected block")
	default:
	}

	// then add first
	require.NoError(t, e.add(s1.Bytes()))
	require.False(t, e.hasNext(), "we received all expected messages, no more should arrive")

	select {
	case rs := <-ch:
		require.Equal(t, s1.Bytes(), []byte(rs))
	default:
		t.Fatal("no block")
	}

	select {
	case rs := <-ch:
		require.Equal(t, s2.Bytes(), []byte(rs))
	default:
		t.Fatal("no block")
	}
}

type peerImpl struct {
}

func (a *peerImpl) ID() string {
	return "aaa"
}

func (a *peerImpl) SendMessage(message proto.Message) {}

type subscriberImpl struct {
	ch chan proto.Message
}

func (a subscriberImpl) Receive(p types.ID, responseMessage proto.Message) bool {
	panic("implement me: Receive")
}

func (a subscriberImpl) Subscribe(p types.ID, responseMessage proto.Message) (chan proto.Message, func(), error) {
	return a.ch, func() {}, nil
}

func TestPreloadSignatures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sig1 := crypto.MustSignatureFromBase58("3EQ3k2n8KtSVVSsNfYbGp2LLwTc45SYWEACcME9KjLKCZSSeuVbVtdxroVysAJRdpoP3tDpy9MNTJMj6TjZ4b4aV")
	sig2 := crypto.MustSignatureFromBase58("ygSR7JmuxSN86VWLeaCx3mu8VtgRkUdh29s5ANtTAW7Lu5mans3WcNWGGWGu1mMxu9cS1HMRNMnr3bV9nWPEPKE")
	incoming := make(chan crypto.Signature, 10)

	last := NewSignatures(sig1)
	ch := make(chan proto.Message, 10)
	subscribe := &subscriberImpl{ch: ch}

	ch <- &proto.SignaturesMessage{
		Signatures: []crypto.Signature{sig2},
	}
	go func() {
		<-time.After(500 * time.Millisecond)
		cancel()
	}()
	_ = PreloadSignatures(ctx, incoming, &peerImpl{}, last, subscribe)

	require.Equal(t, sig2, <-incoming)
}
