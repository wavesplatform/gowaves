package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type account struct {
	Address   proto.Address    `json:"address"`
	PublicKey crypto.PublicKey `json:"public_key"`
}

type SchedulerEmits interface {
	Emits() []scheduler.Emit
}

type App struct {
	hashedApiKey  crypto.Digest
	apiKeyEnabled bool
	scheduler     SchedulerEmits
	utx           types.UtxPool
	state         state.State
	peers         peer_manager.PeerManager
	sync          types.StateSync
	services      services.Services
}

func NewApp(apiKey string, scheduler SchedulerEmits, services services.Services) (*App, error) {
	digest, err := crypto.SecureHash([]byte(apiKey))
	if err != nil {
		return nil, err
	}

	return &App{
		hashedApiKey:  digest,
		apiKeyEnabled: len(apiKey) > 0,
		state:         services.State,
		scheduler:     scheduler,
		utx:           services.UtxPool,
		peers:         services.Peers,
		services:      services,
	}, nil
}

func (a *App) TransactionsBroadcast(ctx context.Context, b []byte) error {
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

	respCh := make(chan error, 1)

	select {
	case a.services.InternalChannel <- messages.NewBroadcastTransaction(respCh, realType):
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "failed to send internal")
	}

	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "ctx cancelled from client")
	case <-time.After(5 * time.Second):
		return errors.New("timeout waiting response from internal")
	case err := <-respCh:
		return err
	}
}

func (a *App) LoadKeys(apiKey string, password []byte) error {
	err := a.checkAuth(apiKey)
	if err != nil {
		return err
	}
	return a.services.Wallet.Load(password)
}

func (a *App) Accounts() ([]account, error) {
	seeds := a.services.Wallet.Seeds()

	accounts := make([]account, 0, len(seeds))
	for _, seed := range seeds {
		_, pk, err := crypto.GenerateKeyPair(seed)
		if err != nil {
			return nil, err
		}
		addr, err := proto.NewAddressFromPublicKey(a.services.Scheme, pk)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account{Address: addr, PublicKey: pk})
	}
	return accounts, nil
}

func (a *App) checkAuth(key string) error {
	if !a.apiKeyEnabled {
		return &AuthError{errors.New("api key disabled")}
	}
	d, err := crypto.SecureHash([]byte(key))
	if err != nil {
		// TODO(nickeskov): it's OK?
		return err
	}
	if d != a.hashedApiKey {
		return &AuthError{errors.New("invalid api key")}
	}
	return nil
}
