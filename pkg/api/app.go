package api

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type SchedulerEmits interface {
	Emits() []scheduler.Emit
}

type App struct {
	hashedApiKey crypto.Digest
	scheduler    SchedulerEmits
	utx          types.UtxPool
	state        state.State
	peers        peer_manager.PeerManager
	sync         types.StateSync
	services     services.Services
}

func NewApp(apiKey string, scheduler SchedulerEmits, sync types.StateSync, services services.Services) (*App, error) {
	digest, err := crypto.SecureHash([]byte(apiKey))
	if err != nil {
		return nil, err
	}

	return &App{
		hashedApiKey: digest,
		state:        services.State,
		scheduler:    scheduler,
		utx:          services.UtxPool,
		peers:        services.Peers,
		sync:         sync,
		services:     services,
	}, nil
}

func (a *App) TransactionsBroadcast(b []byte) error {
	tt := proto.TransactionTypeVersion{}
	err := json.Unmarshal(b, &tt)
	if err != nil {
		return &BadRequestError{err}
	}

	realType, err := proto.GuessTransactionType(&tt)
	if err != nil {
		return &BadRequestError{err}
	}

	err = json.Unmarshal(b, realType)
	if err != nil {
		return &BadRequestError{err}
	}

	bts, err := realType.MarshalBinary()
	if err != nil {
		return &BadRequestError{err}
	}
	return a.utx.AddWithBytes(realType, bts)
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
