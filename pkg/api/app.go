package api

import (
	"context"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type Node interface {
	State() state.State
	SpawnOutgoingConnection(ctx context.Context, addr proto.TCPAddr) error
	PeerManager() node.PeerManager
}

type App struct {
	hashedApiKey crypto.Digest
	node         Node
}

func NewApp(apiKey string, node Node) (*App, error) {
	digest, err := crypto.SecureHash([]byte(apiKey))
	if err != nil {
		return nil, err
	}

	return &App{
		hashedApiKey: digest,
		node:         node,
	}, nil
}

func (a *App) checkAuth(key string) error {
	d, err := crypto.SecureHash([]byte(key))
	if err != nil {
		return &AuthError{err}
	}

	if d != a.hashedApiKey {
		return &AuthError{errors.New("invalid api key")}
	}

	return nil
}
