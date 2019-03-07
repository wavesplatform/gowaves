package node

import (
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"reflect"
	"sync"
	"time"
)

type Subscribe struct {
	mu      sync.Mutex
	running map[string]chan proto.Message
}

func NewSubscribeService() *Subscribe {
	return &Subscribe{
		running: make(map[string]chan proto.Message),
	}
}

//func (a *Subscribe) Subscribe(p peer.Peer, responseMessage proto.Message) *Ask {
//	a.mu.Lock()
//	defer a.mu.Unlock()
//	name := fmt.Sprintf("%s-%s", p.ID(), reflect.TypeOf(responseMessage).String())
//	if p, ok := a.running[name]; ok {
//		return p
//	} else {
//		ask := NewAsk(responseMessage)
//		a.running[name] = ask
//		return ask
//	}
//}

func (a *Subscribe) Receive(p peer.Peer, responseMessage proto.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()

	name := name(p, responseMessage)
	if ch, ok := a.running[name]; ok {
		ch <- responseMessage
	}
}

func (a *Subscribe) add(p peer.Peer, responseMessage proto.Message) (chan proto.Message, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	name := name(p, responseMessage) //fmt.Sprintf("%s-%s", p.ID(), reflect.TypeOf(responseMessage).String())

	unsubscribe := func() {
		a.mu.Lock()
		delete(a.running, name)
		a.mu.Unlock()
	}

	ch := make(chan proto.Message, 10)
	a.running[name] = ch
	return ch, unsubscribe
}

func (a *Subscribe) Exists(p peer.Peer, responseMessage proto.Message) bool {
	name := name(p, responseMessage)
	a.mu.Lock()
	_, ok := a.running[name]
	a.mu.Unlock()
	return ok
}

func (a *Subscribe) Subscribe(p peer.Peer, responseMessage proto.Message) (chan proto.Message, func()) {
	return a.add(p, responseMessage)
}

func name(p peer.Peer, responseMessage proto.Message) string {
	return fmt.Sprintf("%s-%s", p.ID(), reflect.TypeOf(responseMessage).String())
}

const Waiting = 0
const Completed = 1
const Timeout = 2

type Ask struct {
	createdAt       time.Time
	timeout         time.Duration
	expect          proto.Message
	resolved        int
	resolvedMessage proto.Message
}

func NewAsk(responseMessage proto.Message) *Ask {
	return &Ask{
		expect:    responseMessage,
		createdAt: time.Now(),
	}
}

func (a *Ask) Completed() bool {
	return a.resolved > 0
}

func (a *Ask) Resolve(m proto.Message) {
	if reflect.TypeOf(a.expect) == reflect.TypeOf(m) {
		a.resolved = Completed
		a.resolvedMessage = m
	}
}

func (a *Ask) Timeout() bool {
	return a.resolved == Timeout
}

func (a *Ask) Wait(timeout time.Duration) {

}

//func (a *Subscribe) Reject() {
//
//}

//func (a *Subscribe) Timeout(timeout time.Duration) *Subscribe {
//	a.timeout = timeout
//	return a
//}

//
//func (a *Subscribe) Expect(m proto.Message) *Subscribe {
//	a.expect = m
//	return a
//}

//func (a *Subscribe) Execute() *Future {
//	return &Future{}
//}

//type Future struct {
//}
//
//func (a *Future) Completed() bool {
//
//}
//
//func (a *Future) Err() error {
//
//}
