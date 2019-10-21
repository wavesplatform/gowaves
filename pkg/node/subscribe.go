package node

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
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

//Receive tries to apply block to any listener, if no one accepted return `false`, otherwise `true`
func (a *Subscribe) Receive(id string, responseMessage proto.Message) bool {
	a.mu.Lock()
	name := name(id, responseMessage)
	if ch, ok := a.running[name]; ok {
		a.mu.Unlock()
		select {
		case ch <- responseMessage:
		default:
			zap.S().Info("Subscribe.Receive ch is full")
		}

		return true
	}
	a.mu.Unlock()
	return false
}

type id interface {
	ID() string
}

// non thread safe
func (a *Subscribe) add(p id, responseMessage proto.Message) (chan proto.Message, func()) {

	name := name(p.ID(), responseMessage)

	unsubscribe := func() {
		a.mu.Lock()
		delete(a.running, name)
		a.mu.Unlock()
	}

	ch := make(chan proto.Message, 150)
	if _, ok := a.running[name]; ok {
		panic("multiple subscribe on " + name)
	}
	a.running[name] = ch
	return ch, unsubscribe
}

func (a *Subscribe) Exists(id string, responseMessage proto.Message) bool {
	name := name(id, responseMessage)
	a.mu.Lock()
	_, ok := a.running[name]
	a.mu.Unlock()
	return ok
}

func (a *Subscribe) Subscribe(p id, responseMessage proto.Message) (chan proto.Message, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.add(p, responseMessage)
}

func name(id string, responseMessage proto.Message) string {
	return fmt.Sprintf("%s-%s", id, reflect.TypeOf(responseMessage).String())
}
