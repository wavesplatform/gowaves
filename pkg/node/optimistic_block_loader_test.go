package node

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/nullable"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func TestExpectedBlocks(t *testing.T) {

	s1, _ := crypto.NewSignatureFromBase58("22gRwjusnFYDoS31hRFEpFq21FjPCca2bUYtwicUH41GwzVkEAv7G22pAbRisu5s3bbhpzRRUpwF5png6ooKkb1n")
	id1 := proto.NewBlockIDFromSignature(s1)
	s2, _ := crypto.NewSignatureFromBase58("3YzRwee4k7ddfXK9FtMtZs9V4r8sxThVLUAF6ATfz1Efrxv29CjoHnw2oCz8uvjFhgPMgrsKMmgSyVZ3nw5Hswme")
	id2 := proto.NewBlockIDFromSignature(s2)

	ch := make(chan blockBytes, 2)
	e := newExpectedBlocks([]proto.BlockID{id1, id2}, ch)

	require.True(t, e.hasNext())

	// first we add second bytes
	require.NoError(t, e.add(id2, blockBytes{s2.Bytes(), false}))
	require.True(t, e.hasNext())

	select {
	case <-ch:
		t.Fatal("received unexpected block")
	default:
	}

	// then add first
	require.NoError(t, e.add(id1, blockBytes{s1.Bytes(), false}))
	require.False(t, e.hasNext(), "we received all expected messages, no more should arrive")

	select {
	case rs := <-ch:
		require.Equal(t, s1.Bytes(), rs.bytes)
	default:
		t.Fatal("no block")
	}

	select {
	case rs := <-ch:
		require.Equal(t, s2.Bytes(), rs.bytes)
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

func (a *peerImpl) Handshake() proto.Handshake {
	return proto.Handshake{}
}

type subscriberImpl struct {
	ch  chan proto.Message
	ch2 chan proto.Message
}

func (a subscriberImpl) Receive(p types.ID, responseMessage proto.Message) bool {
	panic("implement me: Receive")
}

func (a subscriberImpl) Subscribe(p types.ID, responseMessage proto.Message) (chan proto.Message, func(), error) {
	switch responseMessage.(type) {
	case *proto.SignaturesMessage:
		return a.ch, func() {}, nil
	case *proto.BlockIdsMessage:
		return a.ch2, func() {}, nil
	default:
		return nil, func() {}, errors.New("bad message type")
	}
}

func TestPreloadBlockIds_Signatures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sig1 := crypto.MustSignatureFromBase58("3EQ3k2n8KtSVVSsNfYbGp2LLwTc45SYWEACcME9KjLKCZSSeuVbVtdxroVysAJRdpoP3tDpy9MNTJMj6TjZ4b4aV")
	sig2 := crypto.MustSignatureFromBase58("ygSR7JmuxSN86VWLeaCx3mu8VtgRkUdh29s5ANtTAW7Lu5mans3WcNWGGWGu1mMxu9cS1HMRNMnr3bV9nWPEPKE")
	id2 := proto.NewBlockIDFromSignature(sig2)

	incoming := make(chan nullable.BlockID, 10)

	last := NewBlockIds(proto.NewBlockIDFromSignature(sig1))
	ch := make(chan proto.Message, 10)
	subscribe := &subscriberImpl{ch: ch}

	ch <- &proto.SignaturesMessage{
		Signatures: []crypto.Signature{sig2},
	}

	var wg sync.WaitGroup

	go func() {
		<-time.After(500 * time.Millisecond)
		cancel()
	}()
	_ = PreloadBlockIds(ctx, incoming, &peerImpl{}, last, subscribe, &wg)
	require.Equal(t, nullable.NewBlockID(id2), <-incoming)
	require.Equal(t, nullable.NewNullBlockID(), <-incoming)
	wg.Wait()
}

func TestPreloadBlockIds_Ids(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sig1 := crypto.MustSignatureFromBase58("3EQ3k2n8KtSVVSsNfYbGp2LLwTc45SYWEACcME9KjLKCZSSeuVbVtdxroVysAJRdpoP3tDpy9MNTJMj6TjZ4b4aV")
	sig2 := crypto.MustSignatureFromBase58("ygSR7JmuxSN86VWLeaCx3mu8VtgRkUdh29s5ANtTAW7Lu5mans3WcNWGGWGu1mMxu9cS1HMRNMnr3bV9nWPEPKE")
	id2 := proto.NewBlockIDFromSignature(sig2)

	incoming := make(chan nullable.BlockID, 10)

	last := NewBlockIds(proto.NewBlockIDFromSignature(sig1))
	ch := make(chan proto.Message, 10)
	subscribe := &subscriberImpl{ch: nil, ch2: ch}

	ch <- &proto.BlockIdsMessage{
		Blocks: []proto.BlockID{id2},
	}

	var wg sync.WaitGroup

	go func() {
		<-time.After(500 * time.Millisecond)
		cancel()
	}()
	_ = PreloadBlockIds(ctx, incoming, &peerImpl{}, last, subscribe, &wg)
	require.Equal(t, nullable.NewBlockID(id2), <-incoming)
	require.Equal(t, nullable.NewNullBlockID(), <-incoming)
	wg.Wait()
}
