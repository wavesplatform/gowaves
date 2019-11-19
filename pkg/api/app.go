package api

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
}

func NewApp(apiKey string, state state.State, peers peer_manager.PeerManager, scheduler SchedulerEmits, utx *utxpool.UtxImpl, sync types.StateSync) (*App, error) {
	digest, err := crypto.SecureHash([]byte(apiKey))
	if err != nil {
		return nil, err
	}

	return &App{
		hashedApiKey: digest,
		state:        state,
		scheduler:    scheduler,
		utx:          utx,
		peers:        peers,
		sync:         sync,
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
	a.utx.AddWithBytes(realType, bts)
	return nil
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
