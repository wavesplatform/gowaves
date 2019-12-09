package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSubscribe(t *testing.T) {
	service := NewSubscribeService()
	m := &proto.GetSignaturesMessage{}
	p := mock.NewPeer()

	if service.Exists(p.ID(), m) {
		t.Error("no subscribes should exists right now")
	}

	ch, cancel, err := service.Subscribe(p, m)
	require.NoError(t, err)
	if !service.Exists(p.ID(), m) {
		t.Error("we subscribed on event, should exists")
	}

	service.Receive(p, &proto.GetSignaturesMessage{})

	if !assert.IsType(t, &proto.GetSignaturesMessage{}, <-ch) {
		t.Error("we should receive message")
	}

	cancel()
	if service.Exists(p.ID(), m) {
		t.Error("after unsubscribe no service should exists")
	}
}
