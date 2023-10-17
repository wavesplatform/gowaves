package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	apiErrors "github.com/wavesplatform/gowaves/pkg/api/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type account struct {
	Address   proto.WavesAddress `json:"address"`
	PublicKey crypto.PublicKey   `json:"public_key"`
}

type SchedulerEmits interface {
	Emits() []scheduler.Emit
}

// default app settings
const (
	defaultBlockRequestLimit = 100
	defaultAssetDetailsLimit = 100
)

type appSettings struct {
	BlockRequestLimit uint64
	AssetDetailsLimit int
}

func defaultAppSettings() *appSettings {
	return &appSettings{
		BlockRequestLimit: defaultBlockRequestLimit,
		AssetDetailsLimit: defaultAssetDetailsLimit,
	}
}

type App struct {
	hashedApiKey  crypto.Digest
	apiKeyEnabled bool
	scheduler     SchedulerEmits
	utx           types.UtxPool
	state         state.State
	time          types.Time
	peers         peers.PeerManager
	scheme        proto.Scheme
	wallet        types.EmbeddedWallet
	settings      *appSettings
	broadcastCh   chan<- *messages.BroadcastTransaction
}

func NewApp(
	apiKey string, scheduler SchedulerEmits,
	st state.State,
	tm types.Time,
	utx types.UtxPool,
	wlt types.EmbeddedWallet,
	peers peers.PeerManager,
	scheme proto.Scheme,
) (*App, <-chan *messages.BroadcastTransaction, error) {
	return newApp(apiKey, scheduler, st, tm, utx, wlt, peers, scheme, nil)
}

func newApp(
	apiKey string,
	scheduler SchedulerEmits,
	st state.State,
	tm types.Time,
	utx types.UtxPool,
	wlt types.EmbeddedWallet,
	peers peers.PeerManager,
	scheme proto.Scheme,
	settings *appSettings,
) (*App, <-chan *messages.BroadcastTransaction, error) {
	if settings == nil {
		settings = defaultAppSettings()
	}
	digest, err := crypto.SecureHash([]byte(apiKey))
	if err != nil {
		return nil, nil, err
	}

	broadcastCh := make(chan *messages.BroadcastTransaction)
	return &App{
		hashedApiKey:  digest,
		apiKeyEnabled: len(apiKey) > 0,
		state:         st,
		time:          tm,
		scheduler:     scheduler,
		utx:           utx,
		wallet:        wlt,
		peers:         peers,
		scheme:        scheme,
		settings:      settings,
		broadcastCh:   broadcastCh,
	}, broadcastCh, nil
}

func (a *App) Close() {
	close(a.broadcastCh)
}

func (a *App) BroadcastChannel() chan<- *messages.BroadcastTransaction {
	return a.broadcastCh
}

func (a *App) State() state.State {
	return a.state
}

func (a *App) Time() types.Time {
	return a.time
}

func (a *App) Scheme() proto.Scheme {
	return a.scheme
}

func (a *App) UtxPool() types.UtxPool {
	return a.utx
}

func (a *App) Wallet() types.EmbeddedWallet {
	return a.wallet
}

func (a *App) TransactionsBroadcast(ctx context.Context, b []byte) (proto.Transaction, error) {
	tt := proto.TransactionTypeVersion{}
	err := json.Unmarshal(b, &tt)
	if err != nil {
		return nil, apiErrors.NewBadTransactionError(err)
	}

	realType, err := proto.GuessTransactionType(&tt)
	if err != nil {
		return nil, apiErrors.NewBadTransactionError(err)
	}

	err = proto.UnmarshalTransactionFromJSON(b, a.scheme, realType)
	if err != nil {
		return nil, apiErrors.NewBadTransactionError(err)
	}

	respCh := make(chan error, 1)

	select {
	case a.broadcastCh <- messages.NewBroadcastTransaction(respCh, realType):
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "failed to send internal")
	}
	var (
		delay = time.NewTimer(5 * time.Second)
		fired bool
	)
	defer func() {
		if !delay.Stop() && !fired {
			<-delay.C
		}
	}()
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "ctx cancelled from client")
	case <-delay.C:
		fired = true
		return nil, errors.New("timeout waiting response from internal")
	case err := <-respCh:
		if err != nil {
			return nil, err
		}
		return realType, nil
	}
}

func (a *App) LoadKeys(apiKey string, password []byte) error {
	err := a.checkAuth(apiKey)
	if err != nil {
		return err
	}
	return a.wallet.Load(password)
}

func (a *App) Accounts() ([]account, error) {
	seeds := a.wallet.AccountSeeds()

	accounts := make([]account, 0, len(seeds))
	for _, seed := range seeds {
		_, pk, err := crypto.GenerateKeyPair(seed)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate key pair for seed")
		}
		addr, err := proto.NewAddressFromPublicKey(a.scheme, pk)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate new address from public key")
		}
		accounts = append(accounts, account{Address: addr, PublicKey: pk})
	}
	return accounts, nil
}

func (a *App) checkAuth(key string) error {
	if !a.apiKeyEnabled {
		return apiErrors.ErrAPIKeyDisabled
	}
	d, err := crypto.SecureHash([]byte(key))
	if err != nil {
		return errors.Wrap(err, "failed to calculate secure hash for API key")
	}
	if d != a.hashedApiKey {
		return apiErrors.ErrAPIKeyNotValid
	}
	return nil
}
