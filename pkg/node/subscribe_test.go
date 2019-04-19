package node

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestSubscribe(t *testing.T) {
	service := NewSubscribeService()
	m := &proto.GetSignaturesMessage{}
	p := mock.NewPeer()

	if service.Exists(p.ID(), m) {
		t.Error("no subscribes should exists right now")
	}

	ch, cancel := service.Subscribe(p, m)
	if !service.Exists(p.ID(), m) {
		t.Error("we subscribed on event, should exists")
	}

	service.Receive(p.ID(), &proto.GetSignaturesMessage{})

	if !assert.IsType(t, &proto.GetSignaturesMessage{}, <-ch) {
		t.Error("we should receive message")
	}

	cancel()
	if service.Exists(p.ID(), m) {
		t.Error("after unsubscribe no service should exists")
	}
}
