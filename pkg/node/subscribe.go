package node

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
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
func (a *Subscribe) Receive(p types.ID, responseMessage proto.Message) bool {
	a.mu.Lock()
	name := name(p.ID(), responseMessage)
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

// non thread safe
func (a *Subscribe) add(p types.ID, responseMessage proto.Message) (chan proto.Message, func(), error) {

	name := name(p.ID(), responseMessage)

	unsubscribe := func() {
		a.mu.Lock()
		delete(a.running, name)
		a.mu.Unlock()
	}

	ch := make(chan proto.Message, 150)
	if _, ok := a.running[name]; ok {
		return nil, nil, errors.Errorf("multiple subscribe on %s", name)
	}
	a.running[name] = ch
	return ch, unsubscribe, nil
}

func (a *Subscribe) Exists(id string, responseMessage proto.Message) bool {
	name := name(id, responseMessage)
	a.mu.Lock()
	_, ok := a.running[name]
	a.mu.Unlock()
	return ok
}

func (a *Subscribe) Subscribe(p types.ID, responseMessage proto.Message) (chan proto.Message, func(), error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.add(p, responseMessage)
}

func name(id string, responseMessage proto.Message) string {
	return fmt.Sprintf("%s-%s", id, reflect.TypeOf(responseMessage).String())
}
